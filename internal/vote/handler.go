package vote

import (
	"bytes"
	"context"
	"errors"
	"filmmash/internal/auth"
	"filmmash/internal/duel"
	"filmmash/internal/middleware"
	"filmmash/internal/view"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	logger         *slog.Logger
	registerVoteUC *RegisterVoteUC
	listVoteUC     *ListVotesUC
}

func NewHandler(logger *slog.Logger, rvuc *RegisterVoteUC, lvuc *ListVotesUC) *Handler {
	return &Handler{
		logger:         logger,
		registerVoteUC: rvuc,
		listVoteUC:     lvuc,
	}
}

func (h *Handler) MyVotesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "MyVotesHandler"), slog.String("request_id", reqId))

	session, ok := auth.SessionFromContext(ctx)
	if !ok {
		http.Redirect(w, r, "/auth/login?next=/ui/my_votes", http.StatusSeeOther)
		return
	}

	matchups, err := h.listVoteUC.ListMyVotes(ctx, session.UserID)
	if err != nil {
		log.ErrorContext(ctx, "listing user's votes", slog.String("error", err.Error()))
		http.Error(w, "Failed to fetch users votes", http.StatusInternalServerError)
		return
	}

	buf := bytes.Buffer{}
	err = view.TemplateCache["myVotesPage"].ExecuteTemplate(&buf, "base", matchups)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template", slog.String("error", err.Error()))
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

type filmVotesView struct {
	FilmId   int
	Matchups []MatchupResult
}

func (h *Handler) ListFilmVotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "ListFilmVotes"), slog.String("request_id", reqId))

	par := chi.URLParam(r, "film_id")
	filmId, err := strconv.Atoi(par)
	if err != nil {
		log.WarnContext(ctx, "Invalid value for film_id",
			slog.String("error", err.Error()),
			slog.String("raw_id", par),
		)
		http.Error(w, "Invalid film_id", http.StatusBadRequest)
		return
	}

	matchups, err := h.listVoteUC.ListVotes(ctx, filmId)
	if err != nil {
		log.ErrorContext(ctx, "Failed to list film votes", slog.String("error", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	votesView := filmVotesView{FilmId: filmId, Matchups: matchups}
	buf := bytes.Buffer{}
	err = view.TemplateCache["filmVotes"].ExecuteTemplate(&buf, "filmVotes", votesView)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template", slog.String("error", err.Error()))
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func (h *Handler) DuelResultHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "DuelResultHandler"), slog.String("request_id", reqId))

	duelId, err := uuid.Parse(chi.URLParam(r, "duel_id"))
	if err != nil {
		log.WarnContext(ctx, "failed to parse duel id from URL",
			slog.String("error", err.Error()),
			slog.String("raw_id", chi.URLParam(r, "duel_id")),
		)
		http.Error(w, "Invalid duel could not be parsed", http.StatusBadRequest)
		return
	}

	winnerId, err := strconv.Atoi(r.FormValue("winner"))
	if err != nil {
		log.WarnContext(ctx, "failed to parse duel winner id from URL",
			slog.String("error", err.Error()),
			slog.String("raw_id", r.FormValue("winner")),
		)
		http.Error(w, "Invalid winner could not be parsed", http.StatusBadRequest)
		return
	}

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.ErrorContext(r.Context(), "recovered from panic in async vote",
					slog.String("duel_id", duelId.String()),
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
				)
			}
		}()

		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
		defer cancel()

		_, err := h.registerVoteUC.RegisterVote(ctx, duelId, winnerId)
		if err != nil {
			log.ErrorContext(ctx, "Failed to register async vote",
				slog.String("duel_id", duelId.String()),
				slog.String("error", err.Error()),
			)
		}
	}()

	d, err := h.registerVoteUC.NewDuelFromWinner(r.Context(), winnerId)
	if err != nil {
		if errors.Is(err, duel.ErrNotEnoughFilms) {
			log.ErrorContext(ctx, "Not enough films to compose a duel",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Could not fetch the films for a duel.", http.StatusServiceUnavailable)
			return
		}
		log.ErrorContext(ctx, "Duel could not be composed", slog.String("error", err.Error()))
		http.Error(w, "Duel could not be composed", http.StatusInternalServerError)
		return
	}

	buf := bytes.Buffer{}
	err = view.TemplateCache["duelCard"].ExecuteTemplate(&buf, "duelCard", d)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template", slog.String("error", err.Error()))
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
