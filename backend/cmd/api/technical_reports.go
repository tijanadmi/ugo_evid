package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/util"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (s *Server) reportStore(c *gin.Context) (repository.TechnicalReportStore, int, int, bool) {
	id, ok := documentID(c)
	if !ok {
		return nil, 0, 0, false
	}
	user, ok := s.documentAccess(c, id, c.Request.Method != "GET")
	if !ok {
		return nil, 0, 0, false
	}
	store, ok := s.store.(repository.TechnicalReportStore)
	if !ok {
		c.JSON(503, gin.H{"error": "Izveštaji nisu dostupni."})
	}
	return store, id, user, ok
}
func reportError(c *gin.Context, err error) {
	if errors.Is(err, util.ErrTemplate) {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	documentError(c, err)
}

func (s *Server) reportFolder(c *gin.Context, id int, section string) (string, error) {
	contract, err := s.store.GetUgoEvidProsireniByID(c.Request.Context(), id)
	if err != nil {
		return "", err
	}
	folder, err := s.store.DocumentFolder(c.Request.Context(), id, util.DocumentFolderName(id, contract.BrUgovor, contract.PredmetUgovora))
	if err != nil {
		return "", err
	}
	return filepath.Join(folder, section), nil
}
func reportValues(p models.TechnicalReport) map[string]string {
	fmtDate := func(s string) string { d, _ := time.Parse("2006-01-02", s); return d.Format("02.01.2006") }
	return map[string]string{"datum_od": fmtDate(p.Start), "datum_do": fmtDate(p.End), "datum_potpisa": fmtDate(p.Signed), "sysdate": fmtDate(p.Signed), "broj_kvartala": strconv.Itoa(p.Quarter)}
}
func readReportFile(root *os.Root, path string) ([]byte, error) {
	f, e := root.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Size() > 32<<20 {
		return nil, util.ErrDocumentFile
	}
	return io.ReadAll(io.LimitReader(f, 32<<20+1))
}
func writeReportFile(root *os.Root, folder, ext string, data []byte) (string, error) {
	if err := root.MkdirAll(folder, 0700); err != nil {
		return "", err
	}
	name, err := util.RandomDocumentName()
	if err != nil {
		return "", err
	}
	path := filepath.ToSlash(filepath.Join(folder, name+ext))
	f, err := root.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		root.Remove(path)
		return "", err
	}
	return path, nil
}
func (s *Server) ReportTemplates(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	store, id, user, ok := s.reportStore(c)
	if !ok {
		return
	}
	if c.Request.Method == "GET" {
		items, err := store.ListReportTemplates(c.Request.Context(), id)
		if err != nil {
			reportError(c, err)
			return
		}
		if len(items) > 0 {
			root, err := util.DocumentStorage(s.config.DocumentRoot)
			if err != nil {
				reportError(c, err)
				return
			}
			defer root.Close()
			for i := range items {
				data, err := readReportFile(root, items[i].Path)
				if err != nil {
					reportError(c, err)
					return
				}
				items[i].UsesQuarter, err = util.TemplateUsesQuarter(data)
				if err != nil {
					reportError(c, err)
					return
				}
			}
		}
		c.JSON(200, gin.H{"items": items, "pdf_available": strings.TrimSpace(s.config.LibreOfficePath) != ""})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, util.TemplateMaxBytes+65536)
	if err := c.Request.ParseMultipartForm(util.TemplateMaxBytes); err != nil {
		c.JSON(400, gin.H{"error": "Pošaljite DOCX šablon do 10 MB."})
		return
	}
	defer c.Request.MultipartForm.RemoveAll()
	form := c.Request.MultipartForm
	if len(form.File["file"]) != 1 {
		c.JSON(400, gin.H{"error": "Izaberite jedan DOCX šablon."})
		return
	}
	dtype, e1 := strconv.Atoi(c.PostForm("id_tip_doc"))
	expected, e2 := strconv.Atoi(c.PostForm("verzija"))
	if e1 != nil || e2 != nil || dtype < 1 || expected < 0 {
		c.JSON(400, gin.H{"error": "Neispravan tip ili verzija šablona."})
		return
	}
	header := form.File["file"][0]
	name := util.SafeOriginalName(header.Filename)
	if strings.ToLower(filepath.Ext(name)) != ".docx" || len([]rune(name)) > 255 {
		reportError(c, util.ErrTemplate)
		return
	}
	file, err := header.Open()
	if err != nil {
		reportError(c, err)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, util.TemplateMaxBytes+1))
	if err != nil {
		reportError(c, err)
		return
	}
	if _, err = util.RenderReportTemplate(data, reportValues(models.TechnicalReport{Start: "2026-01-01", End: "2026-03-31", Signed: "2026-04-01", Quarter: 1})); err != nil {
		reportError(c, err)
		return
	}
	root, err := util.DocumentStorage(s.config.DocumentRoot)
	if err != nil {
		reportError(c, err)
		return
	}
	defer root.Close()
	folder, err := s.reportFolder(c, id, "sabloni")
	if err != nil {
		reportError(c, err)
		return
	}
	path, err := writeReportFile(root, folder, ".docx", data)
	if err != nil {
		reportError(c, err)
		return
	}
	t := models.ReportTemplate{ContractID: id, TypeID: dtype, Name: name, Path: path}
	// Retain file on ambiguous commit: never leave a committed row without bytes.
	if err = store.SaveReportTemplate(c.Request.Context(), &t, expected, user); err != nil {
		reportError(c, err)
		return
	}
	c.JSON(201, t)
}
func (s *Server) TemplateContent(c *gin.Context) {
	store, id, _, ok := s.reportStore(c)
	if !ok {
		return
	}
	tid, _ := strconv.Atoi(c.Param("template"))
	items, err := store.ListReportTemplates(c.Request.Context(), id)
	if err != nil {
		reportError(c, err)
		return
	}
	for _, t := range items {
		if t.ID == tid {
			s.sendReportFile(c, t.Path, t.Name, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
			return
		}
	}
	c.JSON(404, gin.H{"error": "Šablon nije pronađen."})
}
func (s *Server) TechnicalReports(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 80*time.Second)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	store, id, user, ok := s.reportStore(c)
	if !ok {
		return
	}
	period, _ := strconv.Atoi(c.Param("period"))
	if period < 1 {
		c.JSON(400, gin.H{"error": "Neispravan period."})
		return
	}
	if c.Request.Method == "GET" {
		items, err := store.ListTechnicalReports(c.Request.Context(), id, period)
		if err != nil {
			reportError(c, err)
			return
		}
		c.JSON(200, gin.H{"items": items})
		return
	}
	var req struct {
		Template int   `json:"id_template"`
		Quarter  int   `json:"broj_kvartala"`
		Version  int64 `json:"version,string"`
		PDF      bool  `json:"pdf"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if c.ShouldBindJSON(&req) != nil || req.Template < 1 || req.Version < 1 || req.Quarter < 0 || req.Quarter > 9999 {
		c.JSON(400, gin.H{"error": "Izaberite šablon i kvartal (1–9999)."})
		return
	}
	tracking, err := s.store.GetTracking(c.Request.Context(), id)
	if err != nil {
		reportError(c, err)
		return
	}
	var row *models.TrackingItem
	for i := range tracking.Periods {
		if tracking.Periods[i].ID == period {
			row = &tracking.Periods[i]
			break
		}
	}
	if row == nil {
		c.JSON(404, gin.H{"error": "Period nije pronađen."})
		return
	}
	if row.Version != req.Version {
		reportError(c, repository.ErrDocumentVersion)
		return
	}
	if row.Signed == "" || row.Status == "OTKAZAN" {
		c.JSON(400, gin.H{"error": "Unesite datum potpisa u izmeni perioda. Otkazan period ne može da se generiše."})
		return
	}
	templates, err := store.ListReportTemplates(c.Request.Context(), id)
	if err != nil {
		reportError(c, err)
		return
	}
	var selected *models.ReportTemplate
	for i := range templates {
		if templates[i].TypeID == row.DocumentType {
			selected = &templates[i]
			break
		}
	}
	if selected == nil || selected.ID != req.Template {
		c.JSON(409, gin.H{"error": "Izaberite aktuelni šablon za tip dokumenta ovog perioda."})
		return
	}
	root, err := util.DocumentStorage(s.config.DocumentRoot)
	if err != nil {
		reportError(c, err)
		return
	}
	defer root.Close()
	data, err := readReportFile(root, selected.Path)
	if err != nil {
		reportError(c, err)
		return
	}
	usesQuarter, err := util.TemplateUsesQuarter(data)
	if err != nil {
		reportError(c, err)
		return
	}
	if usesQuarter && req.Quarter < 1 {
		c.JSON(400, gin.H{"error": "Izabrani šablon zahteva broj kvartala (1–9999)."})
		return
	}
	if !usesQuarter {
		req.Quarter = 0
	}
	p := models.TechnicalReport{ContractID: id, PeriodID: period, TemplateID: req.Template, Quarter: req.Quarter, Start: row.Start, End: row.End, Signed: row.Signed}
	docx, err := util.RenderReportTemplate(data, reportValues(p))
	if err != nil {
		reportError(c, err)
		return
	}
	var pdf []byte
	if req.PDF {
		pdf, err = util.ReportPDF(c.Request.Context(), s.config.LibreOfficePath, docx)
		if err != nil {
			c.JSON(503, gin.H{"error": err.Error()})
			return
		}
	}
	folder, err := s.reportFolder(c, id, fmt.Sprintf("izvestaji/%d", period))
	if err != nil {
		reportError(c, err)
		return
	}
	p.DOCX, err = writeReportFile(root, folder, ".docx", docx)
	if err != nil {
		reportError(c, err)
		return
	}
	if req.PDF {
		p.PDF, err = writeReportFile(root, folder, ".pdf", pdf)
		if err != nil {
			root.Remove(p.DOCX)
			reportError(c, err)
			return
		}
		p.HasPDF = true
	}
	if err = store.SaveTechnicalReport(c.Request.Context(), &p, row.Version, user); err != nil {
		reportError(c, err)
		return
	}
	c.JSON(201, p)
}
func (s *Server) TechnicalReportContent(c *gin.Context) {
	store, id, _, ok := s.reportStore(c)
	if !ok {
		return
	}
	period, _ := strconv.Atoi(c.Param("period"))
	report, _ := strconv.Atoi(c.Param("report"))
	items, err := store.ListTechnicalReports(c.Request.Context(), id, period)
	if err != nil {
		reportError(c, err)
		return
	}
	for _, p := range items {
		if p.ID == report {
			ext := c.Query("format")
			path, mimeType := p.DOCX, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
			if ext == "pdf" {
				path = p.PDF
				mimeType = "application/pdf"
			} else if ext != "docx" {
				c.JSON(400, gin.H{"error": "Nepoznat format."})
				return
			}
			if path == "" {
				break
			}
			s.sendReportFile(c, path, fmt.Sprintf("tehnicka-podrska-%d-%d.%s", period, report, ext), mimeType)
			return
		}
	}
	c.JSON(404, gin.H{"error": "Izveštaj nije pronađen."})
}
func (s *Server) sendReportFile(c *gin.Context, path, name, mimeType string) {
	root, err := util.DocumentStorage(s.config.DocumentRoot)
	if err != nil {
		reportError(c, err)
		return
	}
	defer root.Close()
	data, err := readReportFile(root, path)
	if err != nil {
		reportError(c, err)
		return
	}
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(200, mimeType, data)
}
