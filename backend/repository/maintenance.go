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
	err := scanNullable(r.DB.QueryRowContext(ctx, `SELECT NVL(e.id_tip_ugovora,0),e.tip_version,TO_CHAR(s.pocetak_ug,'YYYY-MM-DD'),TO_CHAR(s.kraj_ug,'YYYY-MM-DD') FROM TED.UGO_EVID e JOIN TED.SAP_UGOVORI s ON s.id=e.id_sap_ugovor WHERE e.id=:id`, sql.Named("id", id)), &m.TypeID, &m.Version, &m.SAPStart, &m.SAPEnd)
	if err != nil {
		return nil, err
	}
	rows, err := r.DB.QueryContext(ctx, `SELECT id,sifra,naziv FROM TED.UGO_RAZ_VRSTA_RADA WHERE status='A' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	m.WorkTypes = []models.WorkType{}
	for rows.Next() {
		var w models.WorkType
		if err = rows.Scan(&w.ID, &w.Code, &w.Name); err != nil {
			rows.Close()
			return nil, err
		}
		m.WorkTypes = append(m.WorkTypes, w)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	d := &models.Maintenance{}
	var months sql.NullInt64
	var development int
	err = scanNullable(r.DB.QueryRowContext(ctx, `SELECT id,TO_CHAR(datum_potpisa,'YYYY-MM-DD'),TO_CHAR(datum_kraja,'YYYY-MM-DD'),ima_razvojnu_podrsku,teh_period_meseci,id_teh_tip_doc,teh_podsetnik_dana,raz_podsetnik_dana,TO_CHAR(pocetak_odrzavanja,'YYYY-MM-DD'),version FROM TED.UGO_ODRZAVANJE WHERE id_ugo_evid=:id`, sql.Named("id", id)), &d.ID, &d.Signed, &d.Ends, &development, &months, &d.DocumentType, &d.TechnicalReminder, &d.DevelopmentReminder, &d.Start, &d.Version)
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
	d.Funds = []models.DevelopmentFund{}
	rows, err = r.DB.QueryContext(ctx, `SELECT id_vrsta_rada,ugovoreno_casova FROM TED.UGO_RAZ_FOND WHERE id_odrzavanja=:id ORDER BY id_vrsta_rada`, sql.Named("id", d.ID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f models.DevelopmentFund
		if err = rows.Scan(&f.TypeID, &f.Hours); err != nil {
			return nil, err
		}
		d.Funds = append(d.Funds, f)
	}
	if err = rows.Err(); err != nil {
		return nil, err
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
	return r.saveMaintenance(ctx, id, m, org, user, false)
}
func (r *OracleStore) UpdateMaintenance(ctx context.Context, id int, m models.Maintenance, org, user int) error {
	return r.saveMaintenance(ctx, id, m, org, user, true)
}
func (r *OracleStore) saveMaintenance(ctx context.Context, id int, m models.Maintenance, org, user int, update bool) error {
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
	var start, end string
	err = scanNullable(tx.QueryRowContext(ctx, `SELECT TO_CHAR(s.pocetak_ug,'YYYY-MM-DD'),TO_CHAR(s.kraj_ug,'YYYY-MM-DD') FROM TED.SAP_UGOVORI s JOIN TED.UGO_EVID e ON e.id_sap_ugovor=s.id WHERE e.id=:id`, sql.Named("id", id)), &start, &end)
	if err != nil {
		return err
	}
	if m.Start != "" {
		start = m.Start
	}
	if start == "" || end == "" || start > end {
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
	seen := map[int]bool{}
	total := int64(0)
	for _, f := range m.Funds {
		if seen[f.TypeID] || f.Hours < 0 || f.Hours > 2147483647 {
			return ErrMaintenanceInput
		}
		seen[f.TypeID] = true
		total += int64(f.Hours)
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_RAZ_VRSTA_RADA WHERE id=:id AND status='A' AND sifra IN ('INZENJER','TEHNICAR','KONSULTANT')`, sql.Named("id", f.TypeID)).Scan(&count)
		if err != nil {
			return err
		}
		if count != 1 {
			return ErrMaintenanceInput
		}
	}
	if (m.Development && total == 0) || (!m.Development && total != 0) {
		return ErrMaintenanceInput
	}
	var months any
	if m.Months != nil {
		months = *m.Months
	}
	dev := 0
	if m.Development {
		dev = 1
	}
	args := []any{sql.Named("start_date", m.Start), sql.Named("development", dev), sql.Named("months", months), sql.Named("doc_type", m.DocumentType), sql.Named("tech_reminder", m.TechnicalReminder), sql.Named("dev_reminder", m.DevelopmentReminder), sql.Named("user_id", user), sql.Named("id", id)}
	maintenanceID := 0
	if update {
		if err = requireMaintenanceLease(ctx, tx, id, user, m.Token); err != nil {
			return err
		}
		var version int64
		err = tx.QueryRowContext(ctx, `SELECT id,version FROM TED.UGO_ODRZAVANJE WHERE id_ugo_evid=:id FOR UPDATE NOWAIT`, sql.Named("id", id)).Scan(&maintenanceID, &version)
		if err != nil {
			return err
		}
		if version != m.Version {
			return ErrMaintenanceConflict
		}
		_, err = tx.ExecContext(ctx, `UPDATE TED.UGO_ODRZAVANJE SET pocetak_odrzavanja=TO_DATE(:start_date,'YYYY-MM-DD'),ima_razvojnu_podrsku=:development,teh_period_meseci=:months,id_teh_tip_doc=:doc_type,teh_podsetnik_dana=:tech_reminder,raz_podsetnik_dana=:dev_reminder,id_korisnika_izmene=:user_id,datizm=SYSTIMESTAMP,version=version+1 WHERE id_ugo_evid=:id`, args...)
	} else {
		args = append(args, sql.Named("new_id", sql.Out{Dest: &maintenanceID}))
		_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_ODRZAVANJE (pocetak_odrzavanja,ima_razvojnu_podrsku,teh_period_meseci,id_teh_tip_doc,teh_podsetnik_dana,raz_podsetnik_dana,id_korisnika_unosa,id_ugo_evid) VALUES (TO_DATE(:start_date,'YYYY-MM-DD'),:development,:months,:doc_type,:tech_reminder,:dev_reminder,:user_id,:id) RETURNING id INTO :new_id`, args...)
	}
	if err != nil {
		if strings.Contains(err.Error(), "ORA-00001") {
			return ErrMaintenanceConflict
		}
		return err
	}
	// Preserve fund IDs for future request foreign keys. Missing entries become zero.
	_, err = tx.ExecContext(ctx, `UPDATE TED.UGO_RAZ_FOND SET ugovoreno_casova=0 WHERE id_odrzavanja=:id`, sql.Named("id", maintenanceID))
	if err != nil {
		return err
	}
	for _, f := range m.Funds {
		_, err = tx.ExecContext(ctx, `MERGE INTO TED.UGO_RAZ_FOND f USING (SELECT :maintenance_id id_odrzavanja,:work_id id_vrsta_rada,:hours ugovoreno_casova FROM DUAL) s ON (f.id_odrzavanja=s.id_odrzavanja AND f.id_vrsta_rada=s.id_vrsta_rada) WHEN MATCHED THEN UPDATE SET f.ugovoreno_casova=s.ugovoreno_casova WHEN NOT MATCHED THEN INSERT (id_odrzavanja,id_vrsta_rada,ugovoreno_casova) VALUES (s.id_odrzavanja,s.id_vrsta_rada,s.ugovoreno_casova)`, sql.Named("maintenance_id", maintenanceID), sql.Named("work_id", f.TypeID), sql.Named("hours", f.Hours))
		if err != nil {
			return err
		}
	}
	if update {
		if err = releaseMaintenanceLease(ctx, tx, id, user, m.Token); err != nil {
			return err
		}
	}
	return tx.Commit()
}
