package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
)

func adminError(c *gin.Context, err error) {
	status, message := 500, "Administrativna operacija nije uspela."
	switch {
	case errors.Is(err, repository.ErrAdminForbidden):
		status, message = 403, err.Error()
	case errors.Is(err, repository.ErrAdminInput):
		status, message = 400, err.Error()
	case errors.Is(err, repository.ErrAdminConflict), errors.Is(err, repository.ErrLastAdmin), errors.Is(err, repository.ErrAdminDuplicate), errors.Is(err, repository.ErrNotificationPending):
		status, message = 409, err.Error()
	case errors.Is(err, sql.ErrNoRows):
		status, message = 404, "Podatak nije pronađen."
	case strings.Contains(err.Error(), "ORA-00054"), strings.Contains(err.Error(), "ORA-30006"):
		status, message = 409, "Druga administrativna izmena je u toku. Pokušajte ponovo."
	}
	if status == 500 {
		log.Error().Err(err).Msg("admin operation failed")
	}
	c.JSON(status, gin.H{"error": message})
}
func (s *Server) adminStore(c *gin.Context) (repository.AdminStore, int, bool) {
	a := models.AccessFrom(c.Request.Context())
	if a == nil || !a.Admin {
		c.JSON(403, gin.H{"error": "Nemate administratorska prava."})
		return nil, 0, false
	}
	store, ok := s.store.(repository.AdminStore)
	if !ok {
		c.JSON(500, gin.H{"error": "Administracija nije dostupna."})
		return nil, 0, false
	}
	return store, a.UserID, true
}

type adminPage struct {
	Page   int    `form:"page_id,default=1" binding:"min=1,max=1000000"`
	Size   int    `form:"page_size,default=20" binding:"min=1,max=100"`
	Filter string `form:"filter" binding:"max=200"`
	Status string `form:"status" binding:"max=20"`
}

func (s *Server) AdminUsers(c *gin.Context) {
	store, actor, ok := s.adminStore(c)
	if !ok {
		return
	}
	if c.Request.Method == "GET" {
		if c.Param("id") != "" {
			id, ok := documentID(c)
			if !ok {
				return
			}
			u, err := store.GetAdminUser(c.Request.Context(), id)
			if err != nil {
				adminError(c, err)
				return
			}
			c.JSON(200, u)
			return
		}
		var p adminPage
		if err := c.ShouldBindQuery(&p); err != nil {
			adminError(c, repository.ErrAdminInput)
			return
		}
		items, total, err := store.ListAdminUsers(c.Request.Context(), p.Filter, (p.Page-1)*p.Size, p.Size)
		if err != nil {
			adminError(c, err)
			return
		}
		c.JSON(200, gin.H{"items": items, "total": total})
		return
	}
	var u models.AdminUser
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32*1024)
	if err := c.ShouldBindJSON(&u); err != nil {
		adminError(c, repository.ErrAdminInput)
		return
	}
	u.ID = 0
	if c.Request.Method == "PUT" {
		id, ok := documentID(c)
		if !ok {
			return
		}
		u.ID = id
		if u.Version < 1 {
			adminError(c, repository.ErrAdminInput)
			return
		}
	}
	if err := store.SaveAdminUser(c.Request.Context(), actor, s.config.ActiveUserStatus, &u); err != nil {
		adminError(c, err)
		return
	}
	status := 200
	if c.Request.Method == "POST" {
		status = 201
	}
	c.JSON(status, u)
}
func (s *Server) AdminCatalog(c *gin.Context) {
	store, _, ok := s.adminStore(c)
	if !ok {
		return
	}
	v, err := store.AdminCatalog(c.Request.Context())
	if err != nil {
		adminError(c, err)
		return
	}
	c.JSON(200, v)
}
func (s *Server) AdminNotifications(c *gin.Context) {
	store, actor, ok := s.adminStore(c)
	if !ok {
		return
	}
	if c.Request.Method == "GET" {
		var p adminPage
		if err := c.ShouldBindQuery(&p); err != nil {
			adminError(c, repository.ErrAdminInput)
			return
		}
		items, total, err := store.ListAdminNotifications(c.Request.Context(), p.Status, (p.Page-1)*p.Size, p.Size)
		if err != nil {
			adminError(c, err)
			return
		}
		c.JSON(200, gin.H{"items": items, "total": total})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		adminError(c, repository.ErrAdminInput)
		return
	}
	var req struct {
		Version int64  `json:"version,string" binding:"required,min=1"`
		Action  string `json:"action" binding:"required,oneof=PREUZMI VRATI RESI"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if err := c.ShouldBindJSON(&req); err != nil {
		adminError(c, repository.ErrAdminInput)
		return
	}
	if err := store.ChangeAdminNotification(c.Request.Context(), actor, s.config.ActiveUserStatus, id, req.Version, req.Action); err != nil {
		adminError(c, err)
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}
func (s *Server) AdminAudit(c *gin.Context) {
	store, _, ok := s.adminStore(c)
	if !ok {
		return
	}
	var p adminPage
	if err := c.ShouldBindQuery(&p); err != nil {
		adminError(c, repository.ErrAdminInput)
		return
	}
	items, total, err := store.ListAdminAudit(c.Request.Context(), (p.Page-1)*p.Size, p.Size)
	if err != nil {
		adminError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": items, "total": total})
}
