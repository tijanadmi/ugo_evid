package api

import (
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
)

type UgoDobLiceRequest struct {
	IDSapDobavljac   int    `json:"id_sap_dobavljac" binding:"required,min=1"`
	Ime              string `json:"ime" binding:"required"`
	RadnoMesto       string `json:"radno_mesto"`
	Telefon          string `json:"telefon"`
	Email            string `json:"email"`
	IDUgoDobLicaRola int    `json:"id_ugo_dob_lica_rola"`
	Status           string `json:"status" binding:"required,oneof=A N"`
	Version          int64  `json:"version,string"`
	LockToken        string `json:"lock_token"`
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
	if err != nil || id < 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id mora biti broj"})
		return
	}

	lice, err := server.store.GetUgoDobLiceById(ctx.Request.Context(), id)
	if err != nil {
		contactError(ctx, err)
		return
	}

	if lice == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	if _, ok := server.contactAccess(ctx, lice.SapDobavljac.ID); !ok {
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
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Proverite obavezna polja i format podataka."})
		return
	}

	limit := int(req.PageSize)
	offset := int((req.PageID - 1) * req.PageSize)

	items, count, err := server.store.GetUgoDobLicePaged(ctx.Request.Context(), offset, limit, req.Filter)
	if err != nil {
		contactError(ctx, err)
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
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Proverite obavezna polja i format podataka."})
		return
	}

	if !server.validateContact(ctx, &req) {
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
		Status:  req.Status,
		Version: req.Version,
	}

	if _, ok := server.contactAccess(ctx, req.IDSapDobavljac); !ok {
		return
	}
	lice, err := server.store.InsertUgoDobLice(ctx.Request.Context(), lice)
	if err != nil {
		contactError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, UgoDobLiceResponse{Data: lice})
}

func (server *Server) UpdateUgoDobLice(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "id mora biti broj"})
		return
	}

	var req UgoDobLiceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Proverite obavezna polja i format podataka."})
		return
	}

	if !server.validateContact(ctx, &req) {
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
		Status:  req.Status,
		Version: req.Version,
	}

	if !validLeaseInput(ctx, req.Version, req.LockToken) {
		return
	}
	current, userID, ok := server.accessibleContact(ctx)
	if !ok {
		return
	}
	if current.SapDobavljac.ID != req.IDSapDobavljac {
		ctx.JSON(400, gin.H{"error": "Dobavljač postojećeg lica se ne može menjati."})
		return
	}
	lice, err = server.store.UpdateUgoDobLice(ctx.Request.Context(), lice, userID, req.LockToken)
	if err != nil {
		contactError(ctx, err)
		return
	}

	if lice == nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}

	ctx.JSON(http.StatusOK, UgoDobLiceResponse{Data: lice})
}

func (server *Server) DeleteUgoDobLice(ctx *gin.Context) {
	var req struct {
		Version int64  `json:"version,string"`
		Token   string `json:"lock_token"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": "Neispravan zahtev za brisanje."})
		return
	}
	if !validLeaseInput(ctx, req.Version, req.Token) {
		return
	}
	m, userID, ok := server.accessibleContact(ctx)
	if !ok {
		return
	}
	if err := server.store.DeleteUgoDobLiceById(ctx.Request.Context(), m.ID, req.Version, userID, req.Token); err != nil {
		contactError(ctx, err)
		return
	}
	ctx.JSON(200, gin.H{"status": "deleted"})
}

func (server *Server) validateContact(ctx *gin.Context, req *UgoDobLiceRequest) bool {
	req.Ime, req.RadnoMesto = strings.TrimSpace(req.Ime), strings.TrimSpace(req.RadnoMesto)
	req.Telefon, req.Email = strings.TrimSpace(req.Telefon), strings.TrimSpace(req.Email)
	// Match the BYTE-sized columns in the Oracle schema.
	if req.Ime == "" || len(req.Ime) > 100 || len(req.RadnoMesto) > 200 || len(req.Telefon) > 100 || len(req.Email) > 100 || req.IDUgoDobLicaRola < 0 {
		ctx.JSON(400, gin.H{"error": "Ime je obavezno. Proverite dužinu unetih podataka i izabranu rolu."})
		return false
	}
	if req.Email != "" {
		address, err := mail.ParseAddress(req.Email)
		if err != nil || address.Address != req.Email {
			ctx.JSON(400, gin.H{"error": "Email adresa nije ispravna."})
			return false
		}
	}
	if req.IDUgoDobLicaRola != 0 {
		role, err := server.store.GetUgoDobLicaRoleById(ctx.Request.Context(), req.IDUgoDobLicaRola)
		if err != nil {
			contactError(ctx, err)
			return false
		}
		if role == nil {
			ctx.JSON(400, gin.H{"error": "Izabrana rola ne postoji."})
			return false
		}
	}
	return true
}
