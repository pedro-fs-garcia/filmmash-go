package main

import (
	"filmmash/internal/database"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type HomeData struct {
	Title   string
	Message string
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	t := template.Must(
		template.ParseFiles("templates/base.html", "templates/index.html"),
	)

	err := t.ExecuteTemplate(w, "base", nil)

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func IdHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	t := template.Must(template.ParseFiles("templates/base.html", "templates/id.html"))

	err := t.ExecuteTemplate(w, "base", map[string]string{"Id": id})
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func HandleHealth(w http.ResponseWriter, r *http.Request) {
	db := database.OpenDB()
	if err := db.Ping(); err != nil {
		w.Write([]byte("database offline"))
		return
	}
	log.Println("Database connected")
	db.Close()
	w.Write([]byte("ok"))
}

func FilmHandler(w http.ResponseWriter, r *http.Request) {
	par := chi.URLParam(r, "id")
	id, err := strconv.Atoi(par)
	if err != nil {
		http.Error(w, "Invalid Id", http.StatusBadRequest)
		return
	}

	film, err := GetFilm(id)
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
