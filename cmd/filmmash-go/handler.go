package main

import (
	"filmmash/internal/duel"
	"filmmash/internal/film"
	"filmmash/internal/vote"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type HomeData struct {
	Title   string
	Message string
}

type AppHandler struct {
	voteService *vote.Service
	filmService *film.Service
	duelService *duel.Service
}

func NewAppHandler(fs *film.Service, ds *duel.Service, vs *vote.Service) *AppHandler {
	return &AppHandler{
		filmService: fs,
		duelService: ds,
		voteService: vs,
	}
}

func (h *AppHandler) FilmHandler(w http.ResponseWriter, r *http.Request) {
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
	// json.NewEncoder(w).Encode(film)
	t := template.Must(template.ParseFiles("templates/base.html", "templates/film.html"))
	err = t.ExecuteTemplate(w, "base", film)
	if err != nil {
		http.Error(w, "Error mounting html file", http.StatusInternalServerError)
		return
	}
}

func (h *AppHandler) DuelHandler(w http.ResponseWriter, r *http.Request) {
	duel, err := h.duelService.CreateDuel(r.Context())
	if err != nil {
		log.Println(err)
		http.Error(w, "Could not fetch the films for a duel.", http.StatusExpectationFailed)
		return
	}

	t := template.Must(template.ParseFiles("templates/base.html", "templates/index.html", "templates/duelCard.html", "templates/filmCard.html"))
	err = t.ExecuteTemplate(w, "base", duel)
	if err != nil {
		log.Println(err)
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}
}

func (h *AppHandler) DuelResultHandler(w http.ResponseWriter, r *http.Request) {

	duelId, err := uuid.Parse(chi.URLParam(r, "duel_id"))
	if err != nil {
		log.Println(err)
	}
	winnerId, err := strconv.Atoi(r.FormValue("winner"))
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(duelId, winnerId)
		var wg sync.WaitGroup
		wg.Go(func() {
			h.voteService.RegisterVote(r.Context(), duelId, winnerId)
		})
		wg.Wait()
	}

	duel, err := h.duelService.ComposeDuel(r.Context(), winnerId)
	// duel, err := h.duelService.CreateDuel(r.Context())
	if err != nil {
		log.Println()
		http.Error(w, "Could not fetch the films for a duel.", http.StatusExpectationFailed)
		return
	}
	t := template.Must(template.ParseFiles("templates/duelCard.html", "templates/filmCard.html"))
	err = t.ExecuteTemplate(w, "duelCard", duel)
	if err != nil {
		log.Println(err)
		http.Error(w, "Error mounting HTML file", http.StatusInternalServerError)
		return
	}
}
