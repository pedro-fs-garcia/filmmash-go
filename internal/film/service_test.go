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
		winnerK    int16
		loserK     int16
		wantWinner float64
		wantLoser  float64
	}{
		{"equal ratings split the pot evenly", 1400, 1400, 20, 20, 1410, 1390},
		{"favorite winning gains little", 1600, 1400, 20, 20, 1604.8050614670408, 1395.1949385329592},
		{"underdog upset gains a lot", 1400, 1600, 20, 20, 1415.1949385329592, 1584.8050614670408},
		{"large gap favorite wins", 2000, 1000, 20, 20, 2000.0630461836652, 999.9369538163348},
		{"large gap underdog wins", 1000, 2000, 20, 20, 1019.9369538163348, 1980.0630461836652},
		{"small gap", 1500, 1450, 20, 20, 1508.5707376518324, 1441.4292623481676},
		{"new winner moves twice as fast", 1400, 1400, 40, 20, 1420, 1390},
		{"new loser drops twice as fast", 1400, 1400, 20, 40, 1410, 1380},
		{"new underdog upsets veteran", 1400, 1600, 40, 20, 1430.3898770659184, 1584.8050614670408},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotWinner, gotLoser := film.CalculateRatings(tc.winner, tc.loser, tc.winnerK, tc.loserK)
			if math.Abs(gotWinner-tc.wantWinner) > epsilon || math.Abs(gotLoser-tc.wantLoser) > epsilon {
				t.Errorf("CalculateRatings(%g, %g, %d, %d) = (%v, %v); want (%v, %v)",
					tc.winner, tc.loser, tc.winnerK, tc.loserK, gotWinner, gotLoser, tc.wantWinner, tc.wantLoser)
			}

			won := gotWinner - tc.winner
			lost := tc.loser - gotLoser
			// Both deltas come from the same expected score, so they must be
			// proportional to each film's K (zero-sum when the K's are equal).
			if math.Abs(won*float64(tc.loserK)-lost*float64(tc.winnerK)) > epsilon {
				t.Errorf("deltas not proportional to K: winner gained %g (K=%d), loser lost %g (K=%d)",
					won, tc.winnerK, lost, tc.loserK)
			}

			if won < 0 || lost < 0 {
				t.Errorf("rating moved the wrong way: winner delta %g, loser delta %g", won, -lost)
			}
		})
	}
}

func TestFilmKFromDuelCount(t *testing.T) {
	cases := []struct {
		duelCount int32
		want      int16
	}{
		{0, 40},
		{1, 39},
		{19, 21},
		{20, 20},
		{100, 20},
	}
	for _, tc := range cases {
		if got := film.FilmKFromDuelCount(tc.duelCount); got != tc.want {
			t.Errorf("FilmKFromDuelCount(%d) = %d, want %d", tc.duelCount, got, tc.want)
		}
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
