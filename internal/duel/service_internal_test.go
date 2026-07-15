package duel

import "testing"

func TestCandidateWeight(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  ServiceConfig
		c    Candidate
		want float64
	}{
		{
			name: "baseline weight is 1",
			cfg:  ServiceConfig{},
			c:    Candidate{},
			want: 1.0,
		},
		{
			name: "popularity scales the weight",
			cfg:  ServiceConfig{PopularityWeight: 2},
			c:    Candidate{Popularity: 3},
			want: 7.0,
		},
		{
			name: "zero popularity weight ignores popularity",
			cfg:  ServiceConfig{},
			c:    Candidate{Popularity: 100},
			want: 1.0,
		},
		{
			name: "boost applies below the duel threshold",
			cfg:  ServiceConfig{NewFilmBoost: 3, NewFilmDuelThreshold: 10},
			c:    Candidate{DuelCount: 9},
			want: 3.0,
		},
		{
			name: "no boost at the duel threshold",
			cfg:  ServiceConfig{NewFilmBoost: 3, NewFilmDuelThreshold: 10},
			c:    Candidate{DuelCount: 10},
			want: 1.0,
		},
		{
			name: "zero-value boost keeps new films selectable",
			cfg:  ServiceConfig{NewFilmDuelThreshold: 10},
			c:    Candidate{DuelCount: 0},
			want: 1.0,
		},
		{
			name: "boost compounds with popularity",
			cfg:  ServiceConfig{PopularityWeight: 2, NewFilmBoost: 3, NewFilmDuelThreshold: 10},
			c:    Candidate{Popularity: 3, DuelCount: 0},
			want: 21.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{cfg: tt.cfg}
			if got := s.candidateWeight(tt.c); got != tt.want {
				t.Errorf("candidateWeight(%+v) = %v, want %v", tt.c, got, tt.want)
			}
		})
	}
}
