package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
)

type registrationStore struct {
	repository.Store
	orgErr           error
	failure          error
	calls, org, user int
	req              models.RegisterContract
}

func (s *registrationStore) GetUserByUsername(context.Context, string) (*models.User, error) {
	return &models.User{ID: 5, Status: "A"}, nil
}
func (s *registrationStore) GetUserOrganization(context.Context, string, string) (int, error) {
	return 3, s.orgErr
}
func (s *registrationStore) RegisterContract(_ context.Context, r models.RegisterContract, org, user int) (*models.UgoEvid, error) {
	s.calls++
	s.org = org
	s.user = user
	s.req = r
	return &models.UgoEvid{ID: 55}, s.failure
}

func (s *registrationStore) GetUnregisteredContracts(_ context.Context, org, offset, limit int, filter string) ([]models.ContractChoice, int, error) {
	s.org = org
	s.calls++
	return nil, 0, s.failure
}
func (s *registrationStore) PrepareContractRegistration(_ context.Context, id, org int) (*models.ContractPreparation, error) {
	s.org = org
	s.calls++
	return &models.ContractPreparation{}, s.failure
}
func TestRegistrationReadsUseAuthenticatedOrganization(t *testing.T) {
	for _, path := range []string{"/sapugovori/neevidentirani?id_ugo_org=3", "/sapugovori/101/priprema?id_ugo_org=3"} {
		store := &registrationStore{}
		server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, store)
		if err != nil {
			t.Fatal(err)
		}
		access, _, _ := server.tokenMaker.CreateToken("ad.user", "user", time.Minute)
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+access)
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)
		if rec.Code != 200 || store.calls != 1 || store.org != 3 {
			t.Fatalf("untrusted scope: %d org=%d", rec.Code, store.org)
		}
		if strings.Contains(path, "neevidentirani") {
			var response struct {
				Org   int                     `json:"id_ugo_org"`
				Rule  string                  `json:"selection_rule"`
				Items []models.ContractChoice `json:"items"`
				Total int                     `json:"total"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Org != 3 || response.Rule != "open-nosilac-nab-skr-v1" || response.Items == nil || response.Total != 0 {
				t.Fatal("missing scope metadata or broken empty response")
			}
		}
	}
}

func TestRegisterContractAPI(t *testing.T) {
	existing := `{"id_sap_ugovor":101,"id_tip_ugovora":1,"id_ugo_dob_lica":7,"contact_version":"2","id_ugo_org":999,"user_id":999,"id_sap_dobavljac":999}`
	for _, tc := range []struct {
		name, body      string
		orgErr, failure error
		status, calls   int
	}{
		{"missing type", strings.Replace(existing, `,"id_tip_ugovora":1`, "", 1), nil, nil, 400, 0},
		{"existing contact", existing, nil, nil, 201, 1},
		{"new contact", `{"id_sap_ugovor":101,"id_tip_ugovora":1,"novo_lice":{"ime":" Ana ","telefon":"011","email":"ana@example.test"}}`, nil, nil, 201, 1},
		{"missing phone", `{"id_sap_ugovor":101,"id_tip_ugovora":1,"novo_lice":{"ime":"Ana","email":"ana@example.test"}}`, nil, nil, 400, 0},
		{"bad email", `{"id_sap_ugovor":101,"id_tip_ugovora":1,"novo_lice":{"ime":"Ana","telefon":"011","email":"invalid"}}`, nil, nil, 400, 0},
		{"no organization", existing, repository.ErrUserOrganization, nil, 403, 0},
		{"duplicate", existing, nil, repository.ErrContractRegistered, 409, 1},
		{"closed or different procurement owner", existing, nil, repository.ErrContractUnavailable, 403, 1},
		{"active editor", existing, nil, repository.ErrContactLocked, 423, 1},
		{"multiple active SLM", existing, nil, repository.ErrContractContacts, 409, 1},
		{"legacy bypass rejected", `{"sap_ugovor_id":101,"ugo_org_id":999,"ime":"Fake"}`, nil, nil, 400, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &registrationStore{orgErr: tc.orgErr, failure: tc.failure}
			server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, store)
			if err != nil {
				t.Fatal(err)
			}
			access, _, _ := server.tokenMaker.CreateToken("ad.user", "user", time.Minute)
			req := httptest.NewRequest("POST", "/ugo_evid", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+access)
			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, req)
			if rec.Code != tc.status || store.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", rec.Code, store.calls, rec.Body.String())
			}
			if store.calls > 0 && (store.org != 3 || store.user != 5) {
				t.Fatal("identity must be resolved by server")
			}
		})
	}
}
