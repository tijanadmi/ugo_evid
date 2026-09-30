package api

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/token"
)

func registrationError(ctx *gin.Context, err error) {
	code, message := 500, "Unos ugovora trenutno nije dostupan."
	switch {
	case errors.Is(err, sql.ErrNoRows):
		code, message = 404, "Izabrani SAP ugovor nije pronađen."
	case errors.Is(err, repository.ErrUserOrganization):
		code, message = 403, "Korisniku mora biti dodeljena jedna aktivna organizaciona jedinica."
	case errors.Is(err, repository.ErrContractRegistered), errors.Is(err, repository.ErrContractContacts), errors.Is(err, repository.ErrContractContactChanged):
		code, message = 409, err.Error()
	case errors.Is(err, repository.ErrContractContactRequired), errors.Is(err, repository.ErrContractSupplier):
		code, message = 400, err.Error()
	case errors.Is(err, repository.ErrRegistrationBusy), errors.Is(err, repository.ErrContactLocked):
		code, message = 423, err.Error()
	case errors.Is(err, context.DeadlineExceeded):
		code, message = 409, "Obrada je trajala predugo. Proverite da li je ugovor evidentiran pre novog pokušaja."
	default:
		log.Error().Err(err).Msg("contract registration failed")
	}
	ctx.JSON(code, gin.H{"error": message})
}

func (server *Server) registrationIdentity(ctx *gin.Context) (int, int, bool) {
	username := ctx.MustGet(authorizationPayloadKey).(*token.Payload).Username
	user, err := server.store.GetUserByUsername(ctx.Request.Context(), username)
	if err != nil {
		registrationError(ctx, err)
		return 0, 0, false
	}
	if user == nil || user.ID < 1 || user.Status != server.config.ActiveUserStatus {
		ctx.JSON(403, gin.H{"error": "Nemate pravo unosa ugovora."})
		return 0, 0, false
	}
	orgID, err := server.store.GetUserOrganization(ctx.Request.Context(), username, server.config.ActiveUserStatus)
	if err != nil {
		registrationError(ctx, err)
		return 0, 0, false
	}
	return orgID, user.ID, true
}

func (server *Server) ListUnregisteredContracts(ctx *gin.Context) {
	var req struct {
		Filter string `form:"filter" binding:"max=200"`
		Page   int    `form:"page_id,default=1" binding:"min=1,max=1000000"`
		Size   int    `form:"page_size,default=20" binding:"min=5,max=100"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(400, gin.H{"error": "Neispravan filter ili veličina stranice."})
		return
	}
	if _, _, ok := server.registrationIdentity(ctx); !ok {
		return
	}
	items, total, err := server.store.GetUnregisteredContracts(ctx.Request.Context(), (req.Page-1)*req.Size, req.Size, req.Filter)
	if err != nil {
		registrationError(ctx, err)
		return
	}
	if items == nil {
		items = []models.ContractChoice{}
	}
	ctx.JSON(200, gin.H{"items": items, "total": total})
}

func (server *Server) PrepareContractRegistration(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id < 1 {
		ctx.JSON(400, gin.H{"error": "Neispravan ID ugovora."})
		return
	}
	orgID, _, ok := server.registrationIdentity(ctx)
	if !ok {
		return
	}
	result, err := server.store.PrepareContractRegistration(ctx.Request.Context(), id)
	if err != nil {
		registrationError(ctx, err)
		return
	}
	ctx.JSON(200, gin.H{"contract": result.Contract, "contact": result.Contact, "id_ugo_org": orgID})
}

func (server *Server) InsertUgoEvid(ctx *gin.Context) {
	var req models.RegisterContract
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": "Proverite izabrani ugovor i podatke kontakt lica."})
		return
	}
	if req.Contact != nil {
		p := req.Contact
		p.Ime, p.RadnoMesto, p.Telefon, p.Email = strings.TrimSpace(p.Ime), strings.TrimSpace(p.RadnoMesto), strings.TrimSpace(p.Telefon), strings.TrimSpace(p.Email)
		address, err := mail.ParseAddress(p.Email)
		if p.Ime == "" || p.Telefon == "" || p.Email == "" || len(p.Ime) > 100 || len(p.RadnoMesto) > 200 || len(p.Telefon) > 100 || len(p.Email) > 100 || err != nil || address.Address != p.Email {
			ctx.JSON(400, gin.H{"error": "Ime, telefon i ispravan email su obavezni. Proverite i dužinu unetih podataka."})
			return
		}
		if req.ContactID != 0 || req.ContactVersion != 0 {
			ctx.JSON(400, gin.H{"error": "Pošaljite postojeće lice ili podatke novog lica."})
			return
		}
	} else if req.ContactID < 1 || req.ContactVersion < 1 {
		ctx.JSON(400, gin.H{"error": "Izaberite kontakt lice ili unesite novo lice."})
		return
	}
	orgID, userID, ok := server.registrationIdentity(ctx)
	if !ok {
		return
	}
	result, err := server.store.RegisterContract(ctx.Request.Context(), req, orgID, userID)
	if err != nil {
		registrationError(ctx, err)
		return
	}
	ctx.JSON(201, UgoEvidResponse{Data: result})
}
