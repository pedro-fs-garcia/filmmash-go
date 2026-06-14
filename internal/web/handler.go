package web

import (
	"context"
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/vote"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	filmService    *film.Service
	registerVoteUc *vote.RegisterVoteUC
	duelService    *duel.Service
}

func NewHandler(fs *film.Service, ds *duel.Service, rvuc *vote.RegisterVoteUC) *Handler {
	return &Handler{
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
}

func (h *Handler) FilmHandler(w http.ResponseWriter, r *http.Request) {
	par := chi.URLParam(r, "id")
	id, err := strconv.Atoi(par)
	if err != nil {
		http.Error(w, "Invalid Id", http.StatusBadRequest)
		return
	}

	film, err := h.filmService.GetFilm(r.Context(), id)
	if err != nil {
		http.Error(w, "Error on GetFilm service", http.StatusInternalServerError)
		return
	}

	err = templateCache["filmPage"].ExecuteTemplate(w, "base", film)
	if err != nil {
		http.Error(w, "Error mounting html file", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) DuelHandler(w http.ResponseWriter, r *http.Request) {
	duel, err := h.duelService.CreateRandomDuel(r.Context())
	if err != nil {
		log.Println(err)
		http.Error(w, "Could not fetch the films for a duel.", http.StatusExpectationFailed)
		return
	}

	err = templateCache["duelPage"].ExecuteTemplate(w, "base", duel)
	if err != nil {
		log.Println(err)
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) DuelResultHandler(w http.ResponseWriter, r *http.Request) {
	duelId, err := uuid.Parse(chi.URLParam(r, "duel_id"))
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid duel could not be parsed", http.StatusBadRequest)
		return
	}
	winnerId, err := strconv.Atoi(r.FormValue("winner"))
	if err != nil {
		log.Println(err)
		http.Error(w, "Invalid winner could not be parsed", http.StatusBadRequest)
		return
	}

	fmt.Println(duelId, winnerId)

	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
		defer cancel()

		_, err := h.registerVoteUc.RegisterVote(ctx, duelId, winnerId)
		if err != nil {
			log.Printf("Failed to register async vote (duelId: %v): %v", duelId, err)
		}
	}()

	duel, err := h.duelService.ComposeDuel(r.Context(), winnerId)
	if err != nil {
		log.Println(err)
		http.Error(w, "Could not fetch the films for a duel.", http.StatusExpectationFailed)
		return
	}

	err = templateCache["duelCard"].ExecuteTemplate(w, "duelCard", duel)
	if err != nil {
		log.Println(err)
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}
}
