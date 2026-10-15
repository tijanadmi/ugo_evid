package api

import (
	"encoding/hex"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
)

func maintenanceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrTrackingInput), errors.Is(err, repository.ErrTrackingCalendar), errors.Is(err, repository.ErrTrackingFunds), errors.Is(err, repository.ErrTrackingDependencies):
		c.JSON(400, gin.H{"error": err.Error()})
	case errors.Is(err, repository.ErrMaintenanceLocked):
		c.JSON(423, gin.H{"error": err.Error()})
	case errors.Is(err, repository.ErrMaintenanceLease), errors.Is(err, repository.ErrMaintenanceVersion), errors.Is(err, repository.ErrMaintenanceConflict):
		c.JSON(409, gin.H{"error": err.Error()})
	case errors.Is(err, repository.ErrMaintenanceInput), errors.Is(err, repository.ErrContractType):
		c.JSON(400, gin.H{"error": err.Error()})
	case errors.Is(err, repository.ErrRegistrationBusy):
		c.JSON(423, gin.H{"error": err.Error()})
	default:
		documentError(c, err)
	}
}
func (s *Server) ListContractTypes(c *gin.Context) {
	if _, _, ok := s.registrationIdentity(c); !ok {
		return
	}
	items, err := s.store.ListContractTypes(c.Request.Context())
	if err != nil {
		maintenanceError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": items})
}
func (s *Server) GetContractSettings(c *gin.Context) {
	id, ok := documentID(c)
	if !ok {
		return
	}
	if _, ok = s.documentAccess(c, id, false); !ok {
		return
	}
	settings, err := s.store.GetContractSettings(c.Request.Context(), id)
	if err != nil {
		maintenanceError(c, err)
		return
	}
	types, err := s.store.ListDocumentTypes(c.Request.Context())
	if err != nil {
		maintenanceError(c, err)
		return
	}
	c.JSON(200, gin.H{"settings": settings, "document_types": types})
}
func (s *Server) SetContractType(c *gin.Context) {
	id, ok := documentID(c)
	if !ok {
		return
	}
	var req struct {
		TypeID  int   `json:"id_tip_ugovora" binding:"required,min=1"`
		Version int64 `json:"version,string" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Izaberite tip ugovora i osvezite podatke."})
		return
	}
	if _, ok = s.documentAccess(c, id, true); !ok {
		return
	}
	org, user, ok := s.registrationIdentity(c)
	if !ok {
		return
	}
	if err := s.store.SetContractType(c.Request.Context(), id, req.TypeID, req.Version, org, user); err != nil {
		maintenanceError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}
func (s *Server) CreateMaintenance(c *gin.Context) {
	id, ok := documentID(c)
	if !ok {
		return
	}
	var req models.Maintenance
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Proverite datume, period i broj dana za podsetnik."})
		return
	}
	if _, ok = s.documentAccess(c, id, true); !ok {
		return
	}
	org, user, ok := s.registrationIdentity(c)
	if !ok {
		return
	}
	var err error
	if c.Request.Method == "PUT" {
		if req.Version < 1 || !validMaintenanceToken(req.Token) {
			c.JSON(400, gin.H{"error": "Nedostaje verzija ili pravo izmene."})
			return
		}
		err = s.store.UpdateMaintenance(c.Request.Context(), id, req, org, user)
	} else {
		err = s.store.CreateMaintenance(c.Request.Context(), id, req, org, user)
	}
	if err != nil {
		maintenanceError(c, err)
		return
	}
	status := 201
	if c.Request.Method == "PUT" {
		status = 200
	}
	c.JSON(status, gin.H{"status": "ok"})
}

func validMaintenanceToken(token string) bool {
	b, err := hex.DecodeString(token)
	return err == nil && len(b) == 32
}
func (s *Server) MaintenanceLock(c *gin.Context) {
	id, ok := documentID(c)
	if !ok {
		return
	}
	action := "acquire"
	token := ""
	if c.Request.Method != "POST" {
		var req struct {
			Token string `json:"lock_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || !validMaintenanceToken(req.Token) {
			c.JSON(400, gin.H{"error": "Neispravan token zakljucavanja."})
			return
		}
		token = req.Token
		action = "renew"
		if c.Request.Method == "DELETE" {
			action = "release"
		}
	}
	if _, ok = s.documentAccess(c, id, action != "release"); !ok {
		return
	}
	org, user, ok := s.registrationIdentity(c)
	if !ok {
		return
	}
	result, err := s.store.MaintenanceLock(c.Request.Context(), id, org, user, action, token)
	if err != nil {
		maintenanceError(c, err)
		return
	}
	c.JSON(200, result)
}
