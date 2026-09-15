package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
)

// Request za Insert / Update
type UgoEvidRequest struct {
	SapUgovorID  int    `json:"sap_ugovor_id" binding:"required"`
	UgoOrgID     int    `json:"ugo_org_id" binding:"required"`
	Ime          string `json:"ime" binding:"required"`
	Telefon      string `json:"telefon"`
	Email        string `json:"email"`
	Status       string `json:"status"`
	UgoDobLiceID int    `json:"ugo_dob_lice_id"`
}

type UgoEvidResponse struct {
	Data *models.UgoEvid `json:"data"`
}

type UgoEvidListResponse struct {
	Total int               `json:"total"`
	Items []*models.UgoEvid `json:"items"`
}

func (server *Server) GetUgoEvid(ctx *gin.Context) {
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

	evid, err := server.store.GetUgoEvidById(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if evid == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	ctx.JSON(http.StatusOK, UgoEvidResponse{Data: evid})
}

type ListUgoEvidRequest struct {
	Filter   string `form:"filter"`
	PageID   int32  `form:"page_id" binding:"required,min=1"`
	PageSize int32  `form:"page_size" binding:"required,min=5,max=100"`
}

func (server *Server) ListUgoEvid(ctx *gin.Context) {
	var req ListUgoEvidRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := int(req.PageSize)
	offset := int((req.PageID - 1) * req.PageSize)

	items, count, err := server.store.GetUgoEvidPaged(ctx, offset, limit, req.Filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rsp := &UgoEvidListResponse{
		Total: count,
		Items: items,
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) InsertUgoEvid(ctx *gin.Context) {
	var req UgoEvidRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	evid := &models.UgoEvid{
		SapUgovor:  models.SapUgovor{ID: req.SapUgovorID},
		UgoOrg:     models.UgoOrg{ID: req.UgoOrgID},
		Ime:        req.Ime,
		Telefon:    req.Telefon,
		Email:      req.Email,
		Status:     req.Status,
		UgoDobLice: models.UgoDobLice{ID: req.UgoDobLiceID},
	}

	evid, err := server.store.InsertUgoEvid(ctx, evid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, UgoEvidResponse{Data: evid})
}

func (server *Server) UpdateUgoEvid(ctx *gin.Context) {
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

	var req UgoEvidRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	evid := &models.UgoEvid{
		ID:         id,
		SapUgovor:  models.SapUgovor{ID: req.SapUgovorID},
		UgoOrg:     models.UgoOrg{ID: req.UgoOrgID},
		Ime:        req.Ime,
		Telefon:    req.Telefon,
		Email:      req.Email,
		Status:     req.Status,
		UgoDobLice: models.UgoDobLice{ID: req.UgoDobLiceID},
	}

	evid, err = server.store.UpdateUgoEvid(ctx, evid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if evid == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	ctx.JSON(http.StatusOK, UgoEvidResponse{Data: evid})
}

func (server *Server) DeleteUgoEvid(ctx *gin.Context) {
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

	err = server.store.DeleteUgoEvidById(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
