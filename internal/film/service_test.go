package film_test

import (
	"filmmash/internal/film"
	"math"
	"reflect"
	"testing"
)

func TestCalculateRatings(t *testing.T) {
	const epsilon = 1e-6
	cases := []struct {
		name       string
		winner     float64
		loser      float64
		wantWinner float64
		wantLoser  float64
	}{
		{"equal ratings split the pot evenly", 1400, 1400, 1410, 1390},
		{"favorite winning gains little", 1600, 1400, 1604.8050614670408, 1395.1949385329592},
		{"underdog upset gains a lot", 1400, 1600, 1415.1949385329592, 1584.8050614670408},
		{"large gap favorite wins", 2000, 1000, 2000.0630461836652, 999.9369538163348},
		{"large gap underdog wins", 1000, 2000, 1019.9369538163348, 1980.0630461836652},
		{"small gap", 1500, 1450, 1508.5707376518324, 1441.4292623481676},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotWinner, gotLoser := film.CalculateRatings(tc.winner, tc.loser)
			if math.Abs(gotWinner-tc.wantWinner) > epsilon || math.Abs(gotLoser-tc.wantLoser) > epsilon {
				t.Errorf("CalculateRatings(%g, %g) = (%v, %v); want (%v, %v)",
					tc.winner, tc.loser, gotWinner, gotLoser, tc.wantWinner, tc.wantLoser)
			}

			won := gotWinner - tc.winner
			lost := tc.loser - gotLoser
			if math.Abs(won-lost) > epsilon {
				t.Errorf("not zero-sum: winner gained %g, loser lost %g", won, lost)
			}

			if won < 0 || lost < 0 {
				t.Errorf("rating moved the wrong way: winner delta %g, loser delta %g", won, -lost)
			}
		})
	}
}

func TestToPaginatedResponse(t *testing.T) {
	cases := map[string]struct {
		size  int
		films []film.Film
		want  film.PaginatedResponse
	}{
		"empty films yields zero response with no cursor": {
			size:  3,
			films: nil,
			want:  film.PaginatedResponse{},
		},
		"cursor is built from the last film": {
			size: 2,
			films: []film.Film{
				{Id: 2001, Rating: 1500},
				{Id: 1234, Rating: 1340},
			},
			want: film.PaginatedResponse{
				Films: []film.Film{
					{Id: 2001, Rating: 1500},
					{Id: 1234, Rating: 1340},
				},
				Next: film.PaginationParameters{
					Size:           2,
					LastSeenId:     1234,
					LastSeenRating: 1340,
				},
			},
		},
		"single film cursor points at that film": {
			size: 5,
			films: []film.Film{
				{Id: 42, Rating: 1400},
			},
			want: film.PaginatedResponse{
				Films: []film.Film{
					{Id: 42, Rating: 1400},
				},
				Next: film.PaginationParameters{
					Size:           5,
					LastSeenId:     42,
					LastSeenRating: 1400,
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := film.ToPaginatedResponse(tc.size, tc.films)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ToPaginatedResponse(%d, %v) = %+v, want %+v", tc.size, tc.films, got, tc.want)
			}
		})
	}
}
