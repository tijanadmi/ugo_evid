package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rs/zerolog/log"
	"github.com/tijanadmi/ugo_evid/models"
	"github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/token"
	"github.com/tijanadmi/ugo_evid/util"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

func documentError(ctx *gin.Context, err error) {
	status, message := 500, "Operacija nad dokumentom nije uspela. Osvezite listu pre ponavljanja."
	switch {
	case errors.Is(err, sql.ErrNoRows):
		status, message = 404, "Dokument ili ugovor nije pronadjen."
	case errors.Is(err, repository.ErrUserOrganization):
		status, message = 403, "Nije odredjena vasa organizaciona jedinica."
	case errors.Is(err, repository.ErrDocumentLocked):
		status, message = 423, err.Error()
	case errors.Is(err, repository.ErrDocumentLease), errors.Is(err, repository.ErrDocumentVersion), errors.Is(err, repository.ErrDocumentClosed):
		status, message = 409, err.Error()
	case errors.Is(err, repository.ErrDocumentType), errors.Is(err, util.ErrDocumentFile):
		status, message = 400, err.Error()
	case errors.Is(err, util.ErrDocumentSize):
		status, message = 413, err.Error()
	default:
		log.Error().Err(err).Msg("document operation failed")
	}
	ctx.JSON(status, gin.H{"error": message})
}

