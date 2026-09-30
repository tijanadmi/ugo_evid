package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/tijanadmi/ugo_evid/models"
)

var (
	ErrContractRegistered      = errors.New("Ugovor je već evidentiran. Osvežite izbor ugovora.")
	ErrContractContacts        = errors.New("Dobavljač ima više aktivnih lica sa rolom 1. Ispravite podatke pre unosa ugovora.")
	ErrContractContactChanged  = errors.New("Kontakt lice je promenjeno. Ponovo učitajte podatke izabranog ugovora.")
	ErrContractContactRequired = errors.New("Unesite ime, telefon i email novog kontakt lica.")
	ErrContractSupplier        = errors.New("Izabrani ugovor nema ispravnog dobavljača.")
	ErrRegistrationBusy        = errors.New("Unos za ovaj ugovor ili dobavljača je u toku. Pokušajte ponovo.")
)

const choiceColumns = `u.id, u.id_sap_dobavljac, u.br_ugovor, u.godina, u.predmet_ugovora, d.naziv, u.otvoren_ug`
const choiceFrom = ` FROM TED.SAP_UGOVORI u LEFT JOIN TED.SAP_DOBAVLJACI d ON d.id = u.id_sap_dobavljac`

func scanChoice(row interface{ Scan(...any) error }, m *models.ContractChoice) error {
	return scanNullable(row, &m.ID, &m.SupplierID, &m.Number, &m.Year, &m.Subject, &m.Supplier, &m.Open)
}

func (r *OracleStore) GetUnregisteredContracts(ctx context.Context, offset, limit int, filter string) ([]models.ContractChoice, int, error) {
	if offset < 0 || limit < 1 || limit > 100 {
		return nil, 0, errors.New("invalid pagination")
	}
	where := ` WHERE NOT EXISTS (SELECT 1 FROM TED.UGO_EVID e WHERE e.id_sap_ugovor = u.id)`
	args := []any{}
	if filter = strings.TrimSpace(filter); filter != "" {
		where += ` AND (INSTR(LOWER(u.br_ugovor), LOWER(:filter_number)) > 0 OR INSTR(LOWER(u.predmet_ugovora), LOWER(:filter_subject)) > 0 OR INSTR(LOWER(d.naziv), LOWER(:filter_supplier)) > 0)`
		args = append(args, sql.Named("filter_number", filter), sql.Named("filter_subject", filter), sql.Named("filter_supplier", filter))
	}
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*)`+choiceFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, sql.Named("offset", offset), sql.Named("limit", limit))
	rows, err := r.DB.QueryContext(ctx, `SELECT `+choiceColumns+choiceFrom+where+` ORDER BY u.id DESC OFFSET :offset ROWS FETCH NEXT :limit ROWS ONLY`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []models.ContractChoice{}
	for rows.Next() {
		var m models.ContractChoice
		if err := scanChoice(rows, &m); err != nil {
			return nil, 0, err
		}
		items = append(items, m)
	}
	return items, total, rows.Err()
}

type contractQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func registrationContract(ctx context.Context, q contractQueryer, id int) (models.ContractChoice, error) {
	var m models.ContractChoice
	err := scanChoice(q.QueryRowContext(ctx, `SELECT `+choiceColumns+choiceFrom+` WHERE u.id = :id`, sql.Named("id", id)), &m)
	if err != nil {
		return m, err
	}
	var exists int
	if err = q.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.SAP_DOBAVLJACI WHERE id = :supplier`, sql.Named("supplier", m.SupplierID)).Scan(&exists); err != nil {
		return m, err
	}
	if m.SupplierID < 1 || exists != 1 {
		return m, ErrContractSupplier
	}
	if err = q.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EVID WHERE id_sap_ugovor = :id`, sql.Named("id", id)).Scan(&exists); err != nil {
		return m, err
	}
	if exists > 0 {
		return m, ErrContractRegistered
	}
	return m, nil
}

func registrationContacts(ctx context.Context, q contractQueryer, supplier int) ([]models.UgoDobLice, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, ime, radno_mesto, telefon, email, version FROM TED.UGO_DOB_LICA
 WHERE id_sap_dobavljac = :supplier AND id_ugo_dob_lica_rola = 1 AND status = 'A' ORDER BY id`, sql.Named("supplier", supplier))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []models.UgoDobLice{}
	for rows.Next() {
		var m models.UgoDobLice
		if err = scanNullable(rows, &m.ID, &m.Ime, &m.RadnoMesto, &m.Telefon, &m.Email, &m.Version); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(list) > 1 {
		return nil, ErrContractContacts
	}
	return list, nil
}

func (r *OracleStore) PrepareContractRegistration(ctx context.Context, id int) (*models.ContractPreparation, error) {
	contract, err := registrationContract(ctx, r.DB, id)
	if err != nil {
		return nil, err
	}
	contacts, err := registrationContacts(ctx, r.DB, contract.SupplierID)
	if err != nil {
		return nil, err
	}
	result := &models.ContractPreparation{Contract: contract}
	if len(contacts) == 1 {
		result.Contact = &contacts[0]
	}
	return result, nil
}

