package predictor

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FG-CollectLabs/grading-predictor-backend/internal/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GCSClient is intentionally an interface{} placeholder — wire in cloud.google.com/go/storage
// when GCS is provisioned. For v0.1 the image path is stored in DB without an actual upload.
type Handler struct {
	DB        *pgxpool.Pool
	GCSClient interface{} // *storage.Client when wired
	GCSBucket string
}

func (h *Handler) Routes(mux *http.ServeMux, auth httpx.Middleware) {
	mux.HandleFunc("GET /v1/cards", h.listCards)
	mux.HandleFunc("GET /v1/cards/{id}", h.getCard)
	mux.HandleFunc("GET /v1/cards/{id}/stats", h.getCardStats)
	mux.HandleFunc("GET /v1/cards/{id}/certs", h.listCertsForCard)

	mux.Handle("POST /v1/cards", auth(http.HandlerFunc(h.createCard)))
	mux.Handle("DELETE /v1/cards/{id}", auth(http.HandlerFunc(h.deleteCard)))
	mux.Handle("POST /v1/certs", auth(http.HandlerFunc(h.createCert)))
	mux.HandleFunc("GET /v1/certs/{id}", h.getCertDetail)
	mux.Handle("PATCH /v1/certs/{id}/grade", auth(http.HandlerFunc(h.setCertGrade)))
	mux.Handle("POST /v1/certs/{id}/images", auth(http.HandlerFunc(h.uploadCertImage)))
	mux.Handle("POST /v1/certs/{id}/inspections", auth(http.HandlerFunc(h.createInspection)))
	mux.HandleFunc("GET /v1/certs/{id}/inspections", h.listInspections)
}

// ── Cards ────────────────────────────────────────────────────────────────────

func (h *Handler) listCards(w http.ResponseWriter, r *http.Request) {
	rows, err := listCards(r.Context(), h.DB)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
}

func (h *Handler) getCard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_id", "invalid card id")
		return
	}
	card, err := getCard(r.Context(), h.DB, id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "card not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, card)
}

func (h *Handler) getCardStats(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_id", "invalid card id")
		return
	}
	rows, err := getCardStats(r.Context(), h.DB, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
}

func (h *Handler) listCertsForCard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_id", "invalid card id")
		return
	}
	rows, err := listCertsForCard(r.Context(), h.DB, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
}

type createCardRequest struct {
	Game             string  `json:"game"`
	SetCode          string  `json:"set_code"`
	SetName          string  `json:"set_name"`
	CardName         string  `json:"card_name"`
	CardNumber       string  `json:"card_number"`
	ImageURL         *string `json:"image_url"`
	MarketDisplayKey *string `json:"market_display_key"`
}

func (h *Handler) createCard(w http.ResponseWriter, r *http.Request) {
	var req createCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Game == "" || req.SetCode == "" || req.CardName == "" || req.CardNumber == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing_fields", "game, set_code, card_name, card_number required")
		return
	}
	card, err := createCard(r.Context(), h.DB, req.Game, req.SetCode, req.SetName, req.CardName, req.CardNumber,
		req.ImageURL, req.MarketDisplayKey)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, card)
}

func (h *Handler) deleteCard(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_id", "invalid card id")
		return
	}
	found, err := deleteCard(r.Context(), h.DB, id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	if !found {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "card not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Certs ────────────────────────────────────────────────────────────────────

func (h *Handler) getCertDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_id", "invalid cert id")
		return
	}
	cert, err := getCertDetail(r.Context(), h.DB, id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "cert not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, cert)
}

type createCertRequest struct {
	CardID     int64  `json:"card_id"`
	CertNumber string `json:"cert_number"`
	Grader     string `json:"grader"`
	Notes      string `json:"notes"`
	Category   string `json:"category"` // raw | psa9 | psa10 | cgc9 | cgc10
	Purpose    string `json:"purpose"`  // analytics | buy_and_grade | crack_and_regrade
}

func (h *Handler) createCert(w http.ResponseWriter, r *http.Request) {
	var req createCertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.CertNumber == "" || req.CardID == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "missing_fields", "card_id and cert_number required")
		return
	}
	if req.Grader == "" {
		req.Grader = "PSA"
	}
	cert, err := createCert(r.Context(), h.DB, req.CardID, req.CertNumber, req.Grader, req.Notes, req.Category, req.Purpose)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, cert)
}

