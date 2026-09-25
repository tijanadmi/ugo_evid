package api

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/token"
	"net/http"
)

func (server *Server) ListMyPartners(ctx *gin.Context) {
	var req struct {
		Naziv    string `form:"naziv" binding:"max=200"`
		PageID   int32  `form:"page_id,default=1" binding:"min=1"`
		PageSize int32  `form:"page_size,default=20" binding:"min=1,max=100"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(400, apiErrorResponse{Error: "neispravna paginacija ili filter (najviše 200 znakova)"})
		return
	}
	payload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	orgID, err := server.store.GetUserOrganization(ctx.Request.Context(), payload.Username, server.config.ActiveUserStatus)
	if errors.Is(err, repository.ErrUserOrganization) {
		ctx.JSON(http.StatusForbidden, apiErrorResponse{Error: "Korisniku mora biti dodeljena jedna aktivna organizaciona jedinica."})
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("cannot resolve partner organization")
		ctx.JSON(500, apiErrorResponse{Error: "Pregled partnera trenutno nije dostupan."})
		return
	}
	items, total, err := server.store.GetPartnersPaged(ctx.Request.Context(), orgID, (int(req.PageID)-1)*int(req.PageSize), int(req.PageSize), req.Naziv)
	if err != nil {
		log.Error().Err(err).Msg("cannot list partners")
		ctx.JSON(500, apiErrorResponse{Error: "Pregled partnera trenutno nije dostupan."})
		return
	}
	if items == nil {
		items = []models.Partner{}
	}
	ctx.JSON(200, gin.H{"items": items, "total": total, "id_ugo_org": orgID})
}
