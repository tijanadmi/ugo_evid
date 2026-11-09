package api

import (
	"bytes"
	"context"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
	"github.com/xuri/excelize/v2"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type exportStore struct {
	prosireniStore
	rows []models.UgoEvidProsireni
}

func (s *exportStore) ExportOpenContracts(_ context.Context, org int, supplier string) ([]models.UgoEvidProsireni, error) {
	s.called = true
	s.orgID = org
	s.supplier = supplier
	return s.rows, s.err
}

func TestContractsExcel(t *testing.T) {
	for _, tc := range []struct {
		name            string
		orgErr, errorDB error
		auth            bool
		count, status   int
	}{
		{name: "all rows beyond page", auth: true, count: 105, status: 200},
		{name: "empty workbook", auth: true, status: 200},
		{name: "unauthenticated", status: 401},
		{name: "missing org", auth: true, orgErr: repository.ErrUserOrganization, status: 403},
		{name: "database failure", auth: true, errorDB: errors.New("secret DB detail"), status: 500},
		{name: "limit refuses truncation", auth: true, count: 10001, status: 422},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &exportStore{prosireniStore: prosireniStore{orgErr: tc.orgErr, err: tc.errorDB}, rows: make([]models.UgoEvidProsireni, tc.count)}
			for i := range store.rows {
				store.rows[i].BrUgovor = "Ugovor"
			}
			if tc.count > 0 {
				d := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
				store.rows[0] = models.UgoEvidProsireni{BrUgovor: "00123", PredmetUgovora: "=1+1", Naziv: "Čačak & partner", PocetakUG: &d, Telefon: "+381001"}
			}
			s, err := NewServer(util.Config{ActiveUserStatus: "A", TokenSymmetricKey: strings.Repeat("k", 32)}, store)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("GET", "/ugo_evid/otvoreni/excel?id_ugo_org=44&dobavljac=%20Firma%20&page_id=8&page_size=1", nil)
			if tc.auth {
				token, _, _ := s.tokenMaker.CreateToken("test", "user", time.Minute)
				req.Header.Set("Authorization", "Bearer "+token)
			}
			rec := httptest.NewRecorder()
			s.router.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
			if (!tc.auth || tc.orgErr != nil) && store.called {
				t.Fatal("unauthorized database call")
			}
			if store.called && (store.orgID != 1 || store.supplier != "Firma") {
				t.Fatal("wrong export scope")
			}
			if tc.status != 200 {
				if strings.Contains(rec.Body.String(), "secret DB") {
					t.Fatal("DB details leaked")
				}
				return
			}
			if !strings.Contains(rec.Header().Get("Content-Disposition"), ".xlsx") {
				t.Fatal("missing attachment")
			}
			f, err := excelize.OpenReader(bytes.NewReader(rec.Body.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			rows, err := f.GetRows("Otvoreni ugovori")
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != tc.count+1 {
				t.Fatalf("got %d rows, want %d", len(rows), tc.count+1)
			}
			if len(rows) == 0 || rows[0][0] != "Broj ugovora" {
				t.Fatal("missing header")
			}
			if tc.count > 0 {
				for cell, want := range map[string]string{"A2": "00123", "C2": "=1+1", "E2": "Čačak & partner", "I2": "25.09.2026", "N2": "+381001"} {
					v, e := f.GetCellValue("Otvoreni ugovori", cell)
					if e != nil || v != want {
						t.Fatalf("%s: %q %v", cell, v, e)
					}
				}
				formula, _ := f.GetCellFormula("Otvoreni ugovori", "C2")
				if formula != "" {
					t.Fatal("text treated as formula")
				}
			}
		})
	}
}
