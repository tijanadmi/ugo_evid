package util

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Never remove a directory or follow a link supplied through a database path.
// os.Root confines all filesystem operations to DOCUMENT_ROOT.
func RemoveDocument(root *os.Root, path string) error {
	if root == nil || !filepath.IsLocal(path) || strings.Contains(path, `\`) || strings.Contains(path, ":") {
		return errors.New("invalid document deletion path")
	}
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == "" || part == "." || part == ".." {
			return errors.New("invalid document deletion path")
		}
	}
	info, err := root.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("document deletion requires a regular file")
	}
	err = root.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
