package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveDocumentConfinement(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, path := range []string{"", ".", "..", "../file.pdf", "folder/../file.pdf", `C:\file.pdf`, "/file.pdf"} {
		if err := RemoveDocument(root, path); err == nil {
			t.Fatalf("unsafe path accepted: %q", path)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "folder"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := RemoveDocument(root, "folder"); err == nil {
		t.Fatal("directory removed")
	}
	if err := os.WriteFile(filepath.Join(dir, "folder", "file.pdf"), []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := RemoveDocument(root, "folder/file.pdf"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "folder")); err != nil {
		t.Fatal("contract folder removed")
	}
}
