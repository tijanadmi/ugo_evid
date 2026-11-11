package api

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/token"
)

func (s *Server) accessMiddleware(c *gin.Context) {
	store, ok := s.store.(repository.AccessStore)
	if !ok {
		c.AbortWithStatusJSON(500, gin.H{"error": "Provera prava nije dostupna."})
		return
	}
	username := c.MustGet(authorizationPayloadKey).(*token.Payload).Username
	a, err := store.LoadAccess(c.Request.Context(), username, s.config.ActiveUserStatus)
	if err != nil {
		status := 500
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, repository.ErrUserOrganization) {
			status = 403
		}
		log.Error().Err(err).Msg("cannot load access rights")
		c.AbortWithStatusJSON(status, gin.H{"error": "Prava pristupa nisu dostupna. Obratite se administratoru."})
		return
	}
	path := c.FullPath()
	write := c.Request.Method != "GET" && c.Request.Method != "HEAD"
	a.WriteRequest = write
	c.Request = c.Request.WithContext(models.WithAccess(c.Request.Context(), a))
	deny := func() {
		c.AbortWithStatusJSON(403, gin.H{"error": "Nemate pravo za ovu akciju ili organizacionu jedinicu."})
	}
	if path == "/users/prava" || path == "/users" {
		c.Next()
		return
	}
	if strings.HasPrefix(path, "/radni_kalendar/") {
		if (write || strings.HasSuffix(path, "/praznici")) && !a.CalendarAdmin {
			deny()
			return
		}
		if !a.CalendarAdmin && len(a.Organizations) == 0 {
			deny()
			return
		}
		c.Next()
		return
	}
	if len(a.Organizations) == 0 {
		deny()
		return
	}
	// No current role grants administration of the OJ or contact-role dictionaries.
	if write && (strings.HasPrefix(path, "/ugo_org") || strings.HasPrefix(path, "/ugo_dob_lica_rola")) {
		deny()
		return
	}
	// The old unscoped SAP/contact lists are not used by the portal.
	if path == "/sapugovori" || path == "/ugo_dob_lica" && !write {
		deny()
		return
	}
	if path == "/ugo_org/:id" {
		id, _ := strconv.Atoi(c.Param("id"))
		if !a.Allows(id, false) {
			deny()
			return
		}
	}
	registration := strings.HasPrefix(path, "/sapugovori/") || path == "/ugo_evid" && c.Request.Method == "POST"
	if registration {
		org, err := strconv.Atoi(c.DefaultQuery("id_ugo_org", "0"))
		if err != nil || org < 0 {
			c.AbortWithStatusJSON(400, gin.H{"error": "Neispravna organizaciona jedinica."})
			return
		}
		if org == 0 {
			for _, o := range a.Organizations {
				if o.Write {
					if org != 0 {
						c.AbortWithStatusJSON(400, gin.H{"error": "Izaberite organizacionu jedinicu za unos ugovora."})
						return
					}
					org = o.ID
				}
			}
		}
		if !a.Allows(org, true) {
			deny()
			return
		}
		a.SelectedOrg = org
	} else if c.Param("id") == "" {
		if value := c.Query("id_ugo_org"); value != "" && value != "0" {
			org, err := strconv.Atoi(value)
			if err != nil || org < 1 {
				c.AbortWithStatusJSON(400, gin.H{"error": "Neispravna organizaciona jedinica."})
				return
			}
			if !a.Allows(org, false) {
				deny()
				return
			}
			a.SelectedOrg = org
		}
	}
	if write {
		allowed := false
		for _, o := range a.Organizations {
			allowed = allowed || o.Write
		}
		if !allowed {
			deny()
			return
		}
	}
	c.Next()
}

func (s *Server) UserAccess(c *gin.Context) { c.JSON(200, models.AccessFrom(c.Request.Context())) }

func allowsContract(c *gin.Context, org, legacyOrg int, write bool) bool {
	if a := models.AccessFrom(c.Request.Context()); a != nil {
		return a.Allows(org, write || a.WriteRequest)
	}
	return org == legacyOrg
}
