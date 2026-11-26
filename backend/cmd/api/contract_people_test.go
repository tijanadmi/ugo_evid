package api

import (
	"context"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/util"
	"strings"
	"testing"
)

type peopleTestStore struct {
	documentTestStore
	changes int
}

func (s *peopleTestStore) ListContractPeople(context.Context, int) ([]models.ContractPerson, error) {
	return []models.ContractPerson{}, nil
}
func (s *peopleTestStore) ContractPeopleCandidates(context.Context, int) ([]models.ContractPerson, error) {
	return []models.ContractPerson{}, nil
}
func (s *peopleTestStore) ContactEngagements(context.Context, int) ([]models.ContactEngagement, error) {
	return []models.ContactEngagement{}, nil
}
func (s *peopleTestStore) ChangeContractPerson(context.Context, int, int, models.ContractPersonChange, bool) error {
	s.changes++
	return nil
}

func TestContractPeopleAccess(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		store := &peopleTestStore{}
		store.foreign = foreign
		server, err := NewServer(util.Config{TokenSymmetricKey: strings.Repeat("k", 32), ActiveUserStatus: "A"}, store)
		if err != nil {
			t.Fatal(err)
		}
		rec := docRequest(t, server, "POST", "/ugo_evid/9/lica", strings.NewReader(`{"id_ugo_dob_lica":8,"id_ugo_dob_lica_rola":1}`), "application/json")
		if foreign {
			if rec.Code != 403 || store.changes != 0 {
				t.Fatal("foreign contract changed")
			}
		} else if rec.Code != 200 || store.changes != 1 {
			t.Fatalf("%d %s", rec.Code, rec.Body.String())
		}
	}
}
