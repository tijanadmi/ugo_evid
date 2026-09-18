package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/tijanadmi/ugo_evid/models"
)

func (server *Server) GetUgoEvidProsireni(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id < 1 {
		ctx.JSON(http.StatusBadRequest, apiErrorResponse{Error: "neispravan ID evidencije"})
		return
	}
	item, err := server.store.GetUgoEvidProsireniByID(ctx.Request.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		ctx.JSON(http.StatusNotFound, apiErrorResponse{Error: "ugovor nije pronađen"})
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("cannot get contract details")
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse{Error: "detalji ugovora trenutno nisu dostupni"})
		return
	}
	ctx.JSON(http.StatusOK, item)
}

type listUgoEvidProsireniRequest struct {
	OrgID    int   `form:"id_ugo_org,default=0" binding:"min=0"`
	PageID   int32 `form:"page_id,default=1" binding:"min=1"`
	PageSize int32 `form:"page_size,default=20" binding:"min=1,max=100"`
}

type listUgoEvidProsireniResponse struct {
	Total int                       `json:"total"`
	Items []models.UgoEvidProsireni `json:"items"`
}

func (server *Server) ListOtvoreniUgovori(ctx *gin.Context)  { server.listEvidProsireni(ctx, true) }
func (server *Server) ListZatvoreniUgovori(ctx *gin.Context) { server.listEvidProsireni(ctx, false) }

func (server *Server) listEvidProsireni(ctx *gin.Context, open bool) {
	var req listUgoEvidProsireniRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, apiErrorResponse{Error: err.Error()})
		return
	}
	offset := (int(req.PageID) - 1) * int(req.PageSize)
	items, total, err := server.store.GetUgoEvidProsireniPaged(ctx.Request.Context(), open, offset, int(req.PageSize), req.OrgID)
	if err != nil {
		log.Error().Err(err).Msg("cannot list UGO_EVID_PROSIRENI_V")
		ctx.JSON(http.StatusInternalServerError, apiErrorResponse{Error: "pregled ugovora trenutno nije dostupan"})
		return
	}
	if items == nil {
		items = []models.UgoEvidProsireni{}
	}
	ctx.JSON(http.StatusOK, listUgoEvidProsireniResponse{Total: total, Items: items})
}
