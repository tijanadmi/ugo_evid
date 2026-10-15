package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
)

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
	if e := s.store.SaveTracking(c.Request.Context(), id, org, user, c.Param("kind"), req); e != nil {
		maintenanceError(c, e)
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
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
