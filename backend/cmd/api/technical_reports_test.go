package api

import (
	"archive/zip"
	"bytes"
	"context"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/util"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type reportTestStore struct {
	documentTestStore
	templates []models.ReportTemplate
	reports   []models.TechnicalReport
	signed    string
}

func (s *reportTestStore) ListReportTemplates(context.Context, int) ([]models.ReportTemplate, error) {
	return s.templates, nil
}
func (s *reportTestStore) SaveReportTemplate(_ context.Context, t *models.ReportTemplate, expected, user int) error {
	t.ID = 1
	t.Version = expected + 1
	s.templates = append(s.templates, *t)
	return nil
}
func (s *reportTestStore) ListTechnicalReports(context.Context, int, int) ([]models.TechnicalReport, error) {
	return s.reports, nil
}
func (s *reportTestStore) SaveTechnicalReport(_ context.Context, p *models.TechnicalReport, _ int64, _ int) error {
	p.ID = 1
	s.reports = append(s.reports, *p)
	return nil
}
func (s *reportTestStore) GetTracking(context.Context, int) (*models.TrackingData, error) {
	return &models.TrackingData{Periods: []models.TrackingItem{{ID: 7, Version: 2, Start: "2026-01-01", End: "2026-03-31", Signed: s.signed, DocumentType: 5, Status: "NA_POTPISU"}}}, nil
}

func TestTechnicalReportUploadGenerateDownload(t *testing.T) {
	store := &reportTestStore{signed: "2026-04-01"}
	server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A", DocumentRoot: t.TempDir()}, store)
	if err != nil {
		t.Fatal(err)
	}
	var template bytes.Buffer
	z := zip.NewWriter(&template)
	for name, body := range map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>{{datum_od}} {{datum_do}} {{datum_potpisa}} {{broj_kvartala}}</w:t></w:r></w:p></w:document>`} {
		w, _ := z.Create(name)
		io.WriteString(w, body)
	}
	z.Close()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	form.WriteField("id_tip_doc", "5")
	form.WriteField("verzija", "0")
	w, _ := form.CreateFormFile("file", "template.docx")
	w.Write(template.Bytes())
	form.Close()
	rec := docRequest(t, server, "POST", "/ugo_evid/9/sabloni", &body, form.FormDataContentType())
	if rec.Code != 201 {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	payload := `{"id_template":1,"broj_kvartala":5,"version":"2","pdf":false}`
	rec = docRequest(t, server, "POST", "/ugo_evid/9/teh-period/7/izvestaji", strings.NewReader(payload), "application/json")
	if rec.Code != 201 {
		t.Fatalf("generate: %d %s", rec.Code, rec.Body.String())
	}
	if len(store.reports) != 1 || store.reports[0].Quarter != 5 || store.reports[0].Signed != store.signed {
		t.Fatal("snapshot not saved")
	}
	rec = docRequest(t, server, "GET", "/ugo_evid/9/teh-period/7/izvestaji/1?format=docx", nil, "")
	if rec.Code != 200 {
		t.Fatalf("download: %d", rec.Code)
	}
	generated, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range generated.File {
		if f.Name == "word/document.xml" {
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			r.Close()
			if !strings.Contains(string(b), "01.01.2026 31.03.2026 01.04.2026 5") {
				t.Fatalf("wrong rendered values: %s", b)
			}
		}
	}
	// An absent quarter marker wins over a stale/malicious client quarter.
	var monthly bytes.Buffer
	mz := zip.NewWriter(&monthly)
	source, _ := zip.NewReader(bytes.NewReader(template.Bytes()), int64(template.Len()))
	for _, f := range source.File {
		r, _ := f.Open()
		b, _ := io.ReadAll(r)
		r.Close()
		w, _ := mz.Create(f.Name)
		w.Write(bytes.ReplaceAll(b, []byte("{{broj_kvartala}}"), nil))
	}
	mz.Close()
	if err = os.WriteFile(filepath.Join(server.config.DocumentRoot, store.templates[0].Path), monthly.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	rec = docRequest(t, server, "GET", "/ugo_evid/9/sabloni", nil, "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"koristi_kvartal":false`) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	rec = docRequest(t, server, "POST", "/ugo_evid/9/teh-period/7/izvestaji", strings.NewReader(payload), "application/json")
	if rec.Code != 201 || len(store.reports) != 2 || store.reports[1].Quarter != 0 {
		t.Fatal("monthly report required or stored a quarter")
	}
	store.signed = ""
	rec = docRequest(t, server, "POST", "/ugo_evid/9/teh-period/7/izvestaji", strings.NewReader(payload), "application/json")
	if rec.Code != 400 || len(store.reports) != 2 {
		t.Fatal("unsigned report generated")
	}
	store.foreign = true
	rec = docRequest(t, server, "GET", "/ugo_evid/9/teh-period/7/izvestaji/1?format=docx", nil, "")
	if rec.Code != 403 {
		t.Fatal("foreign contract report leaked")
	}
}