// All document routes (including bytes) require active membership of the
// contract's organization; a supplier's unrelated contracts confer no access.
func (s *Server) documentAccess(ctx *gin.Context, contractID int, write bool) (int, bool) {
	username := ctx.MustGet(authorizationPayloadKey).(*token.Payload).Username
	user, err := s.store.GetUserByUsername(ctx.Request.Context(), username)
	if err != nil {
		documentError(ctx, err)
		return 0, false
	}
	if user == nil || user.ID < 1 || user.Status != s.config.ActiveUserStatus {
		ctx.JSON(403, gin.H{"error": "Nemate pravo pristupa dokumentima."})
		return 0, false
	}
	org, err := s.store.GetUserOrganization(ctx.Request.Context(), username, s.config.ActiveUserStatus)
	if err != nil {
		documentError(ctx, err)
		return 0, false
	}
	contract, err := s.store.GetUgoEvidProsireniByID(ctx.Request.Context(), contractID)
	if err != nil {
		documentError(ctx, err)
		return 0, false
	}
	if contract.IDUgoOrg != org {
		ctx.JSON(403, gin.H{"error": "Dokumenta pripadaju drugoj organizacionoj jedinici."})
		return 0, false
	}
	if write && contract.OtvorenUG != "X" {
		documentError(ctx, repository.ErrDocumentClosed)
		return 0, false
	}
	return user.ID, true
}
func documentID(ctx *gin.Context) (int, bool) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id < 1 {
		ctx.JSON(400, gin.H{"error": "Neispravan ID."})
		return 0, false
	}
	return id, true
}
func (s *Server) accessibleDocument(ctx *gin.Context, write bool) (*models.Document, int, bool) {
	id, ok := documentID(ctx)
	if !ok {
		return nil, 0, false
	}
	d, err := s.store.GetDocument(ctx.Request.Context(), id)
	if err != nil {
		documentError(ctx, err)
		return nil, 0, false
	}
	user, ok := s.documentAccess(ctx, d.ContractID, write)
	return d, user, ok
}
func (s *Server) ListDocuments(ctx *gin.Context) {
	id, ok := documentID(ctx)
	if !ok {
		return
	}
	if _, ok = s.documentAccess(ctx, id, false); !ok {
		return
	}
	docs, err := s.store.ListDocuments(ctx.Request.Context(), id)
	if err != nil {
		documentError(ctx, err)
		return
	}
	types, err := s.store.ListDocumentTypes(ctx.Request.Context())
	if err != nil {
		documentError(ctx, err)
		return
	}
	ctx.JSON(200, gin.H{"items": docs, "types": types, "max_bytes": s.documentLimit()})
}
func (s *Server) GetDocument(ctx *gin.Context) {
	d, _, ok := s.accessibleDocument(ctx, false)
	if ok {
		ctx.JSON(200, d)
	}
}
func (s *Server) documentLimit() int64 {
	if s.config.DocumentMaxBytes > 0 {
		return s.config.DocumentMaxBytes
	}
	return 25 * 1024 * 1024
}
func normalizeDocumentMetadata(m *models.DocumentMetadata) error {
	m.Name = strings.TrimSpace(m.Name)
	m.Number = strings.TrimSpace(m.Number)
	m.Note = strings.TrimSpace(m.Note)
	return binding.Validator.ValidateStruct(m)
}
func (s *Server) UploadDocument(ctx *gin.Context) {
	id, ok := documentID(ctx)
	if !ok {
		return
	}
	user, ok := s.documentAccess(ctx, id, true)
	if !ok {
		return
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, s.documentLimit()+1024*1024)
	defer func() {
		if ctx.Request.MultipartForm != nil {
			ctx.Request.MultipartForm.RemoveAll()
		}
	}()
	if err := ctx.Request.ParseMultipartForm(1024 * 1024); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			documentError(ctx, util.ErrDocumentSize)
		} else {
			ctx.JSON(400, gin.H{"error": "Neispravan upload zahtev."})
		}
		return
	}
	form := ctx.Request.MultipartForm
	if form == nil || len(form.File) != 1 || len(form.File["file"]) != 1 || len(form.Value["metadata"]) != 1 {
		ctx.JSON(400, gin.H{"error": "Posaljite jedan fajl i podatke dokumenta."})
		return
	}
	var meta models.DocumentMetadata
	dec := json.NewDecoder(strings.NewReader(form.Value["metadata"][0]))
	dec.DisallowUnknownFields()
	err := dec.Decode(&meta)
	if err == nil {
		err = normalizeDocumentMetadata(&meta)
	}
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Proverite naziv, tip, datum i duzinu podataka dokumenta."})
		return
	}
	header := form.File["file"][0]
	original := util.SafeOriginalName(header.Filename)
	if utf8.RuneCountInString(original) > 255 || original == "" || original == "." {
		ctx.JSON(400, gin.H{"error": "Neispravan naziv fajla."})
		return
	}
	if header.Size > s.documentLimit() {
		documentError(ctx, util.ErrDocumentSize)
		return
	}
	in, err := header.Open()
	if err != nil {
		documentError(ctx, err)
		return
	}
	defer in.Close()
	root, err := util.DocumentStorage(s.config.DocumentRoot)
	if err != nil {
		documentError(ctx, err)
		return
	}
	defer root.Close()
	temp, ext, kind, hash, size, err := util.ReceiveDocument(root, in, s.documentLimit(), original)
	if err != nil {
		documentError(ctx, err)
		return
	}
	defer root.Remove(temp)
	contract, err := s.store.GetUgoEvidProsireniByID(ctx.Request.Context(), id)
	if err != nil {
		documentError(ctx, err)
		return
	}
	folder, err := s.store.DocumentFolder(ctx.Request.Context(), id, util.DocumentFolderName(id, contract.BrUgovor, contract.PredmetUgovora))
	if err != nil {
		documentError(ctx, err)
		return
	}
	name, err := util.RandomDocumentName()
	if err != nil {
		documentError(ctx, err)
		return
	}
	doc := &models.Document{DocumentMetadata: meta, ContractID: id, UserID: user, OriginalName: original, Path: filepath.ToSlash(filepath.Join(folder, name+ext)), MIME: kind, Size: size, SHA256: hash}
	if err = s.store.ReserveDocument(ctx.Request.Context(), doc); err != nil {
		documentError(ctx, err)
		return
	}
	if err = util.PublishDocument(root, temp, doc.Path); err != nil {
		documentError(ctx, err)
		return
	}
	// On an ambiguous database response retain the final file and pending row.
	// Reconciliation validates size/hash before making pending uploads visible.
	if err = s.store.FinalizeDocument(ctx.Request.Context(), doc.ID); err != nil {
		documentError(ctx, err)
		return
	}
	ctx.JSON(201, gin.H{"id": doc.ID})
}
func (s *Server) DocumentContent(ctx *gin.Context) {
	d, _, ok := s.accessibleDocument(ctx, false)
	if !ok {
		return
	}
	root, err := util.DocumentStorage(s.config.DocumentRoot)
	if err != nil {
		documentError(ctx, err)
		return
	}
	defer root.Close()
	f, err := root.Open(d.Path)
	if err != nil {
		documentError(ctx, err)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		documentError(ctx, err)
		return
	}
	if !st.Mode().IsRegular() || st.Size() != d.Size {
		documentError(ctx, util.ErrDocumentSize)
		return
	}
	disposition := "attachment"
	if ctx.Query("preview") == "1" && (d.MIME == "application/pdf" || d.MIME == "image/png" || d.MIME == "image/jpeg") {
		disposition = "inline"
	}
	ctx.Header("Content-Type", d.MIME)
	ctx.Header("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": d.OriginalName}))
	ctx.Header("Cache-Control", "private, no-store")
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Header("Content-Security-Policy", "sandbox")
	http.ServeContent(ctx.Writer, ctx.Request, d.OriginalName, st.ModTime(), f)
}
func (s *Server) DocumentLock(ctx *gin.Context) {
	action, token := "acquire", ""
	if ctx.Request.Method != http.MethodPost {
		var req struct {
			Token string `json:"lock_token" binding:"required,len=64"`
		}
		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(400, gin.H{"error": "Token zakljucavanja je obavezan."})
			return
		}
		token = req.Token
		action = "renew"
		if ctx.Request.Method == http.MethodDelete {
			action = "release"
		}
	}
	d, user, ok := s.accessibleDocument(ctx, action != "release")
	if !ok {
		return
	}
	lease, err := s.store.DocumentLock(ctx.Request.Context(), d.ID, user, action, token)
	if err != nil {
		documentError(ctx, err)
		return
	}
	ctx.JSON(200, lease)
}
func (s *Server) ChangeDocument(ctx *gin.Context) {
	d, user, ok := s.accessibleDocument(ctx, true)
	if !ok {
		return
	}
	var req struct {
		models.DocumentMetadata
		Version int64  `json:"version,string"`
		Token   string `json:"lock_token"`
	}
	// DELETE only requires version/token, so validate metadata separately.
	if err := json.NewDecoder(io.LimitReader(ctx.Request.Body, 32*1024)).Decode(&req); err != nil {
		ctx.JSON(400, gin.H{"error": "Neispravni podaci dokumenta."})
		return
	}
	if !validLeaseInput(ctx, req.Version, req.Token) {
		return
	}
	remove := ctx.Request.Method == http.MethodDelete
	if !remove {
		if err := normalizeDocumentMetadata(&req.DocumentMetadata); err != nil {
			ctx.JSON(400, gin.H{"error": "Proverite podatke dokumenta."})
			return
		}
		d.DocumentMetadata = req.DocumentMetadata
	}
	d.Version = req.Version
	if err := s.store.ChangeDocument(ctx.Request.Context(), d, user, req.Token, remove); err != nil {
		documentError(ctx, err)
		return
	}
	ctx.JSON(200, gin.H{"status": "ok"})
}