type setCertGradeRequest struct {
	Grade    int16  `json:"grade"`
	GradedAt string `json:"graded_at"` // YYYY-MM-DD, optional
}

func (h *Handler) setCertGrade(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_id", "invalid cert id")
		return
	}
	var req setCertGradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	var gradedAt pgtype.Date
	if req.GradedAt != "" {
		t, err := time.Parse("2006-01-02", req.GradedAt)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "bad_date", "graded_at must be YYYY-MM-DD")
			return
		}
		gradedAt = pgtype.Date{Time: t, Valid: true}
	}
	cert, err := setCertGrade(r.Context(), h.DB, id, req.Grade, gradedAt)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "cert not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, cert)
}

// ── Images ───────────────────────────────────────────────────────────────────

func (h *Handler) uploadCertImage(w http.ResponseWriter, r *http.Request) {
	certID, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_id", "invalid cert id")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_form", err.Error())
		return
	}

	side := r.FormValue("side")
	if side != "front" && side != "back" {
		httpx.WriteError(w, http.StatusBadRequest, "bad_side", "side must be 'front' or 'back'")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "no_file", "image file required")
		return
	}
	defer file.Close()

	// Look up cert_number for the GCS path
	cert, err := getCert(r.Context(), h.DB, certID)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "cert not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	gcsPath := fmt.Sprintf("%s/%s%s", cert.CertNumber, side, ext)

	// GCS upload: wire h.GCSClient (*storage.Client) when bucket is provisioned.
	slog.Info("image received, skipping GCS upload (not configured)", "path", gcsPath)

	img, err := upsertCertImage(r.Context(), h.DB, certID, side, gcsPath)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, img)
}


// ── Inspections ──────────────────────────────────────────────────────────────

type createInspectionRequest struct {
	CenteringFrontLR *int16  `json:"centering_front_lr"`
	CenteringFrontTB *int16  `json:"centering_front_tb"`
	CenteringBackLR  *int16  `json:"centering_back_lr"`
	CenteringBackTB  *int16  `json:"centering_back_tb"`
	SurfaceFront     *string `json:"surface_front"`
	SurfaceBack      *string `json:"surface_back"`
	CornerTL         *string `json:"corner_tl"`
	CornerTR         *string `json:"corner_tr"`
	CornerBL         *string `json:"corner_bl"`
	CornerBR         *string `json:"corner_br"`
	EdgeTop          *string `json:"edge_top"`
	EdgeBottom       *string `json:"edge_bottom"`
	EdgeLeft         *string `json:"edge_left"`
	EdgeRight        *string `json:"edge_right"`
	Notes            *string `json:"notes"`
	Source           string  `json:"source"` // "manual" | "auto"
}

func (h *Handler) createInspection(w http.ResponseWriter, r *http.Request) {
	certID, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_id", "invalid cert id")
		return
	}
	var req createInspectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Source == "" {
		req.Source = "manual"
	}
	if req.Source != "manual" && req.Source != "auto" {
		httpx.WriteError(w, http.StatusBadRequest, "bad_source", "source must be 'manual' or 'auto'")
		return
	}

	insp, err := createInspection(r.Context(), h.DB, certID, req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, insp)
}

func (h *Handler) listInspections(w http.ResponseWriter, r *http.Request) {
	certID, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "bad_id", "invalid cert id")
		return
	}
	rows, err := listInspectionsForCert(r.Context(), h.DB, certID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func parseID(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

func isNotFound(err error) bool {
	return err == pgx.ErrNoRows
}
