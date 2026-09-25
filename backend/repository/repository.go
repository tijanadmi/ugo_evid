package repository

import (
	"context"

	"github.com/tijanadmi/ugo_evid/models"
)

type Store interface {
	GetUserOrganization(ctx context.Context, username, activeStatus string) (int, error)
	GetPartnersPaged(ctx context.Context, orgID, offset, limit int, naziv string) ([]models.Partner, int, error)
	GetUgoEvidProsireniPaged(ctx context.Context, open bool, offset, limit, orgID int) ([]models.UgoEvidProsireni, int, error)
	GetUgoEvidProsireniByID(ctx context.Context, id int) (models.UgoEvidProsireni, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	InsertUser(ctx context.Context, user *models.User) (*models.User, error)

	GetSapUgovoriPaged(ctx context.Context, offset, limit int, dobavljacNaziv string) ([]*models.SapUgovor, int, error)
	GetOdgLicaForUgovor(ctx context.Context, ugovorID int) ([]models.SapOdglica, error)

	GetUgoOrgById(ctx context.Context, id int) (*models.UgoOrg, error)
	GetUgoOrgPaged(ctx context.Context, offset, limit int, filter string) ([]*models.UgoOrg, int, error)
	InsertUgoOrg(ctx context.Context, org *models.UgoOrg) (*models.UgoOrg, error)
	UpdateUgoOrg(ctx context.Context, org *models.UgoOrg) (*models.UgoOrg, error)
	DeleteUgoOrgById(ctx context.Context, id int) error

	GetUgoDobLicaRoleById(ctx context.Context, id int) (*models.UgoDobLicaRola, error)
	GetUgoDobLicaRolePaged(ctx context.Context, offset, limit int, filter string) ([]*models.UgoDobLicaRola, int, error)
	InsertUgoDobLicaRole(ctx context.Context, m *models.UgoDobLicaRola) (*models.UgoDobLicaRola, error)
	UpdateUgoDobLicaRole(ctx context.Context, m *models.UgoDobLicaRola) (*models.UgoDobLicaRola, error)
	DeleteUgoDobLicaRoleById(ctx context.Context, id int) error

	GetUgoDobLiceById(ctx context.Context, id int) (*models.UgoDobLice, error)
	GetUgoDobLicePaged(ctx context.Context, offset, limit int, filter string) ([]*models.UgoDobLice, int, error)
	InsertUgoDobLice(ctx context.Context, m *models.UgoDobLice) (*models.UgoDobLice, error)
	UpdateUgoDobLice(ctx context.Context, m *models.UgoDobLice) (*models.UgoDobLice, error)
	DeleteUgoDobLiceById(ctx context.Context, id int) error

	GetUgoEvidById(ctx context.Context, id int) (*models.UgoEvid, error)
	GetUgoEvidPaged(ctx context.Context, offset, limit int, filter string) ([]*models.UgoEvid, int, error)
	InsertUgoEvid(ctx context.Context, m *models.UgoEvid) (*models.UgoEvid, error)
	UpdateUgoEvid(ctx context.Context, m *models.UgoEvid) (*models.UgoEvid, error)
	DeleteUgoEvidById(ctx context.Context, id int) error
}
