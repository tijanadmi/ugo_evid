package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
)

var ErrContractType = errors.New("Izaberite aktivan tip ugovora.")
var ErrMaintenanceConflict = errors.New("Podaci su promenjeni ili odrzavanje vec postoji. Osvezite podatke.")
var ErrMaintenanceInput = errors.New("Proverite tip ugovora, datume, period i tip dokumenta odrzavanja.")

func (r *OracleStore) ListContractTypes(ctx context.Context) ([]models.ContractType, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id,sifra,naziv FROM TED.UGO_TIP_UGOVORA WHERE status='A' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ContractType{}
	for rows.Next() {
		var m models.ContractType
		if err = rows.Scan(&m.ID, &m.Code, &m.Name); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (r *OracleStore) GetContractSettings(ctx context.Context, id int) (*models.ContractSettings, error) {
	m := &models.ContractSettings{}
	err := r.DB.QueryRowContext(ctx, `SELECT NVL(id_tip_ugovora,0),tip_version FROM TED.UGO_EVID WHERE id=:id`, sql.Named("id", id)).Scan(&m.TypeID, &m.Version)
	if err != nil {
		return nil, err
	}
	d := &models.Maintenance{}
	var months sql.NullInt64
	var development int
	err = r.DB.QueryRowContext(ctx, `SELECT id,TO_CHAR(datum_potpisa,'YYYY-MM-DD'),TO_CHAR(datum_kraja,'YYYY-MM-DD'),ima_razvojnu_podrsku,teh_period_meseci,id_teh_tip_doc,teh_podsetnik_dana,raz_podsetnik_dana FROM TED.UGO_ODRZAVANJE WHERE id_ugo_evid=:id`, sql.Named("id", id)).Scan(&d.ID, &d.Signed, &d.Ends, &development, &months, &d.DocumentType, &d.TechnicalReminder, &d.DevelopmentReminder)
	if errors.Is(err, sql.ErrNoRows) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	d.Development = development == 1
	if months.Valid {
		v := int(months.Int64)
		d.Months = &v
	}
	m.Maintenance = d
	return m, nil
}

// Serialize creation/type changes on the existing parent row, including the
// empty-maintenance case. Recheck organization and open status inside the tx.
func lockMaintenanceParent(ctx context.Context, tx *sql.Tx, id, org int) (int, int64, error) {
	var typeID, owner, sapID int
	var version int64
	err := tx.QueryRowContext(ctx, `SELECT NVL(id_tip_ugovora,0),tip_version,id_ugo_org,id_sap_ugovor FROM TED.UGO_EVID WHERE id=:id FOR UPDATE NOWAIT`, sql.Named("id", id)).Scan(&typeID, &version, &owner, &sapID)
	if err != nil {
		if strings.Contains(err.Error(), "ORA-00054") {
			return 0, 0, ErrRegistrationBusy
		}
		return 0, 0, err
	}
	if owner != org {
		return 0, 0, ErrUserOrganization
	}
	var open string
	err = tx.QueryRowContext(ctx, `SELECT NVL(otvoren_ug,'-') FROM TED.SAP_UGOVORI WHERE id=:id FOR UPDATE NOWAIT`, sql.Named("id", sapID)).Scan(&open)
	if err != nil {
		if strings.Contains(err.Error(), "ORA-00054") {
			return 0, 0, ErrRegistrationBusy
		}
		return 0, 0, err
	}
	if open != "X" {
		return 0, 0, ErrDocumentClosed
	}
	return typeID, version, nil
}
func activeContractType(ctx context.Context, tx *sql.Tx, id int) (string, error) {
	var code string
	err := tx.QueryRowContext(ctx, `SELECT sifra FROM TED.UGO_TIP_UGOVORA WHERE id=:id AND status='A'`, sql.Named("id", id)).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrContractType
	}
	return code, err
}
func (r *OracleStore) SetContractType(ctx context.Context, id, typeID int, expected int64, org, user int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	old, version, err := lockMaintenanceParent(ctx, tx, id, org)
	if err != nil {
		return err
	}
	if version != expected {
		return ErrMaintenanceConflict
	}
	if _, err = activeContractType(ctx, tx, typeID); err != nil {
		return err
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_ODRZAVANJE WHERE id_ugo_evid=:id`, sql.Named("id", id)).Scan(&count); err != nil {
		return err
	}
	if count > 0 && old != typeID {
		return ErrMaintenanceConflict
	}
	_, err = tx.ExecContext(ctx, `UPDATE TED.UGO_EVID SET id_tip_ugovora=:type_id,tip_version=tip_version+1,tip_id_korisnika=:user_id,tip_datizm=SYSTIMESTAMP WHERE id=:id`, sql.Named("type_id", typeID), sql.Named("user_id", user), sql.Named("id", id))
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *OracleStore) CreateMaintenance(ctx context.Context, id int, m models.Maintenance, org, user int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	typeID, _, err := lockMaintenanceParent(ctx, tx, id, org)
	if err != nil {
		return err
	}
	code, err := activeContractType(ctx, tx, typeID)
	if err != nil {
		return err
	}
	if code != "ODRZAVANJE" {
		return ErrMaintenanceInput
	}
	var count int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_TIP_DOC WHERE id=:id AND status='A' AND sifra IN ('POTVRDA_TEH','ZAPISNIK_TEH')`, sql.Named("id", m.DocumentType)).Scan(&count)
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrMaintenanceInput
	}
	var months any
	if m.Months != nil {
		months = *m.Months
	}
	development := 0
	if m.Development {
		development = 1
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_ODRZAVANJE (id_ugo_evid,datum_potpisa,datum_kraja,ima_razvojnu_podrsku,teh_period_meseci,id_teh_tip_doc,teh_podsetnik_dana,raz_podsetnik_dana,id_korisnika_unosa) VALUES (:id,TO_DATE(:signed,'YYYY-MM-DD'),TO_DATE(:ends,'YYYY-MM-DD'),:development,:months,:doc_type,:tech_reminder,:dev_reminder,:user_id)`, sql.Named("id", id), sql.Named("signed", m.Signed), sql.Named("ends", m.Ends), sql.Named("development", development), sql.Named("months", months), sql.Named("doc_type", m.DocumentType), sql.Named("tech_reminder", m.TechnicalReminder), sql.Named("dev_reminder", m.DevelopmentReminder), sql.Named("user_id", user))
	if err != nil {
		if strings.Contains(err.Error(), "ORA-00001") {
			return ErrMaintenanceConflict
		}
		return err
	}
	return tx.Commit()
}
