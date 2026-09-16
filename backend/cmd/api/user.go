package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/util"
)

func newUserResponse(user *models.User) userResponse {
	return userResponse{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.DateOfCreation,
	}
}

// Paths Information

// @Summary Provides a JSON Web Token
// @Description Authenticates a user and provides a Paseto/JWT to Authorize API calls
// @ID loginUser
// @Consume json
// @Produce json
// @Param loginUserRequest body loginUserRequest true "User login request"
// @Success 200 {object} loginUserResponse
// @Failure 400 {object} apiErrorResponse
// @Failure 401 {object} apiErrorResponse
// @Router /users/login [post]
func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, apiErrorResponse{Error: err.Error()})
		return
	}

	user, err := server.store.GetUserByUsername(ctx.Request.Context(), strings.TrimSpace(req.Username))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse{Error: "prijava trenutno nije dostupna"})
		return
	}
	if user == nil || user.Status != server.config.ActiveUserStatus || user.ADUsername == "" {
		ctx.JSON(http.StatusUnauthorized, apiErrorResponse{Error: "neuspesna prijava"})
		return
	}
	err = server.authenticateLDAP(util.LDAPConfig{
		Servers: server.config.LDAPServers, Port: server.config.LDAPPort,
		Domain: server.config.LDAPDomain, Timeout: server.config.LDAPTimeout,
	}, user.ADUsername, req.Password)
	if err != nil {
		status := http.StatusUnauthorized
		message := "neuspesna prijava"
		if errors.Is(err, util.ErrLDAPUnavailable) {
			log.Error().Err(err).Msg("AD authentication service failed")
			status = http.StatusServiceUnavailable
			message = "AD servis trenutno nije dostupan"
		}
		ctx.JSON(status, apiErrorResponse{Error: message})
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		/*user.Role,*/
		"user",
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse{Error: err.Error()})
		return
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		//user.Role,
		"user",
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse{Error: err.Error()})
		return
	}

	rsp := loginUserResponse{
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  newUserResponse(user),
	}
	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) getUserByUsername(ctx *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, apiErrorResponse{Error: err.Error()})
		return
	}

	user, err := server.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, user)
}
