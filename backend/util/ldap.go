package util

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

type LDAPConfig struct {
	Servers []string
	Port    int
	Domain  string
	Timeout time.Duration
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrLDAPUnavailable    = errors.New("ldap unavailable")
)

func AuthenticateLDAP(cfg LDAPConfig, username, password string) error {

	if username == "" || password == "" {
		return ErrInvalidCredentials
	}

	login := fmt.Sprintf("%s@%s", username, cfg.Domain)

	var lastErr error

	for _, server := range cfg.Servers {

		address := fmt.Sprintf("%s:%d", strings.TrimSpace(server), cfg.Port)

		conn, err := ldap.DialURL(
			"ldap://"+address,
			ldap.DialWithDialer(&net.Dialer{
				Timeout: cfg.Timeout,
			}),
		)

		if err != nil {
			lastErr = err
			continue
		}

		conn.SetTimeout(cfg.Timeout)

		err = conn.Bind(login, password)

		conn.Close()

		if err == nil {
			return nil
		}

		if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			return ErrInvalidCredentials
		}

		lastErr = err
	}

	if lastErr != nil {
		return ErrLDAPUnavailable
	}

	return ErrLDAPUnavailable
}
