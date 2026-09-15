package api

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type loginStore struct {
	repository.Store
	user *models.User
	err  error
}

func (s loginStore) GetUserByUsername(context.Context, string) (*models.User, error) {
	return s.user, s.err
}

func TestADLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	active := &models.User{ID: 1, Username: "app.alias", ADUsername: "ad.account", Status: "A"}
	for _, tc := range []struct {
		name           string
		user           *models.User
		dbErr, ldapErr error
		status         int
		bind           bool
	}{
		{"success", active, nil, nil, 200, true},
		{"missing", nil, nil, nil, 401, false},
		{"inactive", &models.User{Status: "N"}, nil, nil, 401, false},
		{"database failure", nil, errors.New("db unavailable"), nil, 500, false},
		{"bad password", active, nil, util.ErrInvalidCredentials, 401, true},
		{"AD unavailable", active, nil, util.ErrLDAPUnavailable, 503, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := util.Config{TokenSymmetricKey: strings.Repeat("k", 32), AccessTokenDuration: time.Minute, RefreshTokenDuration: time.Hour, ActiveUserStatus: "A"}
			server, err := NewServer(cfg, loginStore{user: tc.user, err: tc.dbErr})
			if err != nil {
				t.Fatal(err)
			}
			bound := false
			server.authenticateLDAP = func(_ util.LDAPConfig, u, p string) error {
				bound = true
				if u != "ad.account" || p != "secret" {
					t.Fatalf("wrong AD credentials passed")
				}
				return tc.ldapErr
			}
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, httptest.NewRequest("POST", "/users/login", strings.NewReader(`{"username":"app.alias","password":"secret"}`)))
			if rec.Code != tc.status || bound != tc.bind {
				t.Fatalf("status=%d bind=%v body=%s", rec.Code, bound, rec.Body.String())
			}
			if rec.Code == 200 {
				var response loginUserResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				payload, err := server.tokenMaker.VerifyToken(response.AccessToken)
				if err != nil || payload.Username != "app.alias" {
					t.Fatal("invalid access token")
				}
				if strings.Contains(rec.Body.String(), "secret") {
					t.Fatal("password leaked")
				}
			}
		})
	}
}

func TestBusinessRoutesRequireAuthentication(t *testing.T) {
	server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32)}, loginStore{})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/sapugovori", "/ugo_org", "/ugo_dob_lica", "/ugo_dob_lica_rola", "/ugo_evid", "/users"} {
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 401 {
			t.Fatalf("%s: %d", path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, httptest.NewRequest("POST", "/users", nil))
	if rec.Code != 404 {
		t.Fatal("public registration is enabled")
	}
}
