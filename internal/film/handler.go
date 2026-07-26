package film

import (
	"bytes"
	"errors"
	"filmmash/internal/middleware"
	"filmmash/internal/view"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	logger  *slog.Logger
	service *Service
}

func NewHandler(logger *slog.Logger, service *Service) *Handler {
	return &Handler{
		logger:  logger,
		service: service,
	}
}

func (h *Handler) FilmHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "FilmHandler"),
		slog.String("request_id", reqId),
	)

	par := chi.URLParam(r, "id")
	id, err := strconv.Atoi(par)
	if err != nil {
		log.WarnContext(ctx, "failed to parse film id param from URL",
			slog.String("error", err.Error()),
			slog.String("raw_id", par),
		)

		http.Error(w, "Invalid Id", http.StatusBadRequest)
		return
	}

	f, err := h.service.GetFilm(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, fmt.Sprintf("Film (id: %v) was not found", id), http.StatusNotFound)
			return
		}

		log.ErrorContext(ctx, "Internal error to get film by id",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	buf := bytes.Buffer{}
	if r.Header.Get("HX-Request") == "true" {
		err = view.TemplateCache["filmModal"].ExecuteTemplate(&buf, "filmModal", f)
	} else {
		err = view.TemplateCache["filmPage"].ExecuteTemplate(&buf, "base", f)
	}
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Error mounting html file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}

func (h *Handler) FilmsListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "FilmsListHandler"), slog.String("request_id", reqId))

	lastSeendId, err := strconv.Atoi(r.FormValue("last_seen_id"))
	if err != nil {
		http.Error(w, "invalid value for last_seen_id", http.StatusBadRequest)
	}
	LastSeenRating, err := strconv.ParseFloat(r.FormValue("last_seen_rating"), 64)
	if err != nil {
		http.Error(w, "invalid value for last_seen_rating", http.StatusBadRequest)
	}
	size, err := strconv.Atoi(r.FormValue("size"))
	if err != nil {
		http.Error(w, "invalid value for list size", http.StatusBadRequest)
		return
	}
	if lastSeendId == 0 && LastSeenRating == 0.0 && size == 0 {
		LastSeenRating = 9999.0
		size = 50
	}

	pars := PaginationParameters{
		LastSeenId:     lastSeendId,
		LastSeenRating: LastSeenRating,
		Size:           size,
	}

	resp, err := h.service.GetFilmsPaginatedByRating(ctx, pars)
	if err != nil {
		log.ErrorContext(ctx, "Failed to get films", slog.String("error", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	buf := bytes.Buffer{}
	if r.Header.Get("HX-Request") == "true" {
		err = view.TemplateCache["filmListItems"].ExecuteTemplate(&buf, "filmListItems", resp)
	} else {
		err = view.TemplateCache["filmListPage"].ExecuteTemplate(&buf, "base", resp)
	}
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template", slog.String("error", err.Error()))
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}

func (h *Handler) SearchFilmsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "SearchFilmsHandler"), slog.String("request_id", reqId))

	search := strings.TrimSpace(r.FormValue("film_name"))
	if len(search) < 3 {
		http.Error(w, "Search parameters should have at least three letters", http.StatusBadRequest)
		return
	}

	resp, err := h.service.SearchFilmByName(ctx, search)
	if err != nil {
		log.ErrorContext(ctx, "Failed to search films by name", slog.String("error", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	buf := bytes.Buffer{}
	err = view.TemplateCache["filmSearchResults"].ExecuteTemplate(&buf, "filmSearchResults", resp)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template", slog.String("error", err.Error()))
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}
