package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
	"testing"
)

func TestContractPeopleReplaceSLM(t *testing.T) {
	for _, mode := range []string{"ok", "stale", "foreign", "inactive", "locked", "closed", "insert_failure"} {
		t.Run(mode, func(t *testing.T) {
			writes := []string{}
			store, tx := editStore(t, func(q string, args []driver.NamedValue) (driver.Rows, error) {
				checkBinds(t, q, args)
				switch {
				case strings.Contains(q, "FROM TED.UGO_DOB_LICA WHERE"):
					status := "A"
					if mode == "inactive" {
						status = "P"
					}
					return editRows(int64(20), status), nil
				case strings.Contains(q, "UGO_EDIT_LOCK"):
					n := int64(0)
					if mode == "locked" {
						n = 1
					}
					return editRows(n), nil
				case strings.Contains(q, "SELECT id_ugo_org"):
					return editRows(int64(3), int64(101)), nil
				case strings.Contains(q, "UGO_EVID_PROSIRENI_V"):
					n := int64(1)
					if mode == "closed" {
						n = 0
					}
					return editRows(n), nil
				case strings.Contains(q, "FROM TED.SAP_UGOVORI"):
					n := int64(1)
					if mode == "foreign" {
						n = 0
					}
					return editRows(n), nil
				case strings.Contains(q, "UGO_DOB_LICA_ROLE"):
					return editRows(int64(1)), nil
				case strings.Contains(q, "SELECT id,version"):
					v := int64(2)
					if mode == "stale" {
						v = 3
					}
					return editRows(int64(4), v), nil
				}
				t.Fatalf("unexpected query %s", q)
				return nil, nil
			}, func(q string, args []driver.NamedValue) (driver.Result, error) {
				checkBinds(t, q, args)
				writes = append(writes, q)
				if strings.Contains(q, "INSERT") && mode == "insert_failure" {
					return nil, errors.New("insert failed")
				}
				if !strings.Contains(q, "TED.UGO_EVID_LICA") {
					t.Fatal("must change only this contract's links")
				}
				return driver.RowsAffected(1), nil
			})
			err := store.ChangeContractPerson(context.Background(), 9, 5, models.ContractPersonChange{PersonID: 8, RoleID: 1, PreviousSLM: 4, PreviousVersion: 2}, false)
			if (err == nil) != (mode == "ok") || tx.committed != (mode == "ok") {
				t.Fatalf("%v committed=%v", err, tx.committed)
			}
			if mode == "ok" && (len(writes) != 2 || !strings.Contains(writes[0], "status='P'") || !strings.Contains(writes[1], "INSERT")) {
				t.Fatal("replacement must be atomic")
			}
			if mode != "ok" && mode != "insert_failure" && len(writes) != 0 {
				t.Fatal("conflict changed assignments")
			}
			if mode == "insert_failure" && !tx.rolledBack {
				t.Fatal("old SLM must remain on insert failure")
			}
		})
	}
}

func TestContractContactsAreMappedByContract(t *testing.T) {
	supplier := 20
	items := []models.UgoEvidProsireni{{IDUgoEvid: 1, IDSapDobavljac: &supplier}, {IDUgoEvid: 2, IDSapDobavljac: &supplier}}
	store := testSchemaStore(t, &schemaConn{query: func(q string, args []driver.NamedValue) (driver.Rows, error) {
		checkBinds(t, q, args)
		if !strings.Contains(q, "a.id_ugo_evid IN") || !strings.Contains(q, "a.status='A'") {
			t.Fatal("missing assignment scope")
		}
		return &schemaRows{width: 8, values: [][]driver.Value{{int64(1), "Ana", nil, "011", "a@example.test", "SLM", int64(10), "A"}, {int64(2), "Bojan", nil, "012", "b@example.test", "SLM", int64(11), "A"}}}, nil
	}})
	if err := store.loadContractContacts(context.Background(), items); err != nil {
		t.Fatal(err)
	}
	if len(items[0].LicaDobavljaca) != 1 || items[0].LicaDobavljaca[0].Ime != "Ana" || len(items[1].LicaDobavljaca) != 1 || items[1].LicaDobavljaca[0].Ime != "Bojan" {
		t.Fatal("supplier teams leaked across contracts")
	}
}
