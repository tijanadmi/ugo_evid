package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/tijanadmi/ugo_evid/models"
)

func TestRegistrationTransaction(t *testing.T) {
	for _, tc := range []struct {
		name                                                            string
		newContact, duplicate, changed, editing, failEvidence, multiple bool
		want                                                            error
	}{
		{name: "existing contact"},
		{name: "new contact", newContact: true},
		{name: "already registered", duplicate: true, want: ErrContractRegistered},
		{name: "contact changed", changed: true, want: ErrContractContactChanged},
		{name: "contact being edited", editing: true, want: ErrContactLocked},
		{name: "multiple active SLM", multiple: true, want: ErrContractContacts},
		{name: "rollback both inserts", newContact: true, failEvidence: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			locks := []string{}
			writes := []string{}
			released := false
			store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
				switch {
				case strings.Contains(q, "u.br_ugovor"):
					if len(locks) != 1 {
						t.Fatal("must lock contract before checking availability")
					}
					return editRows(int64(101), int64(20), "460001", "2026", "Oprema", "Dobavljac", "X"), nil
				case strings.Contains(q, "COUNT(*) FROM TED.SAP_DOBAVLJACI"):
					return editRows(int64(1)), nil
				case strings.Contains(q, "COUNT(*) FROM TED.UGO_EVID"):
					n := int64(0)
					if tc.duplicate {
						n = 1
					}
					return editRows(n), nil
				case strings.Contains(q, "ORDER BY id"):
					if len(locks) != 2 {
						t.Fatal("supplier must be locked before resolving SLM")
					}
					if !strings.Contains(q, "status = 'A'") || !strings.Contains(q, "id_ugo_dob_lica_rola = 1") {
						t.Fatal("SLM filter missing")
					}
					if tc.newContact {
						return &schemaRows{width: 6}, nil
					}
					values := [][]driver.Value{{int64(7), "Ana", nil, "011", "ana@example.test", int64(2)}}
					if tc.multiple {
						values = append(values, []driver.Value{int64(8), "Drugo", nil, "012", "drugo@example.test", int64(1)})
					}
					return &schemaRows{width: 6, values: values}, nil
				case strings.Contains(q, "FOR UPDATE NOWAIT"):
					version := int64(2)
					if tc.changed {
						version = 3
					}
					return editRows(version), nil
				case strings.Contains(q, "COUNT(*) FROM TED.UGO_EDIT_LOCK"):
					active := int64(0)
					if tc.editing {
						active = 1
					}
					return editRows(active), nil
				}
				t.Fatal(q)
				return nil, nil
			}, func(q string, a []driver.NamedValue) (driver.Result, error) {
				switch {
				case strings.Contains(q, "INSERT INTO TED.UGO_EDIT_LOCK"):
					locks = append(locks, a[0].Value.(string))
				case strings.Contains(q, "INSERT INTO TED.UGO_DOB_LICA"):
					writes = append(writes, "contact")
					if !strings.Contains(q, "1,'A',SYSDATE,SYSDATE") {
						t.Fatal("new contact role/status/timestamps")
					}
					*a[len(a)-1].Value.(sql.Out).Dest.(*int) = 99
				case strings.Contains(q, "INSERT INTO TED.UGO_EVID"):
					writes = append(writes, "evidence")
					if a[0].Value != 101 || a[1].Value != 3 || !strings.Contains(q, "'A',SYSDATE") {
						t.Fatal("wrong registration values")
					}
					if tc.failEvidence {
						return nil, fmt.Errorf("test insert failure")
					}
					*a[len(a)-1].Value.(sql.Out).Dest.(*int) = 55
				case strings.Contains(q, "DELETE FROM TED.UGO_EDIT_LOCK"):
					released = true
				default:
					t.Fatal(q)
				}
				return driver.RowsAffected(1), nil
			})
			req := models.RegisterContract{ContractID: 101, ContactID: 7, ContactVersion: 2}
			if tc.newContact {
				req.ContactID = 0
				req.ContactVersion = 0
				req.Contact = &models.NewContractContact{Ime: "Novo", Telefon: "012", Email: "novo@example.test"}
			}
			result, err := store.RegisterContract(context.Background(), req, 3, 5)
			if tc.failEvidence {
				if err == nil || tx.committed || !tx.rolledBack || released || strings.Join(writes, ",") != "contact,evidence" {
					t.Fatalf("failed inserts not rolled back: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if tc.want != nil {
				if tx.committed || len(writes) != 0 || !tx.rolledBack {
					t.Fatal("conflict wrote data")
				}
				return
			}
			if !tx.committed || !released || result.ID != 55 || strings.Join(locks, ",") != "UGO_EVID_CREATE,UGO_SLM_CREATE" {
				t.Fatal("atomic registration failed")
			}
			if tc.newContact && (result.UgoDobLice.ID != 99 || result.Ime != "Novo") {
				t.Fatal("new contact not linked")
			}
			if !tc.newContact && (result.UgoDobLice.ID != 7 || result.Ime != "Ana") {
				t.Fatal("existing contact not copied")
			}
		})
	}
}

func TestUnregisteredContractsQuery(t *testing.T) {
	queries := 0
	store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
		checkBinds(t, q, args)
		queries++
		if !strings.Contains(q, "NOT EXISTS (SELECT 1 FROM TED.UGO_EVID e WHERE e.id_sap_ugovor = u.id)") || strings.Contains(q, "e.id_ugo_org") {
			t.Fatal("must exclude registered contracts globally")
		}
		if args[0].Value != "ACME" {
			t.Fatal("filter must be bound")
		}
		if strings.Contains(q, "SELECT COUNT(*) FROM TED.SAP_UGOVORI") {
			return editRows(int64(25)), nil
		}
		if !strings.Contains(q, "OFFSET :offset ROWS FETCH NEXT :limit ROWS ONLY") {
			t.Fatal("missing pagination")
		}
		return editRows(int64(101), int64(20), "460001", "2026", "Oprema", "ACME", "X"), nil
	}})
	items, total, err := store.GetUnregisteredContracts(context.Background(), 20, 20, " ACME ")
	if err != nil || total != 25 || len(items) != 1 || items[0].SupplierID != 20 || queries != 2 {
		t.Fatalf("items=%v total=%d err=%v", items, total, err)
	}
}

func TestRegistrationRejectsNewContactIfAnotherRequestCreatedSLM(t *testing.T) {
	store, tx := editStore(t, func(q string, a []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(q, "u.br_ugovor") {
			return editRows(int64(101), int64(20), "1", "2026", "P", "D", "X"), nil
		}
		if strings.Contains(q, "COUNT(*) FROM TED.SAP_DOBAVLJACI") {
			return editRows(int64(1)), nil
		}
		if strings.Contains(q, "COUNT(*) FROM TED.UGO_EVID") {
			return editRows(int64(0)), nil
		}
		return editRows(int64(7), "Ana", nil, "011", "ana@example.test", int64(1)), nil
	}, func(q string, a []driver.NamedValue) (driver.Result, error) {
		if !strings.Contains(q, "INSERT INTO TED.UGO_EDIT_LOCK") {
			t.Fatal("must not create duplicate contact")
		}
		return driver.RowsAffected(1), nil
	})
	_, err := store.RegisterContract(context.Background(), models.RegisterContract{ContractID: 101, Contact: &models.NewContractContact{Ime: "Novo"}}, 3, 5)
	if !errors.Is(err, ErrContractContactChanged) || tx.committed || !tx.rolledBack {
		t.Fatal(err)
	}
}
