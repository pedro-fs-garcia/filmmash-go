package view_test

import (
	"bytes"
	"filmmash/internal/film"
	"filmmash/internal/view"
	"strings"
	"testing"
)

func TestIMDBLinkOnlyWhenStored(t *testing.T) {
	for _, test := range []struct{ cache, name string }{{"filmPage", "base"}, {"filmModal", "filmModal"}} {
		for _, id := range []string{"tt1234567", ""} {
			var out bytes.Buffer
			if err := view.TemplateCache[test.cache].ExecuteTemplate(&out, test.name, film.Film{Title: "Fixture", ImdbId: id}); err != nil {
				t.Fatal(err)
			}
			present := strings.Contains(out.String(), "https://www.imdb.com/title/")
			if present != (id != "") {
				t.Fatalf("%s: empty IMDB link condition failed", test.cache)
			}
			if present && (!strings.Contains(out.String(), `target="_blank"`) || !strings.Contains(out.String(), `rel="noopener noreferrer"`)) {
				t.Fatal("external link protections missing")
			}
		}
	}
}
