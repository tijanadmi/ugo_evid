package api

import (
	"context"
	"github.com/tijanadmi/ugo_evid/models"
)

// Existing handler tests explicitly grant their fixture user IUD for one OJ.
// Mixed GLE/IUD and calendar separation are exercised in access_test.go.
func fixtureAccess(user, org int, err error) (*models.Access, error) {
	if err != nil {
		return nil, err
	}
	return &models.Access{UserID: user, Organizations: []models.OrganizationAccess{{ID: org, Write: true}}}, nil
}
func (s *prosireniStore) LoadAccess(context.Context, string, string) (*models.Access, error) {
	return fixtureAccess(5, 1, s.orgErr)
}
func (s *registrationStore) LoadAccess(context.Context, string, string) (*models.Access, error) {
	return fixtureAccess(5, 3, s.orgErr)
}
func (s *documentTestStore) LoadAccess(context.Context, string, string) (*models.Access, error) {
	return fixtureAccess(7, 3, nil)
}
func (s *detailStore) LoadAccess(context.Context, string, string) (*models.Access, error) {
	return fixtureAccess(5, 1, nil)
}
func (s *partnerStore) LoadAccess(context.Context, string, string) (*models.Access, error) {
	return fixtureAccess(7, 3, s.orgErr)
}
func (s *contactStore) LoadAccess(context.Context, string, string) (*models.Access, error) {
	return fixtureAccess(7, 3, nil)
}
func (s *maintenanceTestStore) LoadAccess(context.Context, string, string) (*models.Access, error) {
	a, e := fixtureAccess(7, 3, nil)
	a.CalendarAdmin = true
	return a, e
}
