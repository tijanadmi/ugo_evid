package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
)

type guardedContactStore struct {
	contactStore
	deny      bool
	failure   error
	calls     int
	userID    int
	lockToken string
	action    string
}

func (s *guardedContactStore) CanManagePartner(_ context.Context, orgID, supplierID int) (bool, error) {
	return !s.deny && orgID == 3 && supplierID == 9, nil
}
func (s *guardedContactStore) UpdateUgoDobLice(_ context.Context, m *models.UgoDobLice, userID int, token string) (*models.UgoDobLice, error) {
	s.calls++
	s.userID = userID
	s.lockToken = token
	return m, s.failure
}
func (s *guardedContactStore) DeleteUgoDobLiceById(_ context.Context, id int, version int64, userID int, token string) error {
	s.calls++
	s.userID = userID
	s.lockToken = token
	return s.failure
}
func (s *guardedContactStore) ContactLock(_ context.Context, id, userID int, action, token string) (*models.EditLock, error) {
	s.calls++
	s.userID = userID
	s.lockToken = token
	s.action = action
	return &models.EditLock{Token: strings.Repeat("b", 64)}, s.failure
}

func TestContactMutationAuthorizationAndConflicts(t *testing.T) {
	body := `{"id_sap_dobavljac":9,"ime":"Ana","status":"A","version":"3","lock_token":"` + strings.Repeat("a", 64) + `","user_id":999}`
	for _, tc := range []struct {
		name, method, path, body string
		deny                     bool
		failure                  error
		want, calls              int
	}{
		{"missing lease", "PUT", "/ugo_dob_lica/8", `{"id_sap_dobavljac":9,"ime":"Ana","status":"A"}`, false, nil, 400, 0},
		{"delete missing lease", "DELETE", "/ugo_dob_lica/8", `{}`, false, nil, 400, 0},
		{"foreign partner", "PUT", "/ugo_dob_lica/8", body, true, nil, 403, 0},
		{"foreign partner insert", "POST", "/ugo_dob_lica", body, true, nil, 403, 0},
		{"foreign partner delete", "DELETE", "/ugo_dob_lica/8", body, true, nil, 403, 0},
		{"foreign partner acquire", "POST", "/ugo_dob_lica/8/lock", `{}`, true, nil, 403, 0},
		{"cannot move contact", "PUT", "/ugo_dob_lica/8", strings.Replace(body, `"id_sap_dobavljac":9`, `"id_sap_dobavljac":10`, 1), false, nil, 400, 0},
		{"version conflict", "PUT", "/ugo_dob_lica/8", body, false, repository.ErrContactVersion, 409, 1},
		{"expired token", "PUT", "/ugo_dob_lica/8", body, false, repository.ErrContactLease, 409, 1},
		{"linked contact", "DELETE", "/ugo_dob_lica/8", body, false, repository.ErrContactLinked, 409, 1},
		{"locked", "POST", "/ugo_dob_lica/8/lock", `{}`, false, repository.ErrContactLocked, 423, 1},
		{"successful delete", "DELETE", "/ugo_dob_lica/8", body, false, nil, 200, 1},
		{"renew", "PUT", "/ugo_dob_lica/8/lock", body, false, nil, 200, 1},
		{"release", "DELETE", "/ugo_dob_lica/8/lock", body, false, nil, 200, 1},
		{"blank name", "POST", "/ugo_dob_lica", strings.Replace(body, `"Ana"`, `"   "`, 1), false, nil, 400, 0},
		{"invalid status", "POST", "/ugo_dob_lica", strings.Replace(body, `"status":"A"`, `"status":"Z"`, 1), false, nil, 400, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &guardedContactStore{deny: tc.deny, failure: tc.failure}
			server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, store)
			if err != nil {
				t.Fatal(err)
			}
			token, _, err := server.tokenMaker.CreateToken("ad.account", "user", time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
			if rec.Code != tc.want || store.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", rec.Code, store.calls, rec.Body.String())
			}
			if store.calls > 0 && store.userID != 7 {
				t.Fatal("owner must be resolved from authenticated user, not request")
			}
			if store.calls > 0 && tc.method != "POST" && store.lockToken != strings.Repeat("a", 64) {
				t.Fatal("wrong lease token")
			}
		})
	}
}
