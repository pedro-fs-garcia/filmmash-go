package film_test

import (
	"filmmash/internal/film"
	"testing"
)

func TestCalculateRatings(t *testing.T) {
	w, l := film.CalculateRatings(1400, 1400)
	if w != 1410 || l != 1390 {
		t.Errorf("CalculateRating(1400, 1400) = %f, %f; want %f, %f", w, l, 1410.0, 1390.0)
	}
}
