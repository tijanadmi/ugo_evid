package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
)

// Request za Insert i Update
type UgoDobLicaRolaRequest struct {
	Naziv  string `json:"naziv" binding:"required"`
	Status string `json:"status"`
}

// Response wrapper
type UgoDobLicaRolaResponse struct {
	Data *models.UgoDobLicaRola `json:"data"`
}

// Response za listu
type UgoDobLicaRolaListResponse struct {
	Total int                      `json:"total"`
	Items []*models.UgoDobLicaRola `json:"items"`
}

func (server *Server) GetUgoDobLicaRolaById(ctx *gin.Context) {
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

	rola, err := server.store.GetUgoDobLicaRoleById(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if rola == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	ctx.JSON(http.StatusOK, UgoDobLicaRolaResponse{Data: rola})
}

type ListUgoDobLicaRolaRequest struct {
	Filter   string `form:"filter"`
	PageID   int32  `form:"page_id" binding:"required,min=1"`
	PageSize int32  `form:"page_size" binding:"required,min=5,max=100"`
}

func (server *Server) ListUgoDobLicaRola(ctx *gin.Context) {
	var req ListUgoDobLicaRolaRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := int(req.PageSize)
	offset := int((req.PageID - 1) * req.PageSize)

	items, count, err := server.store.GetUgoDobLicaRolePaged(ctx, offset, limit, req.Filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rsp := &UgoDobLicaRolaListResponse{
		Total: count,
		Items: items,
	}

	ctx.JSON(http.StatusOK, rsp)
}

func (server *Server) InsertUgoDobLicaRola(ctx *gin.Context) {
	var req UgoDobLicaRolaRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rola := &models.UgoDobLicaRola{
		Naziv:  req.Naziv,
		Status: req.Status,
	}

	rola, err := server.store.InsertUgoDobLicaRole(ctx, rola)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, UgoDobLicaRolaResponse{Data: rola})
}

func (server *Server) UpdateUgoDobLicaRola(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id mora biti broj"})
		return
	}

	var req UgoDobLicaRolaRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rola := &models.UgoDobLicaRola{
		ID:     id,
		Naziv:  req.Naziv,
		Status: req.Status,
	}

	rola, err = server.store.UpdateUgoDobLicaRole(ctx, rola)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if rola == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	ctx.JSON(http.StatusOK, UgoDobLicaRolaResponse{Data: rola})
}

func (server *Server) DeleteUgoDobLicaRola(ctx *gin.Context) {
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

	err = server.store.DeleteUgoDobLicaRoleById(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
