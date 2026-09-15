package util

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/go-ldap/ldap/v3"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrLDAPUnavailable = errors.New("LDAP unavailable")

type LDAPConfig struct {
	Servers  []string
	Port     int
	Domain   string
	Timeout  time.Duration
	Security string
	CACert   string
}

// LDAPPrincipal uses the AD identity stored in the application's user record.
func LDAPPrincipal(username, domain string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" || strings.ContainsAny(username, "\x00\r\n") {
		return "", ErrInvalidCredentials
	}
	if strings.Contains(username, "\\") || strings.Contains(username, "@") {
		return username, nil
	}
	if domain == "" {
		return "", ErrInvalidCredentials
	}
	return username + "@" + domain, nil
}

func AuthenticateLDAP(cfg LDAPConfig, username, password string) error {
	if password == "" {
		return ErrInvalidCredentials
	}
	principal, err := LDAPPrincipal(username, cfg.Domain)
	if err != nil {
		return err
	}
	if cfg.Timeout <= 0 || (cfg.Security != "ldaps" && cfg.Security != "starttls") {
		return ErrLDAPUnavailable
	}
	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}
	if cfg.CACert != "" {
		pem, err := os.ReadFile(cfg.CACert)
		if err != nil || !roots.AppendCertsFromPEM(pem) {
			return ErrLDAPUnavailable
		}
	}
	for _, host := range cfg.Servers {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host, RootCAs: roots}
		scheme := "ldaps"
		if cfg.Security == "starttls" {
			scheme = "ldap"
		}
		conn, err := ldap.DialURL(scheme+"://"+net.JoinHostPort(host, strconv.Itoa(cfg.Port)), ldap.DialWithDialer(&net.Dialer{Timeout: cfg.Timeout}), ldap.DialWithTLSConfig(tlsConfig))
		if err != nil {
			continue
		}
		conn.SetTimeout(cfg.Timeout)
		if cfg.Security == "starttls" {
			if err = conn.StartTLS(tlsConfig); err != nil {
				conn.Close()
				continue
			}
		}
		err = conn.Bind(principal, password)
		conn.Close()
		if err == nil {
			return nil
		}
		if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			return ErrInvalidCredentials
		}
	}
	return ErrLDAPUnavailable
}
