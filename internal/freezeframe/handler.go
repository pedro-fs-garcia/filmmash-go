// Package freezeframe serves the frame-guessing game: a still lifted from a
// film, and a handful of titles to pick between.
package freezeframe

import (
	"bytes"
	"encoding/json"
	"errors"
	"filmmash/internal/database"
	"filmmash/internal/middleware"
	"filmmash/internal/view"
	"log/slog"
	"net/http"
)

type Handler struct {
	logger  *slog.Logger
	service *Service
}

func NewHandler(logger *slog.Logger, service *Service) *Handler {
	return &Handler{logger: logger, service: service}
}

func (h *Handler) FreezeFrameHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "FreezeFrameHandler"),
		slog.String("request_id", reqId),
	)

	buf := bytes.Buffer{}
	err := view.TemplateCache["freezeFramePage"].ExecuteTemplate(&buf, "base", nil)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template",
			slog.String("error", err.Error()),
		)
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}

func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "CreateGame"),
		slog.String("request_id", reqId),
	)

	var game Game
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		log.ErrorContext(ctx, "could not parse request body into a game", slog.String("error", err.Error()))
		http.Error(w, "request body must be a valid game", http.StatusBadRequest)
		return
	}

	if err := game.Validate(); err != nil {
		log.ErrorContext(ctx, "submitted invalid game", slog.String("error", err.Error()))
		http.Error(w, "request body must be a valid game: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.SeedGame(ctx, &game); err != nil {
		if errors.Is(err, database.ErrDuplicateEntry) {
			http.Error(w, "game already exists", http.StatusConflict)
		} else if errors.Is(err, database.ErrInvalidInput) {
			http.Error(w, "invalid input value", http.StatusBadRequest)
		} else if errors.Is(err, database.ErrMissingData) {
			http.Error(w, "missing data", http.StatusBadRequest)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		log.ErrorContext(ctx, "failed to seed game", slog.String("error", err.Error()))
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(game)
}
