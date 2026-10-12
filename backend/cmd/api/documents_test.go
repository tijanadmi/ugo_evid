package api

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type documentTestStore struct {
	repository.Store
	doc                        *models.Document
	foreign, closed, failFinal bool
	writes                     int
}

func (s *documentTestStore) GetUserByUsername(context.Context, string) (*models.User, error) {
	return &models.User{ID: 7, Status: "A"}, nil
}
func (s *documentTestStore) GetUserOrganization(context.Context, string, string) (int, error) {
	return 3, nil
}
func (s *documentTestStore) GetUgoEvidProsireniByID(context.Context, int) (models.UgoEvidProsireni, error) {
	org := 3
	if s.foreign {
		org = 4
	}
	open := "X"
	if s.closed {
		open = ""
	}
	return models.UgoEvidProsireni{IDUgoEvid: 9, IDUgoOrg: org, BrUgovor: "460", PredmetUgovora: "Oprema", OtvorenUG: open}, nil
}
func (s *documentTestStore) ListDocumentTypes(context.Context) ([]models.DocumentType, error) {
	return []models.DocumentType{{ID: 1, Code: "UGOVOR", Name: "Ugovor"}}, nil
}
func (s *documentTestStore) ListDocuments(context.Context, int) ([]models.Document, error) {
	if s.doc != nil && s.doc.Status == "A" {
		return []models.Document{*s.doc}, nil
	}
	return []models.Document{}, nil
}
func (s *documentTestStore) GetDocument(context.Context, int) (*models.Document, error) {
	if s.doc == nil || s.doc.Status != "A" {
		return nil, sql.ErrNoRows
	}
	d := *s.doc
	return &d, nil
}
func (s *documentTestStore) DocumentFolder(_ context.Context, _ int, proposed string) (string, error) {
	return proposed, nil
}
func (s *documentTestStore) ReserveDocument(_ context.Context, d *models.Document) error {
	s.writes++
	d.ID = 11
	d.Status = "P"
	d.Version = 1
	copy := *d
	s.doc = &copy
	return nil
}
func (s *documentTestStore) FinalizeDocument(context.Context, int) error {
	if s.failFinal {
		return errors.New("database response lost")
	}
	s.doc.Status = "A"
	return nil
}
func (s *documentTestStore) DocumentLock(context.Context, int, int, string, string) (*models.EditLock, error) {
	return &models.EditLock{Token: strings.Repeat("a", 64)}, nil
}
func (s *documentTestStore) ChangeDocument(_ context.Context, d *models.Document, userID int, token string, remove bool) error {
	if token != strings.Repeat("a", 64) {
		return repository.ErrDocumentLease
	}
	if d.Version != s.doc.Version {
		return repository.ErrDocumentVersion
	}
	s.writes++
	if remove {
		s.doc.Status = "D"
	} else {
		s.doc.DocumentMetadata = d.DocumentMetadata
	}
	s.doc.Version++
	return nil
}
func docServer(t *testing.T, s *documentTestStore, max int64) (*Server, string) {
	t.Helper()
	root := t.TempDir()
	server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A", DocumentRoot: root, DocumentMaxBytes: max}, s)
	if err != nil {
		t.Fatal(err)
	}
	return server, root
}
func docRequest(t *testing.T, s *Server, method, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	access, _, err := s.tokenMaker.CreateToken("test.user", "user", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+access)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	return rec
}
func uploadBody(t *testing.T, name, content string, count int) (*bytes.Buffer, string) {
	t.Helper()
	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	writer.WriteField("metadata", `{"id_tip_doc":1,"naziv":"Ugovor","datum_dokumenta":"2026-10-11"}`)
	for i := 0; i < count; i++ {
		part, err := writer.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		io.WriteString(part, content)
	}
	writer.Close()
	return buf, writer.FormDataContentType()
}
func TestDocumentUploadAndDownload(t *testing.T) {
	store := &documentTestStore{}
	server, root := docServer(t, store, 1024)
	pdf := "%PDF-1.4\n%%EOF"
	body, kind := uploadBody(t, `C:\fakepath\ugovor.pdf`, pdf, 1)
	rec := docRequest(t, server, "POST", "/ugo_evid/9/dokumenti", body, kind)
	if rec.Code != 201 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if store.doc.UserID != 7 || store.doc.ContractID != 9 || store.doc.OriginalName != "ugovor.pdf" || store.doc.Status != "A" {
		t.Fatalf("%+v", store.doc)
	}
	data, err := os.ReadFile(filepath.Join(root, store.doc.Path))
	if err != nil || string(data) != pdf {
		t.Fatalf("%s %v", data, err)
	}
	rec = docRequest(t, server, "GET", "/ugo_doc/11/sadrzaj", nil, "")
	if rec.Code != 200 || rec.Body.String() != pdf || !strings.HasPrefix(rec.Header().Get("Content-Disposition"), "attachment;") {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	rec = docRequest(t, server, "GET", "/ugo_doc/11/sadrzaj?preview=1", nil, "")
	if !strings.HasPrefix(rec.Header().Get("Content-Disposition"), "inline;") || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("unsafe content headers")
	}
	rec = docRequest(t, server, "GET", "/ugo_evid/9/dokumenti", nil, "")
	if rec.Code != 200 || strings.Contains(rec.Body.String(), store.doc.Path) {
		t.Fatal("storage path leaked")
	}
	rec = docRequest(t, server, "DELETE", "/ugo_doc/11", strings.NewReader(`{"version":"1","lock_token":"`+strings.Repeat("a", 64)+`"}`), "application/json")
	if rec.Code != 200 || store.doc.Status != "D" {
		t.Fatal(rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, store.doc.Path)); err != nil {
		t.Fatal("logical delete removed physical file")
	}
	rec = docRequest(t, server, "GET", "/ugo_doc/11/sadrzaj", nil, "")
	if rec.Code != 404 {
		t.Fatal("deleted document remains public")
	}
}
func TestDocumentUploadValidation(t *testing.T) {
	for _, tc := range []struct {
		name, content string
		count         int
		want          int
	}{
		{"none.pdf", "", 0, 400}, {"two.pdf", "%PDF-1.4", 2, 400}, {"empty.pdf", "", 1, 413},
		{"fake.pdf", "<html>fake</html>", 1, 400}, {"large.pdf", strings.Repeat("x", 200), 1, 413}, {"bad.exe", "%PDF-1.4", 1, 400},
	} {
		store := &documentTestStore{}
		server, _ := docServer(t, store, 100)
		body, kind := uploadBody(t, tc.name, tc.content, tc.count)
		rec := docRequest(t, server, "POST", "/ugo_evid/9/dokumenti", body, kind)
		if rec.Code != tc.want || store.writes != 0 {
			t.Fatalf("%s: %d %s", tc.name, rec.Code, rec.Body.String())
		}
	}
}
func TestDocumentAccessAllEndpoints(t *testing.T) {
	for _, target := range []struct{ method, path string }{
		{"GET", "/ugo_evid/9/dokumenti"}, {"POST", "/ugo_evid/9/dokumenti"}, {"GET", "/ugo_doc/11"}, {"GET", "/ugo_doc/11/sadrzaj"},
		{"POST", "/ugo_doc/11/lock"}, {"PUT", "/ugo_doc/11"}, {"DELETE", "/ugo_doc/11"},
	} {
		store := &documentTestStore{foreign: true, doc: &models.Document{ID: 11, ContractID: 9, Status: "A"}}
		server, _ := docServer(t, store, 1024)
		rec := docRequest(t, server, target.method, target.path, nil, "")
		if rec.Code != 403 || store.writes != 0 {
			t.Fatalf("%s: %d %s", target.path, rec.Code, rec.Body.String())
		}
	}
	store := &documentTestStore{closed: true}
	server, _ := docServer(t, store, 1024)
	rec := docRequest(t, server, "POST", "/ugo_evid/9/dokumenti", nil, "")
	if rec.Code != 409 {
		t.Fatal(rec.Body.String())
	}
}
func TestDocumentAmbiguousFinalizationPreservesFile(t *testing.T) {
	store := &documentTestStore{failFinal: true}
	server, root := docServer(t, store, 1024)
	body, kind := uploadBody(t, "a.pdf", "%PDF-1.4", 1)
	rec := docRequest(t, server, "POST", "/ugo_evid/9/dokumenti", body, kind)
	if rec.Code != 500 || store.doc.Status != "P" {
		t.Fatal(rec.Body.String())
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := util.VerifyDocument(r, store.doc.Path, store.doc.SHA256, store.doc.Size); err != nil {
		t.Fatal("lost recoverable file", err)
	}
	rec = docRequest(t, server, "GET", "/ugo_doc/11/sadrzaj", nil, "")
	if rec.Code != 404 {
		t.Fatal("pending document exposed")
	}
}
func TestDocumentUnauthenticated(t *testing.T) {
	server, _ := docServer(t, &documentTestStore{}, 1024)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ugo_evid/9/dokumenti", nil))
	if rec.Code != 401 {
		t.Fatal(rec.Code)
	}
}
