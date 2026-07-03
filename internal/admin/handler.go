package admin

import (
	"bytes"
	"filmmash/internal/middleware"
	"filmmash/internal/view"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

type Handler struct {
	logger  *slog.Logger
	service *Service
}

func NewHandler(logger *slog.Logger, service *Service) *Handler {
	return &Handler{
		logger:  logger.With("component", "Handler"),
		service: service,
	}
}

func IsValidUUID7(u string) bool {
	parsed, err := uuid.Parse(u)
	if err != nil {
		return false
	}
	if parsed.Version() != 7 || parsed.Variant() != uuid.RFC4122 {
		return false
	}
	return true
}

func (h *Handler) ListUsersPaginatedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "ListUsersPaginatedHandler"), slog.String("request_id", reqId))

	size, _ := strconv.Atoi(r.FormValue("size"))
	if size <= 0 {
		size = 50
	}

	lastSeenId := r.FormValue("last_seen_id")
	if lastSeenId == "" {
		lastSeenId = uuid.Nil.String()
	} else if !IsValidUUID7(lastSeenId) {
		log.WarnContext(ctx, "invalid uuid for last_seen_id", slog.String("last_seen_id", lastSeenId))
		http.Error(w, "Invalid uuid for last_seen_id", http.StatusBadRequest)
		return
	}

	pars := PaginationParameters{
		Size:       size,
		LastSeenId: lastSeenId,
	}
	resp, err := h.service.GetUsersPaginated(ctx, pars)
	if err != nil {
		log.ErrorContext(ctx, "error to get user data", slog.String("error", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	buf := bytes.Buffer{}
	if r.Header.Get("HX-Request") == "true" {
		err = view.TemplateCache["adminUsers"].ExecuteTemplate(&buf, "adminUsers", resp)
	} else {
		err = view.TemplateCache["adminDashboardPage"].ExecuteTemplate(&buf, "base", resp)
	}
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template", slog.String("error", err.Error()))
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
