package util

import "testing"

func TestLDAPEmptyCredentials(t *testing.T) {
	for _, pair := range [][2]string{{"", "secret"}, {"account", ""}} {
		if err := AuthenticateLDAP(LDAPConfig{}, pair[0], pair[1]); err != ErrInvalidCredentials {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if err := AuthenticateLDAP(LDAPConfig{}, "account", "secret"); err != ErrLDAPUnavailable {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConfigFromEnvironment(t *testing.T) {
	for k, v := range map[string]string{"DB_DRIVER": "oracle", "DB_SOURCE": "oracle://user:pass@host/service", "TOKEN_SYMMETRIC_KEY": "12345678901234567890123456789012", "LDAP_SERVERS": "dc1,dc2", "LDAP_DOMAIN": "example.local", "LDAP_PORT": "389", "LDAP_TIMEOUT": "5s", "ACTIVE_USER_STATUS": "A", "ACCESS_TOKEN_DURATION": "15m", "REFRESH_TOKEN_DURATION": "24h"} {
		t.Setenv(k, v)
	}
	cfg, err := LoadConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.LDAPServers) != 2 || cfg.LDAPServers[1] != "dc2" {
		t.Fatalf("servers: %v", cfg.LDAPServers)
	}
	if cfg.LDAPPort != 389 {
		t.Fatalf("port: %d", cfg.LDAPPort)
	}
}
