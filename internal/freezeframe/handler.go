// Package freezeframe serves the frame-guessing game: a still lifted from a
// film, and a handful of titles to pick between.
package freezeframe

import (
	"encoding/json"
	"errors"
	"filmmash/internal/database"
	"filmmash/internal/middleware"
	"filmmash/internal/tmdb"
	"log/slog"
	"math/rand/v2"
	"net/http"
)

type Handler struct {
	logger     *slog.Logger
	service    *Service
	tmdbClient *tmdb.Client
	baseURL    string
}

func (h *Handler) ConfigureDaily(client *tmdb.Client, baseURL string) {
	h.tmdbClient = client
	h.baseURL = baseURL
}

func NewHandler(logger *slog.Logger, service *Service) *Handler {
	return &Handler{logger: logger, service: service}
}

// Compatibility entry points use the same server-backed daily flow.
func (h *Handler) GetRound(w http.ResponseWriter, r *http.Request)     { h.DailyGame(w, r) }
func (h *Handler) SubmitAnswer(w http.ResponseWriter, r *http.Request) { h.DailyAnswer(w, r) }

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

	// API scheduling follows the same curation gate as the admin UI.
	for _, reel := range game.Reels {
		for _, rf := range reel.ReelFrames {
			var approved bool
			err := h.service.repo.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM frames WHERE film_id=$1 AND image_path=$2 AND approved AND curated_order=$3)`, reel.Film.Id, rf.Frame.ImagePath, rf.Seq).Scan(&approved)
			if err != nil {
				http.Error(w, "could not check curated frames", 500)
				return
			}
			if !approved {
				http.Error(w, "each reel requires four approved frames in curated order", 400)
				return
			}
		}
	}
	for i := range game.Reels {
		options := game.Reels[i].Alternatives
		rand.Shuffle(len(options), func(a, b int) { options[a], options[b] = options[b], options[a] })
		for j := range options {
			options[j].Seq = int16(j + 1)
		}
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
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(game); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
