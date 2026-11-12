package models

import "context"

type OrganizationAccess struct {
	ID    int    `json:"id"`
	Code  string `json:"sifra"`
	Name  string `json:"naziv"`
	Write bool   `json:"can_write"`
}

// Access is resolved from active database assignments on every HTTP request.
type Access struct {
	Admin         bool                 `json:"admin"`
	UserID        int                  `json:"-"`
	Organizations []OrganizationAccess `json:"organizations"`
	CalendarAdmin bool                 `json:"calendar_admin"`
	SelectedOrg   int                  `json:"-"`
	WriteRequest  bool                 `json:"-"`
}

func (a *Access) Allows(org int, write bool) bool {
	for _, o := range a.Organizations {
		if o.ID == org && (!write || o.Write) {
			return true
		}
	}
	return false
}

type accessKey struct{}

func WithAccess(ctx context.Context, a *Access) context.Context {
	return context.WithValue(ctx, accessKey{}, a)
}
func AccessFrom(ctx context.Context) *Access {
	a, _ := ctx.Value(accessKey{}).(*Access)
	return a
}
