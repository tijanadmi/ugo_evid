package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestDeletionNotificationsThresholdAndDeduplication(t *testing.T) {
	for _, mode := range []string{"early", "overdue", "resolved", "no error"} {
		t.Run(mode, func(t *testing.T) {
			merges := 0
			message := "permission denied"
			if mode == "no error" {
				message = ""
			}
			store, tx := editStore(t, func(q string, args []driver.NamedValue) (driver.Rows, error) {
				if !strings.Contains(q, "FOR UPDATE WAIT 5") || !strings.Contains(q, "INTERVAL '15' MINUTE") {
					return nil, errors.New("missing lock or threshold")
				}
				if mode == "resolved" {
					return editRows(time.Now(), int64(1)), nil
				}
				if mode == "early" {
					return editRows(nil, int64(0)), nil
				}
				return editRows(nil, int64(1)), nil
			}, func(q string, args []driver.NamedValue) (driver.Result, error) {
				if strings.Contains(q, "MERGE INTO") {
					merges++
					if !strings.Contains(q, "n.izvor_kljuc=s.source_key") || !strings.Contains(q, "broj_ponavljanja+1") || strings.Contains(q, "status='OTVORENO'") {
						return nil, errors.New("must reuse incident and preserve ownership")
					}
					if args[0].Value != "DOC_DELETE:11" {
						return nil, errors.New("wrong event key")
					}
				} else if mode == "resolved" && args[0].Value != "" {
					return nil, errors.New("late failed report must not restore error after success")
				}
				return driver.RowsAffected(1), nil
			})
			err := store.RecordDocumentDeletionAttempt(context.Background(), 11, message)
			want := 0
			if mode == "overdue" {
				want = 1
			}
			if err != nil || !tx.committed || merges != want {
				t.Fatalf("err=%v merges=%d", err, merges)
			}
		})
	}
	if s := limitedError(strings.Repeat("š", 2000)); len(s) > 2000 || !utf8.ValidString(s) {
		t.Fatal("invalid error truncation")
	}
}

func TestNotificationClaimAndResolveGuards(t *testing.T) {
	for _, mode := range []string{"claim", "stale", "already claimed", "pending delete", "resolved delete"} {
		t.Run(mode, func(t *testing.T) {
			writes := 0
			store, tx := editStore(t, func(q string, args []driver.NamedValue) (driver.Rows, error) {
				switch {
				case strings.Contains(q, "PORTAL_ADMIN_LOCK"):
					return editRows(int64(1)), nil
				case strings.Contains(q, "k.id=:actor"):
					return editRows(int64(1)), nil
				case strings.HasPrefix(q, "SELECT id FROM TED.PORTAL_OBAVESTENJE"):
					return editRows(int64(8)), nil
				case strings.HasPrefix(q, "SELECT COUNT(*) FROM TED.UGO_DOC_BRISANJE"):
					if mode == "pending delete" {
						return editRows(int64(1)), nil
					}
					return editRows(int64(0)), nil
				default:
					status := "OTVORENO"
					if mode == "already claimed" {
						status = "PREUZETO"
					}
					return editRows(int64(8), "DOC_DELETE", int64(11), "Title", "error", status, int64(0), nil, "2026-11-12T10:00:00", "2026-11-12T10:15:00", int64(1), nil, int64(2), "doc.pdf", int64(9), int64(15)), nil
				}
			}, func(q string, args []driver.NamedValue) (driver.Result, error) {
				writes++
				return driver.RowsAffected(1), nil
			})
			version := int64(2)
			if mode == "stale" {
				version = 1
			}
			action := "PREUZMI"
			if strings.Contains(mode, "delete") {
				action = "RESI"
			}
			err := store.ChangeAdminNotification(context.Background(), 7, "A", 8, version, action)
			success := mode == "claim" || mode == "resolved delete"
			if (err == nil) != success || tx.committed != success {
				t.Fatalf("err=%v committed=%v", err, tx.committed)
			}
			if !success && writes != 0 {
				t.Fatal("conflicting notification was changed")
			}
		})
	}
}
