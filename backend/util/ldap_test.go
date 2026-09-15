package util

import "testing"

func TestLDAPPrincipal(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{" user ", "user@example.local"}, {"user@example.local", "user@example.local"}, {`EXAMPLE\user`, `EXAMPLE\user`},
	} {
		got, err := LDAPPrincipal(tc.input, "example.local")
		if err != nil || got != tc.want {
			t.Fatalf("%q: %q %v", tc.input, got, err)
		}
	}
	if _, err := LDAPPrincipal("", "example.local"); err == nil {
		t.Fatal("empty principal accepted")
	}
	if err := AuthenticateLDAP(LDAPConfig{}, "user", ""); err != ErrInvalidCredentials {
		t.Fatal("empty password accepted")
	}
}

func TestConfigFromEnvironment(t *testing.T) {
	for k, v := range map[string]string{"DB_DRIVER": "oracle", "DB_SOURCE": "oracle://user:pass@host/service", "TOKEN_SYMMETRIC_KEY": "12345678901234567890123456789012", "LDAP_SERVERS": "dc1,dc2", "LDAP_DOMAIN": "example.local", "LDAP_SECURITY": "ldaps", "LDAP_PORT": "636", "LDAP_TIMEOUT": "5s", "ACTIVE_USER_STATUS": "A", "ACCESS_TOKEN_DURATION": "15m", "REFRESH_TOKEN_DURATION": "24h"} {
		t.Setenv(k, v)
	}
	cfg, err := LoadConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.LDAPServers) != 2 || cfg.LDAPServers[1] != "dc2" {
		t.Fatalf("servers: %v", cfg.LDAPServers)
	}
	t.Setenv("LDAP_SECURITY", "plain")
	if _, err := LoadConfig(t.TempDir()); err == nil {
		t.Fatal("unencrypted LDAP accepted")
	}
}
