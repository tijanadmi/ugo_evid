package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"strings"
	"time"
)

func (s *Server) ExportOpenContracts(c *gin.Context) {
	var req struct {
		Supplier string `form:"dobavljac" binding:"max=200"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(400, gin.H{"error": "Neispravan filter dobavljača."})
		return
	}
	org, _, ok := s.registrationIdentity(c)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	items, err := s.store.ExportOpenContracts(ctx, org, strings.TrimSpace(req.Supplier))
	if err != nil {
		log.Error().Err(err).Msg("cannot export contracts")
		c.JSON(500, gin.H{"error": "Izvoz ugovora trenutno nije dostupan."})
		return
	}
	if len(items) > 10000 {
		c.JSON(422, gin.H{"error": "Izvoz podržava do 10.000 ugovora. Sužite filter dobavljača."})
		return
	}
	data, err := contractsWorkbook(items)
	if err != nil {
		log.Error().Err(err).Msg("cannot create workbook")
		c.JSON(500, gin.H{"error": "Excel fajl nije moguće kreirati."})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Disposition", `attachment; filename="otvoreni-ugovori.xlsx"`)
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
