package tmdb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListUsesOneDetailsCallForDirectorAndImdb(t *testing.T) {
	details := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/movie/top_rated", "/movie/popular":
			w.Write([]byte(`{"results":[{"id":42,"title":"Film"}]}`))
		case "/movie/42":
			details++
			if r.URL.Query().Get("append_to_response") != "credits" {
				t.Error("credits not appended")
			}
			w.Write([]byte(`{"id":42,"title":"Film","imdb_id":"tt12345","credits":{"crew":[{"id":7,"name":"Director","job":"Director"}]}}`))
		default:
			t.Error("unexpected extra call: " + r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := NewClient(server.URL, "test")
	for _, fetch := range []func(int16) ([]Movie, error){client.GetTopRated, client.GetPopulars} {
		movies, err := fetch(1)
		if err != nil || len(movies) != 1 || movies[0].ImdbId != "tt12345" || movies[0].Director.Name != "Director" {
			t.Fatalf("mapping failed: %v %#v", err, movies)
		}
	}
	if details != 2 {
		t.Fatalf("expected one details call per film, got %d", details)
	}
}
func TestNullImdb(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"id":1,"imdb_id":null}`)) }))
	defer server.Close()
	m, err := NewClient(server.URL, "test").GetFilm(1)
	if err != nil || strings.TrimSpace(m.ImdbId) != "" {
		t.Fatal("null IMDB should be empty")
	}
}
