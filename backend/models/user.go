package models

import (
	"time"
)

type User struct {
	ID         int    `json:"id,omitempty"`
	ADUsername string `json:"ad_username,omitempty"`
	// Username is the API/token alias for AD_SIFRA, not a separate database column.
	Username         string    `json:"username"`
	FullName         string    `json:"full_name,omitempty"`
	DateOfCreation   time.Time `json:"creation_date,omitempty"`
	DateOfLastUpdate time.Time `json:"update_date,omitempty"`
	Status           string    `json:"status,omitempty"`
	Roles            []int     `json:"roles,omitempty"`
}

// type Role struct {
// 	ID   int `json:"id,omitempty"`
// 	Code string `json:"code"`
// 	Name string `json:"name"`
// }
// type UserRole struct {
// 	ID       int
// 	IdUser   int
// 	IdRole   int
// 	RoleCode string
// 	RoleName string
// }
