package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"strings"
	"testing"
)

func TestTechnicalReportChecksSnapshotAndLease(t *testing.T) {
	for _, mode := range []string{"success", "monthly", "lease", "version", "template", "signed"} {
		t.Run(mode, func(t *testing.T) {
			writes := 0
			store, tx := editStore(t, func(q string, args []driver.NamedValue) (driver.Rows, error) {
				switch {
				case strings.Contains(q, "UGO_EVID_PROSIRENI_V"):
					return editRows(int64(1)), nil
				case strings.Contains(q, "UGO_EDIT_LOCK"):
					if mode == "lease" {
						return editRows(int64(1)), nil
					}
					return editRows(int64(0)), nil
				case strings.Contains(q, "FROM TED.UGO_ODRZAVANJE"):
					return editRows(int64(7)), nil
				case strings.Contains(q, "FROM TED.UGO_TEH_PERIOD"):
					signed := "2026-04-01"
					version := int64(2)
					if mode == "signed" {
						signed = "2026-04-02"
					}
					if mode == "version" {
						version = 3
					}
					return editRows(version, int64(5), "2026-01-01", "2026-03-31", signed, "ZAVRSEN"), nil
				case strings.Contains(q, "FROM TED.UGO_DOC_TEMPLATE"):
					if mode == "template" {
						return editRows(int64(10)), nil
					}
					return editRows(int64(9)), nil
				}
				return nil, errors.New("unexpected query")
			}, func(q string, args []driver.NamedValue) (driver.Result, error) {
				writes++
				if strings.Contains(q, "UGO_TEH_IZVESTAJ") {
					t.Fatal("must not archive reports")
				}
				if strings.Contains(q, "UPDATE TED.UGO_TEH_PERIOD") && strings.Contains(q, "status=") {
					return nil, errors.New("generation must not complete period")
				}
				return driver.RowsAffected(1), nil
			})
			p := models.TechnicalReport{ContractID: 1, PeriodID: 2, TemplateID: 9, Quarter: 3, Start: "2026-01-01", End: "2026-03-31", Signed: "2026-04-01"}
			if mode == "monthly" {
				p.Quarter = 0
			}
			err := store.ConfirmTechnicalReport(context.Background(), &p, 2, 4)
			success := mode == "success" || mode == "monthly"
			if (err == nil) != success || tx.committed != success {
				t.Fatalf("err=%v committed=%v", err, tx.committed)
			}
			if !success && writes != 0 {
				t.Fatal("conflict wrote data")
			}
		})
	}
}
