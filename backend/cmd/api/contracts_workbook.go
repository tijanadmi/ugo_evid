package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/tijanadmi/ugo_evid/models"
	"github.com/xuri/excelize/v2"
)

// The first four rows form the report heading; contract records start at row 5.
func contractsWorkbook(items []models.UgoEvidProsireni) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Otvoreni ugovori"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	if err := formatContractsWorkbook(f, sheet, len(items)); err != nil {
		return nil, err
	}
	for i, m := range items {
		active := "NE"
		if strings.TrimSpace(m.OtvorenUG) == "X" {
			active = "DA"
		}
		values := []string{
			m.Godina, active, m.JN, m.PredmetUgovora, m.BrPozPlana,
			m.BrUgovor, m.UgovorDMS, m.Naziv,
			workbookDate(m.PocetakUG), workbookDate(m.KrajUG),
			workbookPerson(m.OdgZap1, m.NazivOdgZap1),
			workbookPerson(m.OdgZap2, m.NazivOdgZap2),
			workbookPerson(m.OdgZap3, m.NazivOdgZap3),
			workbookPerson(m.OdgZap4, m.NazivOdgZap4),
			workbookPerson(m.OdgZap5, m.NazivOdgZap5),
			workbookPerson(m.OdgZap6, m.NazivOdgZap6),
			m.Ime, m.Telefon, m.Email,
		}
		for col, value := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, i+5)
			// Preserve leading zeroes and phone numbers, and never interpret text as formulas.
			if err := f.SetCellStr(sheet, cell, value); err != nil {
				return nil, err
			}
		}
	}
	b, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func workbookDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("02.01.2006")
}

func workbookPerson(code, name string) string {
	if name = strings.TrimSpace(name); name != "" {
		return name
	}
	return strings.TrimSpace(code)
}

func formatContractsWorkbook(f *excelize.File, sheet string, count int) error {
	border := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
	}
	style := func(start, end, color string, bold, italic bool, align string) error {
		id, err := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Family: "Arial", Size: 11, Bold: bold, Italic: italic},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1},
			Border:    border,
			Alignment: &excelize.Alignment{Horizontal: align, Vertical: "center", WrapText: true},
		})
		if err != nil {
			return err
		}
		return f.SetCellStyle(sheet, start, end, id)
	}
	for _, region := range []struct {
		start, end, color string
		bold, italic      bool
		align             string
	}{
		{"A1", "S1", "FCE4D6", true, true, "left"},
		{"A3", "P4", "D9D9D9", true, false, "center"},
		{"Q3", "S4", "FFFF00", true, false, "center"},
	} {
		if err := style(region.start, region.end, region.color, region.bold, region.italic, region.align); err != nil {
			return err
		}
	}
	if count > 0 {
		if err := style("A5", fmt.Sprintf("S%d", count+4), "FFFF00", false, false, "left"); err != nil {
			return err
		}
	}
	for _, heading := range []struct{ start, end, text string }{
		{"A1", "S1", "Za kompanijsku upotrebu"},
		{"A3", "A4", "Godina"},
		{"B3", "B4", "Aktivan\nDA/NE"},
		{"C3", "C4", "Broj nabavke"},
		{"D3", "D4", "Predmet ugovora"},
		{"E3", "E4", "Pozicija"},
		{"F3", "F4", "Ugovor SAP"},
		{"G3", "G4", "Ugovor DMS"},
		{"H3", "H4", "Partner"},
		{"I3", "J3", "Ugovor važi"},
		{"K3", "P3", "Lica za operativno praćenje ugovora"},
		{"Q3", "S3", "Kontakt osoba - Service Level Manager PARTNER"},
	} {
		if err := f.MergeCell(sheet, heading.start, heading.end); err != nil {
			return err
		}
		if err := f.SetCellStr(sheet, heading.start, heading.text); err != nil {
			return err
		}
	}
	for i, label := range []string{"Od", "Do", "Lice 1", "Lice 2", "Lice 3", "Lice 4", "Lice 5", "Lice 6", "Ime i prezime", "Telefon", "Email"} {
		cell, _ := excelize.CoordinatesToCellName(i+9, 4)
		if err := f.SetCellStr(sheet, cell, label); err != nil {
			return err
		}
	}
	for _, column := range []struct {
		start, end string
		width      float64
	}{
		{"A", "B", 11}, {"C", "C", 18}, {"D", "D", 65},
		{"E", "E", 15}, {"F", "F", 19}, {"G", "G", 43}, {"H", "H", 50},
		{"I", "J", 15}, {"K", "P", 26}, {"Q", "Q", 30}, {"R", "R", 23}, {"S", "S", 38},
	} {
		if err := f.SetColWidth(sheet, column.start, column.end, column.width); err != nil {
			return err
		}
	}
	for row, height := range []float64{24, 15, 36, 26} {
		if err := f.SetRowHeight(sheet, row+1, height); err != nil {
			return err
		}
	}
	// Leave data row heights automatic so wrapped long subjects and names remain visible.
	return f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 4, TopLeftCell: "A5", ActivePane: "bottomLeft"})
}
