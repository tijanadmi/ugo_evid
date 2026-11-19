package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
)

func (s *Server) PreviewRequestDeadline(c *gin.Context) {
	id, ok := documentID(c)
	if !ok {
		return
	}
	if _, ok = s.documentAccess(c, id, true); !ok {
		return
	}
	var req struct {
		Start string `form:"datum_zahteva" binding:"required"`
		Days  int    `form:"rok_radnih_dana" binding:"min=1,max=3660"`
	}
	if c.ShouldBindQuery(&req) != nil {
		c.JSON(400, gin.H{"error": "Unesite datum zahteva i broj radnih dana."})
		return
	}
	store, ok := s.store.(interface {
		PreviewRequestDeadline(context.Context, string, int) (string, error)
	})
	if !ok {
		c.JSON(500, gin.H{"error": "Obračun roka nije dostupan."})
		return
	}
	deadline, err := store.PreviewRequestDeadline(c.Request.Context(), req.Start, req.Days)
	if err != nil {
		maintenanceError(c, err)
		return
	}
	c.JSON(200, gin.H{"rok": deadline})
}

func (s *Server) GetTracking(c *gin.Context) {
	id, ok := documentID(c)
	if !ok {
		return
	}
	if _, ok = s.documentAccess(c, id, false); !ok {
		return
	}
	data, e := s.store.GetTracking(c.Request.Context(), id)
	if e != nil {
		maintenanceError(c, e)
		return
	}
	c.JSON(200, data)
}
func (s *Server) PreviewPeriods(c *gin.Context) {
	id, ok := documentID(c)
	if !ok {
		return
	}
	if _, ok = s.documentAccess(c, id, true); !ok {
		return
	}
	var req models.PeriodPreview
	if e := c.ShouldBindJSON(&req); e != nil {
		c.JSON(400, gin.H{"error": "Neispravan nacin generisanja."})
		return
	}
	data, e := s.store.PreviewPeriods(c.Request.Context(), id, req.Mode)
	if e != nil {
		maintenanceError(c, e)
		return
	}
	c.JSON(200, gin.H{"items": data})
}
func (s *Server) SaveTracking(c *gin.Context) {
	id, ok := documentID(c)
	if !ok {
		return
	}
	if _, ok = s.documentAccess(c, id, true); !ok {
		return
	}
	org, user, ok := s.registrationIdentity(c)
	if !ok {
		return
	}
	var req models.TrackingChange
	if e := c.ShouldBindJSON(&req); e != nil || !validMaintenanceToken(req.Token) {
		c.JSON(400, gin.H{"error": "Neispravni podaci ili token izmene."})
		return
	}
	kind := c.Param("kind")
	if c.Request.Method == "DELETE" && kind != "periodi" && kind != "zahtevi" {
		c.JSON(400, gin.H{"error": "Nepoznata vrsta evidencije."})
		return
	}
	if c.Request.Method == "DELETE" {
		if kind == "zahtevi" {
			kind = "obrisi-zahtev"
		} else {
			kind = "obrisi-period"
		}
	} else if kind != "periodi" && kind != "zahtevi" {
		c.JSON(400, gin.H{"error": "Nepoznata vrsta evidencije."})
		return
	}
	if e := s.store.SaveTracking(c.Request.Context(), id, org, user, kind, req); e != nil {
		maintenanceError(c, e)
		return
	}
	c.JSON(200, gin.H{"status": "ok", "id": req.Items[0].ID})
}
func (s *Server) ListReminders(c *gin.Context) {
	org, _, ok := s.registrationIdentity(c)
	if !ok {
		return
	}
	data, e := s.store.ListReminders(c.Request.Context(), org)
	if e != nil {
		maintenanceError(c, e)
		return
	}
	c.JSON(200, gin.H{"items": data})
}
