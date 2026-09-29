package models

import "time"

type EditLock struct {
	Token     string    `json:"lock_token"`
	ExpiresAt time.Time `json:"expires_at"`
}
