package api

import (
	"time"
)

// loginUserRequest godoc
type loginUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// loginUserResponse godoc
type loginUserResponse struct {
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}

// apiErrorResponse godoc
type apiErrorResponse struct {
	Error string `json:"error"`
}

// userResponse godoc
type userResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// apiErrorResponse godoc
type apiResponse struct {
	Message string `json:"message"`
}

type reservationRequest struct {
	Username    string   `json:"username" binding:"required"`
	MovieID     string   `json:"movieId" binding:"required"`
	Date        string   `json:"date" binding:"required"`
	Time        string   `json:"time" binding:"required"`
	Hall        string   `json:"hall" binding:"required"`
	ReservSeats []string `json:"reservSeats" binding:"required"`
}
