package admin

import (
	"bytes"
	"errors"
	"filmmash/internal/film"
	"filmmash/internal/middleware"
	"filmmash/internal/tmdb"
	"filmmash/internal/view"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	logger     *slog.Logger
	service    *Service
	tmdbClient *tmdb.Client
}

func NewHandler(logger *slog.Logger, service *Service, tmdbClient *tmdb.Client) *Handler {
	return &Handler{
		logger:     logger.With("component", "Handler"),
		service:    service,
		tmdbClient: tmdbClient,
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
	_, _ = buf.WriteTo(w)
}

func (h *Handler) SearchFilmOnTmdbHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "SearchFilmOnTmdbHandler"), slog.String("request_id", reqId))

	query := strings.TrimSpace(r.URL.Query().Get("search"))
	if len(query) < 3 {
		http.Error(w, "search parameter should have at least three letters", http.StatusBadRequest)
		return
	}

	page := r.URL.Query().Get("page")
	res64, err := strconv.ParseInt(page, 10, 32)
	if err != nil {
		http.Error(w, "page parameter should be a valid integer", http.StatusBadRequest)
		return
	}

	pars := tmdb.SearchFilmParams{Query: query, Page: int32(res64)}
	res, err := h.tmdbClient.SearchFilm(&pars)
	if err != nil {
		log.ErrorContext(ctx, "failed to search films in TMDB", slog.String("error", err.Error()))
		http.Error(w, "Failed to search films on TMDB", http.StatusBadGateway)
		return
	}

	data := TmdbSearchView{
		Movies:   res.Results,
		Search:   query,
		NextPage: res.Page + 1,
		HasMore:  res.Page < res.TotalPages,
	}

	if err := h.service.MarkInCatalogue(ctx, data.Movies); err != nil {
		log.WarnContext(ctx, "failed to mark films already in catalogue", slog.String("error", err.Error()))
	}

	buf := bytes.Buffer{}
	err = view.TemplateCache["adminTmdbResults"].ExecuteTemplate(&buf, "adminTmdbResults", data)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template", slog.String("error", err.Error()))
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}

func (h *Handler) AddTmdbFilmHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqId := middleware.GetRequestID(ctx)
	log := h.logger.With(slog.String("method", "AddTmdbFilmHandler"), slog.String("request_id", reqId))

	par := chi.URLParam(r, "id")
	filmId, err := strconv.ParseInt(par, 10, 32)
	if err != nil {
		log.WarnContext(ctx, "failed to search films in TMDB", slog.String("error", err.Error()))
		http.Error(w, "Failed to search films on TMDB", http.StatusBadGateway)
		return
	}

	movie, err := h.tmdbClient.GetFilm(int32(filmId))
	if err != nil {
		log.ErrorContext(ctx, "failed to get film details in TMDB", slog.String("error", err.Error()))
		http.Error(w, "Failed to get film on TMDB", http.StatusBadGateway)
		return
	}

	if movie.Director.Name == "" {
		log.WarnContext(ctx, "film has no resolvable director on TMDB", slog.Int("tmdb_id", movie.Id))
		http.Error(w, "Film has no director on TMDB and cannot be added", http.StatusUnprocessableEntity)
		return
	}

	year, _, _ := strings.Cut(movie.ReleaseDate, "-")
	release_year, _ := strconv.Atoi(year)

	f := film.Film{
		Id:          movie.Id,
		Title:       movie.Title,
		Year:        release_year,
		ImagePath:   movie.PosterPath,
		Popularity:  float64(movie.Popularity),
		VoteAverage: float64(movie.VoteAverage),
		Director: film.Director{
			Id:   movie.Director.Id,
			Name: movie.Director.Name,
		},
	}

	result := TmdbAddResultView{Movie: movie, Status: "Added to catalog ✓", Added: true}
	err = h.service.InsertFilm(ctx, &f)
	if errors.Is(err, film.ErrDuplicateEntry) {
		log.InfoContext(ctx, "film already in catalog", slog.Int("tmdb_id", movie.Id))
		result = TmdbAddResultView{Movie: movie, Status: "Already in the catalog"}
	} else if err != nil {
		log.ErrorContext(ctx, "failed to insert film in local database", slog.String("error", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	buf := bytes.Buffer{}
	err = view.TemplateCache["adminTmdbAddResult"].ExecuteTemplate(&buf, "adminTmdbAddResult", result)
	if err != nil {
		log.ErrorContext(ctx, "Failed to render HTML template", slog.String("error", err.Error()))
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}
