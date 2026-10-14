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

	state, err := GetCookie[StateCookie](ctx, r)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		log.ErrorContext(ctx, "failed to read cookie from request", slog.String("error", err.Error()))
	}

	game, err := h.service.GetTodaysGame(ctx)
	if err != nil {
		if errors.Is(err, ErrNoGameForDate) {
			http.Error(w, "no game available", http.StatusNotFound)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	if state.GameID == game.ID && state.Finished {
		score, _ := GetCookie[ScoreCookie](ctx, r)
		if score.GameID != game.ID {
			score.CurrentScore = 0
		}
		h.renderResults(w, r, score.CurrentScore, len(game.Reels))
		return
	}
	var round Round
	if state.StateID == uuid.Nil || state.GameID != game.ID {
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
	setStateCookie(w, stateID, round.GameID, round.ReelID, round.ReelSeq, false)

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

func (h *Handler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetRequestID(ctx)
	log := h.logger.With(
		slog.String("method", "SubmitAnswer"),
		slog.String("request_id", reqID),
	)

	var submission Submission
	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		log.ErrorContext(ctx, "could not parse answer submission body into submission", slog.String("error", err.Error()))
		http.Error(w, "must be a valid submission format", http.StatusBadRequest)
		return
	}
	if err := submission.Validate(); err != nil {
		log.DebugContext(ctx, "could not parse answer submission body into submission", slog.String("error", err.Error()))
		http.Error(w, "must be a valid submission format", http.StatusBadRequest)
		return
	}

	stateCookie, err := GetCookie[StateCookie](ctx, r)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		log.ErrorContext(ctx, "failed to read state cookie from request", slog.String("error", err.Error()))
		h.GetRound(w, r)
		return
	}

	scoreCookie, err := GetCookie[ScoreCookie](ctx, r)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		log.ErrorContext(ctx, "failed to read state cookie from request", slog.String("error", err.Error()))
	}

	g, err := h.service.GetTodaysGame(ctx)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if scoreCookie.GameID != g.ID {
		scoreCookie = ScoreCookie{GameID: g.ID}
	}

	if stateCookie.GameID != g.ID || stateCookie.StateID == uuid.Nil || stateCookie.Finished {
		h.GetRound(w, r)
		return
	}
	if submission.ReelSeq != stateCookie.ReelSeq || submission.ReelID != stateCookie.ReelID {
		http.Error(w, "answer does not match the current reel", http.StatusConflict)
		return
	}
	score, err := h.service.ProcessSubmission(submission, stateCookie, scoreCookie.CurrentScore)
	if err != nil {
		http.Error(w, "invalid answer submission", http.StatusBadRequest)
		return
	}
	SetScoreCookie(w, g.ID, score)
	if int(stateCookie.ReelSeq) == len(g.Reels) {
		setStateCookie(w, stateCookie.StateID, g.ID, stateCookie.ReelID, stateCookie.ReelSeq, true)
		h.renderResults(w, r, score, len(g.Reels))
		return
	}
	setStateCookie(w, stateCookie.StateID, g.ID, g.Reels[stateCookie.ReelSeq].ID, stateCookie.ReelSeq+1, false)

	round, err := h.service.GetRound(ctx, stateCookie.ReelSeq+1)
	if err != nil {
		log.ErrorContext(ctx, "failed to get expected round from today's game", slog.String("error", err.Error()))
		if errors.Is(err, ErrNoGameForDate) {
			http.Error(w, "no games available today", http.StatusNotFound)
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
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

func (h *Handler) renderResults(w http.ResponseWriter, r *http.Request, score int16, totalReels int) {
	data := struct {
		Score      int16
		TotalReels int
		MaxScore   int
	}{score, totalReels, totalReels * 200}
	var buf bytes.Buffer
	if err := view.TemplateCache["freezeFrameResultsPage"].ExecuteTemplate(&buf, "base", data); err != nil {
		h.logger.ErrorContext(r.Context(), "failed to render results", slog.String("error", err.Error()))
		http.Error(w, "failed to render results", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
