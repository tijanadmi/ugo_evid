package api

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/xuri/excelize/v2"
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

func contractsWorkbook(items []models.UgoEvidProsireni) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Otvoreni ugovori"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	headers := []string{"Broj ugovora", "Godina", "Predmet ugovora", "JN", "Dobavljač", "Adresa", "Grad", "Služba", "Početak", "Završetak", "Odgovorna lica", "Kontakt", "Email", "Telefon"}
	write := func(row int, values []string) error {
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			if err := f.SetCellStr(sheet, cell, v); err != nil {
				return err
			}
		}
		return nil
	}
	if err := write(1, headers); err != nil {
		return nil, err
	}
	date := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.Format("02.01.2006")
	}
	for i, m := range items {
		people := []string{}
		for _, p := range [][2]string{{m.OdgZap1, m.NazivOdgZap1}, {m.OdgZap2, m.NazivOdgZap2}, {m.OdgZap3, m.NazivOdgZap3}, {m.OdgZap4, m.NazivOdgZap4}, {m.OdgZap5, m.NazivOdgZap5}, {m.OdgZap6, m.NazivOdgZap6}} {
			parts := []string{}
			for _, v := range p {
				if strings.TrimSpace(v) != "" {
					parts = append(parts, strings.TrimSpace(v))
				}
			}
			if len(parts) > 0 {
				people = append(people, strings.Join(parts, " — "))
			}
		}
		if err := write(i+2, []string{m.BrUgovor, m.Godina, m.PredmetUgovora, m.JN, m.Naziv, m.Adresa, m.Grad, m.Sluzba, date(m.PocetakUG), date(m.KrajUG), strings.Join(people, "\n"), m.Ime, m.Email, m.Telefon}); err != nil {
			return nil, err
		}
	}
	header, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Color: []string{"E7EDF5"}, Pattern: 1}})
	if err != nil {
		return nil, err
	}
	if err = f.SetCellStyle(sheet, "A1", "N1", header); err != nil {
		return nil, err
	}
	if err = f.SetColWidth(sheet, "A", "N", 22); err != nil {
		return nil, err
	}
	if err = f.SetColWidth(sheet, "C", "C", 55); err != nil {
		return nil, err
	}
	if err = f.SetColWidth(sheet, "K", "K", 45); err != nil {
		return nil, err
	}
	if err = f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return nil, err
	}
	if err = f.AutoFilter(sheet, fmt.Sprintf("A1:N%d", len(items)+1), nil); err != nil {
		return nil, err
	}
	b, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
