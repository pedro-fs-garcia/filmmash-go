package web

import (
	"bytes"
	"context"
	"errors"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/middleware"
	"filmmash/internal/vote"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	logger         *slog.Logger
	filmService    *film.Service
	registerVoteUc *vote.RegisterVoteUC
	duelService    *duel.Service
}

func NewHandler(l *slog.Logger, fs *film.Service, ds *duel.Service, rvuc *vote.RegisterVoteUC) *Handler {
	return &Handler{
		logger:         l,
		filmService:    fs,
		duelService:    ds,
		registerVoteUc: rvuc,
	}
}

var templateCache = make(map[string]*template.Template)

func init() {
	templateCache["filmPage"] = template.Must(template.ParseFS(TemplatesFS,
		"template/base.html",
		"template/film.html",
	))
	templateCache["filmModal"] = template.Must(template.ParseFS(TemplatesFS,
		"template/film_modal.html",
	))
	templateCache["duelPage"] = template.Must(template.ParseFS(TemplatesFS,
		"template/base.html",
		"template/index.html",
		"template/duelCard.html",
		"template/filmCard.html",
	))
	templateCache["duelCard"] = template.Must(template.ParseFS(TemplatesFS,
		"template/duelCard.html",
		"template/filmCard.html",
	))
	templateCache["filmListPage"] = template.Must(template.ParseFS(TemplatesFS,
		"template/base.html",
		"template/filmList.html",
		"template/film_list_items.html",
		"template/film_list_item.html",
	))
	templateCache["filmListItems"] = template.Must(template.ParseFS(TemplatesFS,
		"template/film_list_items.html",
		"template/film_list_item.html",
	))
	templateCache["filmSearchResults"] = template.Must(template.ParseFS(TemplatesFS,
		"template/film_search_results.html",
		"template/film_list_item.html",
	))
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

	f, err := h.filmService.GetFilm(ctx, id)
	if err != nil {
		if errors.Is(err, film.ErrNotFound) {
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
		err = templateCache["filmModal"].ExecuteTemplate(&buf, "filmModal", f)
	} else {
		err = templateCache["filmPage"].ExecuteTemplate(&buf, "base", f)
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
	buf.WriteTo(w)
}

func (h *Handler) DuelHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "DuelHandler"), slog.String("request_id", reqId))

	d, err := h.duelService.CreateRandomDuel(ctx)
	if err != nil {
		if errors.Is(err, duel.ErrNotEnoughFilms) {
			log.ErrorContext(ctx, "Duel could not fetch films for a duel",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Could not fetch films for a duel", http.StatusServiceUnavailable)
			return
		}
		log.ErrorContext(ctx, "failed to create duel", slog.String("error", err.Error()))
		http.Error(w, "Could not fetch the films for a duel.", http.StatusInternalServerError)
		return
	}

	buf := bytes.Buffer{}
	err = templateCache["duelPage"].ExecuteTemplate(&buf, "base", d)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template",
			slog.String("error", err.Error()),
		)
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

		_, err := h.registerVoteUc.RegisterVote(ctx, duelId, winnerId)
		if err != nil {
			log.ErrorContext(ctx, "Failed to register async vote",
				slog.String("duel_id", duelId.String()),
				slog.String("error", err.Error()),
			)
		}
	}()

	d, err := h.duelService.ComposeDuel(r.Context(), winnerId)
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
	err = templateCache["duelCard"].ExecuteTemplate(&buf, "duelCard", d)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template", slog.String("error", err.Error()))
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func (h *Handler) FilmsListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "FilmsListHandler"), slog.String("request_id", reqId))

	lastSeendId, err := strconv.Atoi(r.FormValue("last_seen_id"))
	LastSeenRating, err := strconv.ParseFloat(r.FormValue("last_seen_rating"), 64)
	size, err := strconv.Atoi(r.FormValue("size"))
	if lastSeendId == 0 && LastSeenRating == 0.0 && size == 0 {
		LastSeenRating = 9999.0
		size = 50
	}

	pars := film.PaginationParameters{
		LastSeenId:     lastSeendId,
		LastSeenRating: LastSeenRating,
		Size:           size,
	}

	resp, err := h.filmService.GetFilmsPaginatedByRating(ctx, pars)
	if err != nil {
		log.ErrorContext(ctx, "Failed to get films", slog.String("error", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	buf := bytes.Buffer{}
	if r.Header.Get("HX-Request") == "true" {
		err = templateCache["filmListItems"].ExecuteTemplate(&buf, "filmListItems", resp)
	} else {
		err = templateCache["filmListPage"].ExecuteTemplate(&buf, "base", resp)
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

func (h *Handler) SearchFilmsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "SearchFilmsHandler"), slog.String("request_id", reqId))

	search := strings.TrimSpace(r.FormValue("film_name"))
	if len(search) < 3 {
		http.Error(w, "Search parameters should have at least three letters", http.StatusBadRequest)
		return
	}

	resp, err := h.filmService.SearchFilmByName(ctx, search)
	if err != nil {
		log.ErrorContext(ctx, "Failed to search films by name", slog.String("error", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	buf := bytes.Buffer{}
	err = templateCache["filmSearchResults"].ExecuteTemplate(&buf, "filmSearchResults", resp)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template", slog.String("error", err.Error()))
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
