package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
)

type UgoDobLiceRequest struct {
	IDSapDobavljac   int    `json:"id_sap_dobavljac" binding:"required"`
	Ime              string `json:"ime" binding:"required"`
	RadnoMesto       string `json:"radno_mesto"`
	Telefon          string `json:"telefon"`
	Email            string `json:"email"`
	IDUgoDobLicaRola int    `json:"id_ugo_dob_lica_rola"`
	Status           string `json:"status"`
}

type UgoDobLiceResponse struct {
	Data *models.UgoDobLice `json:"data"`
}

type UgoDobLiceListResponse struct {
	Total int                  `json:"total"`
	Items []*models.UgoDobLice `json:"items"`
}

func (server *Server) GetUgoDobLice(ctx *gin.Context) {
	idStr := ctx.Param("id")
	if idStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id je obavezan"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id mora biti broj"})
		return
	}

	lice, err := server.store.GetUgoDobLiceById(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if lice == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	ctx.JSON(http.StatusOK, UgoDobLiceResponse{Data: lice})
}

type ListUgoDobLiceRequest struct {
	Filter   string `form:"filter"`
	PageID   int32  `form:"page_id" binding:"required,min=1"`
	PageSize int32  `form:"page_size" binding:"required,min=5,max=200"`
}

func (server *Server) ListUgoDobLice(ctx *gin.Context) {
	var req ListUgoDobLiceRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := int(req.PageSize)
	offset := int((req.PageID - 1) * req.PageSize)

	items, count, err := server.store.GetUgoDobLicePaged(ctx, offset, limit, req.Filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rsp := &UgoDobLiceListResponse{
		Total: count,
		Items: items,
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) InsertUgoDobLice(ctx *gin.Context) {
	var req UgoDobLiceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lice := &models.UgoDobLice{
		SapDobavljac: models.SapDobavljac{ID: req.IDSapDobavljac},
		Ime:          req.Ime,
		RadnoMesto:   req.RadnoMesto,
		Telefon:      req.Telefon,
		Email:        req.Email,
		UgoDobLicaRola: models.UgoDobLicaRola{
			ID: req.IDUgoDobLicaRola,
		},
		Status: req.Status,
	}

	lice, err := server.store.InsertUgoDobLice(ctx, lice)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, UgoDobLiceResponse{Data: lice})
}

func (server *Server) UpdateUgoDobLice(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id mora biti broj"})
		return
	}

	var req UgoDobLiceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lice := &models.UgoDobLice{
		ID:           id,
		SapDobavljac: models.SapDobavljac{ID: req.IDSapDobavljac},
		Ime:          req.Ime,
		RadnoMesto:   req.RadnoMesto,
		Telefon:      req.Telefon,
		Email:        req.Email,
		UgoDobLicaRola: models.UgoDobLicaRola{
			ID: req.IDUgoDobLicaRola,
		},
		Status: req.Status,
	}

	lice, err = server.store.UpdateUgoDobLice(ctx, lice)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if lice == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	ctx.JSON(http.StatusOK, UgoDobLiceResponse{Data: lice})
}

func (server *Server) DeleteUgoDobLice(ctx *gin.Context) {
	idStr := ctx.Param("id")
	if idStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id je obavezan"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id mora biti broj"})
		return
	}

	err = server.store.DeleteUgoDobLiceById(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
