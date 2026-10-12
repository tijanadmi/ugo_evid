package util

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentFiles(t *testing.T) {
	root, err := DocumentStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	pdf := []byte("%PDF-1.4\n1 0 obj <<>> endobj\n%%EOF")
	temp, ext, mime, hash, size, err := ReceiveDocument(root, bytes.NewReader(pdf), 1024, "../../ugovor.pdf")
	if err != nil || ext != ".pdf" || mime != "application/pdf" || size != int64(len(pdf)) || len(hash) != 64 {
		t.Fatalf("%s %s %d %v", ext, mime, size, err)
	}
	path := "1_460_oprema/unique.pdf"
	if err := PublishDocument(root, temp, path); err != nil {
		t.Fatal(err)
	}
	if err := PublishDocument(root, temp, path); err == nil {
		t.Fatal("overwrote existing file")
	}
	if err := VerifyDocument(root, path, hash, size); err != nil {
		t.Fatal(err)
	}
	if err := VerifyDocument(root, path, strings.Repeat("0", 64), size); err == nil {
		t.Fatal("accepted corrupt checksum")
	}
	if err := PublishDocument(root, temp, "../outside.pdf"); err == nil {
		t.Fatal("escaped storage root")
	}
	if err := root.Remove(temp); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, content string
		limit         int64
		want          error
	}{
		{"empty.pdf", "", 100, ErrDocumentSize}, {"large.pdf", string(pdf), 4, ErrDocumentSize},
		{"fake.pdf", "<html>not PDF</html>", 100, ErrDocumentFile}, {"bad.exe", string(pdf), 100, ErrDocumentFile},
	} {
		_, _, _, _, _, err := ReceiveDocument(root, strings.NewReader(tc.content), tc.limit, tc.name)
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
	files, err := os.ReadDir(filepath.Join(root.Name(), ".staging"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary files leaked: %v %v", files, err)
	}
	if SafeOriginalName(`C:\fakepath\a.pdf`) != "a.pdf" {
		t.Fatal("Windows original name")
	}
	if name := DocumentFolderName(7, "460/12", strings.Repeat("subject:*?", 100)); strings.ContainsAny(name, `/:*?\`) || len(name) > 130 || !strings.HasPrefix(name, "7_") {
		t.Fatal(name)
	}
}
func TestDocxValidation(t *testing.T) {
	for _, macro := range []bool{false, true} {
		var data bytes.Buffer
		z := zip.NewWriter(&data)
		w, _ := z.Create("[Content_Types].xml")
		w.Write([]byte(`<Types><Override ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`))
		w, _ = z.Create("word/document.xml")
		w.Write([]byte(`<document/>`))
		if macro {
			w, _ = z.Create("word/vbaProject.bin")
			w.Write([]byte("macro"))
		}
		z.Close()
		root, err := DocumentStorage(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		_, ext, _, _, _, err := ReceiveDocument(root, bytes.NewReader(data.Bytes()), 1024*1024, "a.docx")
		root.Close()
		if macro && !errors.Is(err, ErrDocumentFile) {
			t.Fatalf("macro accepted: %v", err)
		}
		if !macro && (err != nil || ext != ".docx") {
			t.Fatalf("valid docx: %v", err)
		}
	}
}
