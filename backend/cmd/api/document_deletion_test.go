package api

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tijanadmi/ugo_evid/util"
)

func (s *documentTestStore) PendingDocumentDeletions(context.Context) ([]int, error) {
	if s.doc != nil && s.doc.Status == "D" {
		return []int{s.doc.ID}, nil
	}
	return nil, nil
}
func (s *documentTestStore) FinishDocumentDeletion(_ context.Context, id int, root *os.Root) error {
	if s.failDelete {
		return errors.New("simulated storage failure")
	}
	if s.doc == nil {
		return nil
	}
	if s.doc.ID != id || s.doc.Status != "D" {
		return errors.New("not pending deletion")
	}
	if err := util.RemoveDocument(root, s.doc.Path); err != nil {
		return err
	}
	s.doc = nil
	return nil
}

func TestPendingDeletionRetriesAndHidesContent(t *testing.T) {
	s := &documentTestStore{failDelete: true}
	server, dir := docServer(t, s, 1024)
	body, kind := uploadBody(t, "file.pdf", "%PDF-1.4\n%%EOF", 1)
	r := docRequest(t, server, "POST", "/ugo_evid/9/dokumenti", body, kind)
	if r.Code != 201 {
		t.Fatal(r.Body.String())
	}
	path := filepath.Join(dir, s.doc.Path)
	r = docRequest(t, server, "DELETE", "/ugo_doc/11", strings.NewReader(`{"version":"1","lock_token":"`+strings.Repeat("a", 64)+`"}`), "application/json")
	if r.Code != 202 || !strings.Contains(r.Body.String(), "pending_deletion") || s.doc.Status != "D" {
		t.Fatalf("%d %s", r.Code, r.Body.String())
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"/ugo_doc/11", "/ugo_doc/11/sadrzaj", "/ugo_doc/11/sadrzaj?preview=1"} {
		if r := docRequest(t, server, "GET", endpoint, nil, ""); r.Code != 404 {
			t.Fatalf("pending file accessible: %s", endpoint)
		}
	}
	server.retryDocumentDeletions(context.Background())
	if s.doc == nil {
		t.Fatal("failed retry removed record")
	}
	s.failDelete = false
	server.retryDocumentDeletions(context.Background())
	if s.doc != nil {
		t.Fatal("retry did not finish deletion")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("file survived retry")
	}
	server.retryDocumentDeletions(context.Background())
}
