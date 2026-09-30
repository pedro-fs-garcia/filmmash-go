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

	"github.com/google/uuid"
)

type Handler struct {
	logger  *slog.Logger
	service *Service
}

func NewHandler(logger *slog.Logger, service *Service) *Handler {
	return &Handler{logger: logger, service: service}
}

func (h *Handler) GetRound(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "GetRound"),
		slog.String("request_id", reqID),
	)

	cookie, err := r.Cookie("freezeframe_state")
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		log.ErrorContext(ctx, "failed to read cookie from request", slog.String("error", err.Error()))
	}

	state := StateCookie{}
	if cookie != nil {
		err = json.Unmarshal([]byte(cookie.Value), &state)
		if err != nil {
			log.ErrorContext(ctx, "failed to unmarshal cookie to struct", slog.String("error", err.Error()))
		}
	}

	var round Round
	if state.StateID == uuid.Nil || state.LastSeenAt.Before(h.service.gameCache.ValidAt()) {
		round, err = h.service.GetRound(ctx, 1)
	} else {
		round, err = h.service.GetRound(ctx, state.ReelSeq)
	}

	if err != nil {
		log.ErrorContext(ctx, "failed to get expected round from today's game", slog.String("error", err.Error()))
		if errors.Is(err, ErrNoGameForDate) {
			http.Error(w, "no games available today", http.StatusNotFound)
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	stateID, err := uuid.NewV7()
	if err != nil {
		log.ErrorContext(ctx, "Failed to generate UUIDV7 for state cookie",
			slog.String("error", err.Error()),
		)
	}
	setStateCookie(w, stateID, round.GameID, round.ReelID)

	buf := bytes.Buffer{}
	err = view.TemplateCache["freezeFramePage"].ExecuteTemplate(&buf, "base", round)
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
	reqID := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "CreateGame"),
		slog.String("request_id", reqID),
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
	if err := json.NewEncoder(w).Encode(game); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) GetTodaysGame(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "GetTodaysGame"),
		slog.String("request_id", reqID),
	)

	game, err := h.service.GetTodaysGame(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to get expected round from today's game", slog.String("error", err.Error()))
		if errors.Is(err, ErrNoGameForDate) {
			http.Error(w, "no games available today", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(game); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