// The unique (ENTITY_TYPE, ENTITY_ID) key serializes insertions even when the
// business row does not exist yet. These rows are inserted AND removed in the
// same short transaction: rollback/crash releases them; no browser lease exists.
func registrationMutex(ctx context.Context, tx *sql.Tx, entity string, id, userID int, token string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO TED.UGO_EDIT_LOCK (entity_type,entity_id,user_id,lock_token,acquired_at,renewed_at,expires_at)
 VALUES (:entity,:id,:user_id,:token,SYSTIMESTAMP,SYSTIMESTAMP,SYSTIMESTAMP + INTERVAL '2' MINUTE)`, sql.Named("entity", entity), sql.Named("id", id), sql.Named("user_id", userID), sql.Named("token", token))
	if err != nil && strings.Contains(err.Error(), "ORA-00001") {
		return ErrRegistrationBusy
	}
	return err
}

func (r *OracleStore) RegisterContract(ctx context.Context, req models.RegisterContract, orgID, userID int) (*models.UgoEvid, error) {
	if req.ContractID < 1 || orgID < 1 || userID < 1 {
		return nil, ErrUserOrganization
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	random := make([]byte, 32)
	if _, err = rand.Read(random); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(random)
	if err = registrationMutex(ctx, tx, "UGO_EVID_CREATE", req.ContractID, userID, token); err != nil {
		return nil, err
	}
	contract, err := registrationContract(ctx, tx, req.ContractID)
	if err != nil {
		return nil, err
	}
	// Different contracts from the same supplier must not both create the first SLM.
	random = make([]byte, 32)
	if _, err = rand.Read(random); err != nil {
		return nil, err
	}
	supplierToken := hex.EncodeToString(random)
	if err = registrationMutex(ctx, tx, "UGO_SLM_CREATE", contract.SupplierID, userID, supplierToken); err != nil {
		return nil, err
	}
	contacts, err := registrationContacts(ctx, tx, contract.SupplierID)
	if err != nil {
		return nil, err
	}
	var person models.UgoDobLice
	if len(contacts) == 0 {
		if req.ContactID != 0 || req.ContactVersion != 0 {
			return nil, ErrContractContactChanged
		}
		if req.Contact == nil {
			return nil, ErrContractContactRequired
		}
		person.Ime, person.RadnoMesto, person.Telefon, person.Email = req.Contact.Ime, req.Contact.RadnoMesto, req.Contact.Telefon, req.Contact.Email
		_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_DOB_LICA (id_sap_dobavljac,ime,radno_mesto,telefon,email,id_ugo_dob_lica_rola,status,datpri,datzm)
 VALUES (:supplier,:ime,:radno_mesto,:telefon,:email,1,'A',SYSDATE,SYSDATE) RETURNING id INTO :new_id`,
			sql.Named("supplier", contract.SupplierID), sql.Named("ime", person.Ime), sql.Named("radno_mesto", person.RadnoMesto), sql.Named("telefon", person.Telefon), sql.Named("email", person.Email), sql.Named("new_id", sql.Out{Dest: &person.ID}))
		if err != nil {
			if strings.Contains(err.Error(), "ORA-00001") {
				return nil, ErrContractContactChanged
			}
			return nil, err
		}
	} else {
		person = contacts[0]
		if req.Contact != nil || req.ContactID != person.ID || req.ContactVersion != person.Version {
			return nil, ErrContractContactChanged
		}
		var version int64
		err = tx.QueryRowContext(ctx, `SELECT version FROM TED.UGO_DOB_LICA WHERE id = :id AND id_sap_dobavljac = :supplier AND id_ugo_dob_lica_rola = 1 AND status = 'A' FOR UPDATE NOWAIT`, sql.Named("id", person.ID), sql.Named("supplier", contract.SupplierID)).Scan(&version)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrContractContactChanged
		}
		if err != nil {
			if strings.Contains(err.Error(), "ORA-00054") {
				return nil, ErrContactLocked
			}
			return nil, err
		}
		if version != person.Version {
			return nil, ErrContractContactChanged
		}
		var editing int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM TED.UGO_EDIT_LOCK WHERE entity_type = 'UGO_DOB_LICA' AND entity_id = :id AND expires_at > SYSTIMESTAMP`, sql.Named("id", person.ID)).Scan(&editing); err != nil {
			return nil, err
		}
		if editing > 0 {
			return nil, ErrContactLocked
		}
	}
	if strings.TrimSpace(person.Ime) == "" {
		return nil, ErrContractContactRequired
	}
	result := &models.UgoEvid{SapUgovor: models.SapUgovor{ID: req.ContractID}, UgoOrg: models.UgoOrg{ID: orgID}, UgoDobLice: models.UgoDobLice{ID: person.ID}, Ime: person.Ime, Telefon: person.Telefon, Email: person.Email, Status: "A"}
	_, err = tx.ExecContext(ctx, `INSERT INTO TED.UGO_EVID (id_sap_ugovor,id_ugo_org,ime,telefon,email,status,datpri,id_ugo_dob_lica)
 VALUES (:contract,:org,:ime,:telefon,:email,'A',SYSDATE,:person) RETURNING id INTO :new_id`, sql.Named("contract", req.ContractID), sql.Named("org", orgID), sql.Named("ime", person.Ime), sql.Named("telefon", person.Telefon), sql.Named("email", person.Email), sql.Named("person", person.ID), sql.Named("new_id", sql.Out{Dest: &result.ID}))
	if err != nil {
		if strings.Contains(err.Error(), "ORA-00001") {
			return nil, ErrContractRegistered
		}
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM TED.UGO_EDIT_LOCK WHERE lock_token IN (:contract_token,:supplier_token) AND user_id = :user_id`, sql.Named("contract_token", token), sql.Named("supplier_token", supplierToken), sql.Named("user_id", userID))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}
