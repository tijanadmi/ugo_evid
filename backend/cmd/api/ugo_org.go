package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
)

// Request za Insert i Update
type UgoOrgRequest struct {
	Sifra    string `json:"sifra" binding:"required"`
	SifraCir string `json:"sifra_cir"`
	Naziv    string `json:"naziv" binding:"required"`
	NazivCir string `json:"naziv_cir"`
	Status   string `json:"status"`
}

// Response wrapper
type UgoOrgResponse struct {
	Data *models.UgoOrg `json:"data"`
}

// Response za listu
type UgoOrgListResponse struct {
	Total int              `json:"total"`
	Items []*models.UgoOrg `json:"items"`
}

func (server *Server) GetUgoOrg(ctx *gin.Context) {
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

	org, err := server.store.GetUgoOrgById(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if org == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	ctx.JSON(http.StatusOK, UgoOrgResponse{Data: org})
}

type ListUgoOrgRequest struct {
	Filter   string `form:"filter"` // deo naziva ili sifra, nije obavezno
	PageID   int32  `form:"page_id" binding:"required,min=1"`
	PageSize int32  `form:"page_size" binding:"required,min=5,max=100"`
}

func (server *Server) ListUgoOrg(ctx *gin.Context) {
	var req ListUgoOrgRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Izračunaj offset za paginaciju
	limit := int(req.PageSize)
	offset := int((req.PageID - 1) * req.PageSize)

	// Poziv repo funkcije koja vraća slice i ukupan broj
	orgs, count, err := server.store.GetUgoOrgPaged(ctx, offset, limit, req.Filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Formiramo response
	rsp := &UgoOrgListResponse{
		Total: count,
		Items: orgs,
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) InsertUgoOrg(ctx *gin.Context) {
	var req UgoOrgRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	org := &models.UgoOrg{
		Sifra:    req.Sifra,
		SifraCir: req.SifraCir,
		Naziv:    req.Naziv,
		NazivCir: req.NazivCir,
		Status:   req.Status,
	}

	org, err := server.store.InsertUgoOrg(ctx, org)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, UgoOrgResponse{Data: org})
}

func (server *Server) UpdateUgoOrg(ctx *gin.Context) {
	var req UgoOrgRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	org := &models.UgoOrg{
		Sifra:    req.Sifra,
		SifraCir: req.SifraCir,
		Naziv:    req.Naziv,
		NazivCir: req.NazivCir,
		Status:   req.Status,
	}

	org, err := server.store.UpdateUgoOrg(ctx, org)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if org == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	ctx.JSON(http.StatusOK, UgoOrgResponse{Data: org})
}
func (server *Server) DeleteUgoOrg(ctx *gin.Context) {
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

	err = server.store.DeleteUgoOrgById(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
