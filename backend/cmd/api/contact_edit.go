package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/token"
)

func contactError(ctx *gin.Context, err error) {
	status, message := 500, "Operacija nad licem trenutno nije dostupna."
	switch {
	case strings.Contains(err.Error(), "ORA-00001") && strings.Contains(strings.ToUpper(err.Error()), "UQ_UGO_DOB_LICA_SLM_A"):
		status, message = 409, "Dobavljač već ima aktivno lice sa rolom 1."
	case errors.Is(err, sql.ErrNoRows):
		status, message = 404, "Lice nije pronađeno."
	case errors.Is(err, repository.ErrUserOrganization):
		status, message = 403, "Korisniku mora biti dodeljena jedna aktivna organizaciona jedinica."
	case errors.Is(err, repository.ErrContactLocked):
		status, message = 423, err.Error()
	case errors.Is(err, repository.ErrContactLease), errors.Is(err, repository.ErrContactVersion), errors.Is(err, repository.ErrContactLinked):
		status, message = 409, err.Error()
	default:
		log.Error().Err(err).Msg("contact operation failed")
	}
	ctx.JSON(status, gin.H{"error": message})
}

func (server *Server) contactAccess(ctx *gin.Context, supplierID int) (int, bool) {
	username := ctx.MustGet(authorizationPayloadKey).(*token.Payload).Username
	user, err := server.store.GetUserByUsername(ctx.Request.Context(), username)
	if err != nil {
		contactError(ctx, err)
		return 0, false
	}
	if user == nil || user.ID < 1 || user.Status != server.config.ActiveUserStatus {
		ctx.JSON(403, gin.H{"error": "Nemate pravo uređivanja partnera."})
		return 0, false
	}
	orgID, err := server.store.GetUserOrganization(ctx.Request.Context(), username, server.config.ActiveUserStatus)
	if err != nil {
		contactError(ctx, err)
		return 0, false
	}
	allowed, err := server.store.CanManagePartner(ctx.Request.Context(), orgID, supplierID)
	if err != nil {
		contactError(ctx, err)
		return 0, false
	}
	if !allowed {
		ctx.JSON(403, gin.H{"error": "Partner ne pripada vašoj organizacionoj jedinici."})
		return 0, false
	}
	return user.ID, true
}

func (server *Server) accessibleContact(ctx *gin.Context) (*models.UgoDobLice, int, bool) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id < 1 {
		ctx.JSON(400, gin.H{"error": "ID mora biti pozitivan broj."})
		return nil, 0, false
	}
	m, err := server.store.GetUgoDobLiceById(ctx.Request.Context(), id)
	if err != nil {
		contactError(ctx, err)
		return nil, 0, false
	}
	if m == nil {
		contactError(ctx, sql.ErrNoRows)
		return nil, 0, false
	}
	userID, ok := server.contactAccess(ctx, m.SapDobavljac.ID)
	return m, userID, ok
}

func validLeaseInput(ctx *gin.Context, version int64, token string) bool {
	if version < 1 || len(token) != 64 {
		ctx.JSON(400, gin.H{"error": "Verzija i token zaključavanja su obavezni."})
		return false
	}
	return true
}

func (server *Server) ContactLock(ctx *gin.Context) {
	action, lockToken := "acquire", ""
	if ctx.Request.Method != http.MethodPost {
		var req struct {
			Token string `json:"lock_token" binding:"required,len=64"`
		}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(400, gin.H{"error": "Token zaključavanja je obavezan."})
			return
		}
		lockToken, action = req.Token, "renew"
		if ctx.Request.Method == http.MethodDelete {
			action = "release"
		}
	}
	m, userID, ok := server.accessibleContact(ctx)
	if !ok {
		return
	}
	lease, err := server.store.ContactLock(ctx.Request.Context(), m.ID, userID, action, lockToken)
	if err != nil {
		contactError(ctx, err)
		return
	}
	ctx.JSON(200, lease)
}
