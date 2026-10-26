package util

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentAndHolidayFileConfig(t *testing.T) {
	for k, v := range map[string]string{"DB_DRIVER": "oracle", "DB_SOURCE": "oracle://user:pass@host/service", "TOKEN_SYMMETRIC_KEY": strings.Repeat("k", 32), "LDAP_SERVERS": "dc1", "LDAP_DOMAIN": "example.test", "DOCUMENT_ROOT": "", "HOLIDAYS_API_URL": ""} {
		t.Setenv(k, v)
	}
	dir := t.TempDir()
	for _, tc := range []struct{ root, url, want string }{
		{"/srv/ugo/documents", "https://example.test/holidays/RS", ""},
		{"", "https://example.test/RS/", "DOCUMENT_ROOT"},
		{"/srv/docs", "", "HOLIDAYS_API_URL"},
		{"/srv/docs", "file:///tmp/data", "HOLIDAYS_API_URL"},
		{"/srv/docs", "https://user:secret@example.test/RS/", "HOLIDAYS_API_URL"},
	} {
		if err := os.WriteFile(filepath.Join(dir, "app.env"), []byte("DOCUMENT_ROOT="+tc.root+"\nHOLIDAYS_API_URL="+tc.url+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadConfig(dir)
		if tc.want != "" {
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %s, got %v", tc.want, err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if cfg.DocumentRoot != tc.root || cfg.HolidaysAPIURL != tc.url+"/" {
			t.Fatalf("file settings not applied")
		}
	}
}
