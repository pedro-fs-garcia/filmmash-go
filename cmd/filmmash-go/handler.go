package main

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type HomeData struct {
	Title   string
	Message string
}

type AppHandler struct {
	service *Service
}

func (h *AppHandler) FilmHandler(w http.ResponseWriter, r *http.Request) {
	par := chi.URLParam(r, "id")
	id, err := strconv.Atoi(par)
	if err != nil {
		http.Error(w, "Invalid Id", http.StatusBadRequest)
		return
	}

	film, err := h.service.GetFilm(r.Context(), id)
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
