package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
)

var ErrContractPeople = errors.New("Angažovanja su promenjena ili lice/uloga nisu dostupni. Osvežite podatke i ponovite izbor.")

type ContractPeopleStore interface {
	ListContractPeople(context.Context, int) ([]models.ContractPerson, error)
	ContractPeopleCandidates(context.Context, int) ([]models.ContractPerson, error)
	ChangeContractPerson(context.Context, int, int, models.ContractPersonChange, bool) error
	ContactEngagements(context.Context, int) ([]models.ContactEngagement, error)
}

func (r *OracleStore) ListContractPeople(ctx context.Context, id int) ([]models.ContractPerson, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT a.id,a.id_ugo_dob_lica,a.id_ugo_dob_lica_rola,r.naziv,l.ime,l.telefon,l.email,a.status,l.status,a.version
 FROM TED.UGO_EVID_LICA a JOIN TED.UGO_DOB_LICA l ON l.id=a.id_ugo_dob_lica
 JOIN TED.UGO_DOB_LICA_ROLE r ON r.id=a.id_ugo_dob_lica_rola WHERE a.id_ugo_evid=:id ORDER BY a.status,a.id_ugo_dob_lica_rola,a.id`, sql.Named("id", id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ContractPerson{}
	for rows.Next() {
		var p models.ContractPerson
		if err = scanNullable(rows, &p.ID, &p.PersonID, &p.RoleID, &p.Role, &p.Name, &p.Phone, &p.Email, &p.Status, &p.PersonStatus, &p.Version); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *OracleStore) ContractPeopleCandidates(ctx context.Context, id int) ([]models.ContractPerson, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT l.id,l.ime,l.telefon,l.email FROM TED.UGO_DOB_LICA l
 JOIN TED.SAP_UGOVORI s ON s.id_sap_dobavljac=l.id_sap_dobavljac JOIN TED.UGO_EVID e ON e.id_sap_ugovor=s.id
 WHERE e.id=:id AND l.status='A' ORDER BY l.ime,l.id`, sql.Named("id", id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ContractPerson{}
	for rows.Next() {
		var p models.ContractPerson
		if err = scanNullable(rows, &p.PersonID, &p.Name, &p.Phone, &p.Email); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Short database row locks serialize writes across servers. Expected link IDs and
// versions reject stale browser decisions, including replacement of a changed SLM.
func (r *OracleStore) ChangeContractPerson(ctx context.Context, contract, user int, p models.ContractPersonChange, remove bool) (err error) {
	if user < 1 || contract < 1 || (remove && (p.ID < 1 || p.Version < 1)) || (!remove && (p.PersonID < 1 || p.RoleID < 1)) {
		return ErrContractPeople
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	defer func() {
		if err != nil && (strings.Contains(err.Error(), "ORA-00054") || strings.Contains(err.Error(), "ORA-00001")) {
			err = ErrContractPeople
		}
	}()
	var supplier int
	if !remove {
		var status string
		err = tx.QueryRowContext(ctx, `SELECT id_sap_dobavljac,status FROM TED.UGO_DOB_LICA WHERE id=:id FOR UPDATE NOWAIT`, sql.Named("id", p.PersonID)).Scan(&supplier, &status)
		if err != nil {
			return err
		}
		if status != "A" {
			return ErrContractPeople
		}
		var editing int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EDIT_LOCK WHERE entity_type='UGO_DOB_LICA' AND entity_id=:id AND expires_at>SYSTIMESTAMP`, sql.Named("id", p.PersonID)).Scan(&editing); err != nil {
			return err
		}
		if editing > 0 {
			return ErrContactLocked
		}
	}
	var org, sap int
	if err = tx.QueryRowContext(ctx, `SELECT id_ugo_org,id_sap_ugovor FROM TED.UGO_EVID WHERE id=:id FOR UPDATE NOWAIT`, sql.Named("id", contract)).Scan(&org, &sap); err != nil {
		return err
	}
	if a := models.AccessFrom(ctx); a != nil && !a.Allows(org, true) {
		return ErrContractUnavailable
	}
	if err = openDocumentContract(ctx, tx, contract); err != nil {
		return err
	}
	if remove {
		result, e := tx.ExecContext(ctx, `UPDATE TED.UGO_EVID_LICA SET status='P',version=version+1,datizm=SYSTIMESTAMP,id_korisnika=:actor WHERE id=:id AND id_ugo_evid=:contract AND version=:version AND status='A'`, sql.Named("actor", user), sql.Named("id", p.ID), sql.Named("contract", contract), sql.Named("version", p.Version))
		if e != nil {
			return e
		}
		n, e := result.RowsAffected()
		if e != nil {
			return e
		}
		if n != 1 {
			return ErrContractPeople
		}
	} else {
		var valid int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.SAP_UGOVORI WHERE id=:sap AND id_sap_dobavljac=:supplier`, sql.Named("sap", sap), sql.Named("supplier", supplier)).Scan(&valid); err != nil {
			return err
		}
		if valid != 1 {
			return ErrContractPeople
		}
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_DOB_LICA_ROLE WHERE id=:role`, sql.Named("role", p.RoleID)).Scan(&valid); err != nil {
			return err
		}
		if valid != 1 {
			return ErrContractPeople
		}
		if p.RoleID == 1 {
			var current int
			var version int64
			err = tx.QueryRowContext(ctx, `SELECT id,version FROM TED.UGO_EVID_LICA WHERE id_ugo_evid=:id AND id_ugo_dob_lica_rola=1 AND status='A'`, sql.Named("id", contract)).Scan(&current, &version)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if current != p.PreviousSLM || version != p.PreviousVersion {
				return ErrContractPeople
			}
			if current > 0 {
				if _, err = tx.ExecContext(ctx, `UPDATE TED.UGO_EVID_LICA SET status='P',version=version+1,datizm=SYSTIMESTAMP,id_korisnika=:actor WHERE id=:id`, sql.Named("actor", user), sql.Named("id", current)); err != nil {
					return err
				}
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_EVID_LICA(id_ugo_evid,id_ugo_dob_lica,id_ugo_dob_lica_rola,status,id_korisnika) VALUES(:contract,:person,:role,'A',:actor)`, sql.Named("contract", contract), sql.Named("person", p.PersonID), sql.Named("role", p.RoleID), sql.Named("actor", user))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *OracleStore) loadContractContacts(ctx context.Context, items []models.UgoEvidProsireni) error {
	ids := []string{}
	args := []any{}
	for i := range items {
		items[i].LicaDobavljaca = []models.UgoDobLiceKontakt{}
		if items[i].IDSapDobavljac == nil {
			continue
		}
		name := fmt.Sprintf("contract_%d", len(args))
		ids = append(ids, ":"+name)
		args = append(args, sql.Named(name, items[i].IDUgoEvid))
	}
	if len(args) == 0 {
		return nil
	}
	rows, err := r.DB.QueryContext(ctx, `SELECT a.id_ugo_evid,l.ime,l.radno_mesto,l.telefon,l.email,r.naziv,l.id,l.status
 FROM TED.UGO_EVID_LICA a JOIN TED.UGO_DOB_LICA l ON l.id=a.id_ugo_dob_lica
 JOIN TED.UGO_DOB_LICA_ROLE r ON r.id=a.id_ugo_dob_lica_rola
 WHERE a.status='A' AND a.id_ugo_evid IN (`+strings.Join(ids, ",")+`) ORDER BY r.id,l.id`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	byContract := map[int][]models.UgoDobLiceKontakt{}
	for rows.Next() {
		var id int
		var p models.UgoDobLiceKontakt
		if err = scanNullable(rows, &id, &p.Ime, &p.RadnoMesto, &p.Telefon, &p.Email, &p.RolaLica, &p.ID, &p.Status); err != nil {
			return err
		}
		byContract[id] = append(byContract[id], p)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for i := range items {
		if p := byContract[items[i].IDUgoEvid]; p != nil {
			items[i].LicaDobavljaca = p
		}
	}
	return nil
}

func (r *OracleStore) ContactEngagements(ctx context.Context, id int) ([]models.ContactEngagement, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT e.id,s.br_ugovor,s.predmet_ugovora,r.naziv FROM TED.UGO_EVID_LICA a
 JOIN TED.UGO_EVID e ON e.id=a.id_ugo_evid JOIN TED.SAP_UGOVORI s ON s.id=e.id_sap_ugovor
 JOIN TED.UGO_DOB_LICA_ROLE r ON r.id=a.id_ugo_dob_lica_rola WHERE a.id_ugo_dob_lica=:id AND a.status='A' AND s.otvoren_ug='X' AND `+accessPredicate(ctx, "e.id_ugo_org", "1=0")+` ORDER BY e.id`, sql.Named("id", id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ContactEngagement{}
	for rows.Next() {
		var p models.ContactEngagement
		if err = scanNullable(rows, &p.ContractID, &p.Number, &p.Subject, &p.Role); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
