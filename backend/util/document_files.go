package util

import (
	"archive/zip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

var ErrDocumentFile = errors.New("Dozvoljeni su PDF, DOCX, JPG i PNG fajlovi sa odgovarajucim sadrzajem.")
var ErrDocumentSize = errors.New("Fajl je prazan ili prevelik.")

func DocumentStorage(path string) (*os.Root, error) {
	if path == "" {
		path = "./doc"
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, err
	}
	return os.OpenRoot(path)
}
func DocumentFolderName(id int, number, subject string) string {
	slug := func(s string, limit int) string {
		var b strings.Builder
		dash := false
		for _, r := range strings.ToLower(s) {
			if b.Len() >= limit {
				break
			}
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
				b.WriteRune(r)
				dash = false
			} else if b.Len() > 0 && !dash {
				b.WriteByte('-')
				dash = true
			}
		}
		return strings.Trim(b.String(), "-")
	}
	return fmt.Sprintf("%d_%s_%s", id, slug(number, 40), slug(subject, 80))
}
func SafeOriginalName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
}
func RandomDocumentName() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Receive to a private staging directory. The caller always removes the temp
// file; final files referenced by a pending DB row survive ambiguous commits.
func ReceiveDocument(root *os.Root, src io.Reader, max int64, original string) (temp, extension, mime, hash string, size int64, err error) {
	if err = root.MkdirAll(".staging", 0700); err != nil {
		return
	}
	var name string
	name, err = RandomDocumentName()
	if err != nil {
		return
	}
	temp = filepath.Join(".staging", name+".upload")
	f, e := root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if e != nil {
		err = e
		return
	}
	defer func() {
		f.Close()
		if err != nil {
			root.Remove(temp)
		}
	}()
	digest := sha256.New()
	size, err = io.Copy(io.MultiWriter(f, digest), io.LimitReader(src, max+1))
	if err != nil {
		return
	}
	if size == 0 || size > max {
		err = ErrDocumentSize
		return
	}
	if err = f.Sync(); err != nil {
		return
	}
	if _, err = f.Seek(0, 0); err != nil {
		return
	}
	header := make([]byte, 512)
	n, e := f.Read(header)
	if e != nil && e != io.EOF {
		err = e
		return
	}
	mime = http.DetectContentType(header[:n])
	extension = strings.ToLower(filepath.Ext(original))
	valid := false
	switch extension {
	case ".pdf":
		valid = mime == "application/pdf"
	case ".png":
		valid = mime == "image/png"
	case ".jpg", ".jpeg":
		valid = mime == "image/jpeg"
		extension = ".jpg"
	case ".docx":
		z, e := zip.NewReader(f, size)
		if e == nil && len(z.File) <= 10000 {
			content, document := false, false
			safe := true
			var expanded uint64
			for _, entry := range z.File {
				if entry.UncompressedSize64 > 100*1024*1024 {
					safe = false
					break
				}
				expanded += entry.UncompressedSize64
				if expanded > 100*1024*1024 || strings.Contains(strings.ToLower(entry.Name), "vbaproject") {
					safe = false
					break
				}
				if entry.Name == "[Content_Types].xml" && entry.UncompressedSize64 <= 1024*1024 {
					rd, e := entry.Open()
					if e != nil {
						safe = false
						break
					}
					data, e := io.ReadAll(io.LimitReader(rd, 1024*1024+1))
					rd.Close()
					if e != nil {
						safe = false
						break
					}
					content = strings.Contains(string(data), "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml")
				}
				if entry.Name == "word/document.xml" {
					document = true
				}
			}
			valid = content && document && safe
		}
		mime = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}
	if !valid {
		err = ErrDocumentFile
		return
	}
	hash = hex.EncodeToString(digest.Sum(nil))
	err = f.Close()
	return
}
func PublishDocument(root *os.Root, temp, path string) error {
	if err := root.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	in, err := root.Open(temp)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := root.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	// Never replace an existing file. A partial copy remains pending, inaccessible
	// to users and detectable by the reconciliation tool's SHA-256 check.
	_, copyErr := io.Copy(out, in)
	if copyErr == nil {
		copyErr = out.Sync()
	}
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
func VerifyDocument(root *os.Root, path, hash string, size int64) error {
	f, err := root.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Size() != size {
		return ErrDocumentSize
	}
	digest := sha256.New()
	if _, err = io.Copy(digest, f); err != nil {
		return err
	}
	if hex.EncodeToString(digest.Sum(nil)) != hash {
		return errors.New("document checksum mismatch")
	}
	return nil
}
