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
				store.rows[0] = models.UgoEvidProsireni{BrUgovor: "00123", PredmetUgovora: "=1+1", Naziv: "Čačak & partner", PocetakUG: &d, Telefon: "+381001", OtvorenUG: "X", Godina: "2026", JN: "327 23", BrPozPlana: "0127-23", UgovorDMS: "500-00-UGO", NazivOdgZap1: "Prvo lice", NazivOdgZap6: "Šesto lice", Ime: "Kontakt osoba", Email: "kontakt@example.com"}
			}
			s, err := NewServer(util.Config{ActiveUserStatus: "A", TokenSymmetricKey: strings.Repeat("k", 32)}, store)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("GET", "/ugo_evid/otvoreni/excel?id_ugo_org=1&dobavljac=%20Firma%20&page_id=8&page_size=1", nil)
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
			if len(rows) != tc.count+4 {
				t.Fatalf("got %d rows, want %d", len(rows), tc.count+4)
			}
			if len(rows) == 0 || rows[0][0] != "Za kompanijsku upotrebu" {
				t.Fatal("missing header")
			}
			for cell, want := range map[string]string{"A3": "Godina", "I3": "Ugovor važi", "K3": "Lica za operativno praćenje ugovora", "Q3": "Kontakt osoba - Service Level Manager PARTNER", "Q4": "Ime i prezime", "R4": "Telefon", "S4": "Email"} {
				if got, err := f.GetCellValue("Otvoreni ugovori", cell); err != nil || got != want {
					t.Fatalf("header %s = %q, %v", cell, got, err)
				}
			}
			merges, err := f.GetMergeCells("Otvoreni ugovori")
			if err != nil || len(merges) != 12 {
				t.Fatalf("expected 12 merged heading ranges: %v, %v", merges, err)
			}
			panes, err := f.GetPanes("Otvoreni ugovori")
			if err != nil || !panes.Freeze || panes.YSplit != 4 {
				t.Fatalf("headings not frozen: %+v, %v", panes, err)
			}
			for cell, color := range map[string]string{"A1": "FCE4D6", "A3": "D9D9D9", "Q3": "FFFF00"} {
				id, err := f.GetCellStyle("Otvoreni ugovori", cell)
				if err != nil {
					t.Fatal(err)
				}
				style, err := f.GetStyle(id)
				if err != nil || len(style.Fill.Color) == 0 || style.Fill.Color[0] != color {
					t.Fatalf("incorrect style for %s: %+v, %v", cell, style, err)
				}
			}
			if tc.count > 0 {
				for cell, want := range map[string]string{"A5": "2026", "B5": "DA", "C5": "327 23", "D5": "=1+1", "E5": "0127-23", "F5": "00123", "G5": "500-00-UGO", "H5": "Čačak & partner", "I5": "25.09.2026", "J5": "", "K5": "Prvo lice", "P5": "Šesto lice", "Q5": "Kontakt osoba", "R5": "+381001", "S5": "kontakt@example.com"} {
					v, e := f.GetCellValue("Otvoreni ugovori", cell)
					if e != nil || v != want {
						t.Fatalf("%s: %q %v", cell, v, e)
					}
				}
				formula, _ := f.GetCellFormula("Otvoreni ugovori", "D5")
				if formula != "" {
					t.Fatal("text treated as formula")
				}
			}
		})
	}
}
