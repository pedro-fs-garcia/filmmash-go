package duel_test

import (
	"filmmash/internal/duel"
	"testing"
)

func TestDrawFromCandidates(t *testing.T) {
	t.Parallel()
	uniform := func(duel.Candidate) float64 { return 1.0 }

	t.Run("empty pool reports no draw", func(t *testing.T) {
		_, ok := duel.DrawFromCandidates(nil, uniform)
		if ok {
			t.Fatal("got ok=true for empty candidates, want false")
		}
	})

	t.Run("single candidate is always drawn", func(t *testing.T) {
		got, ok := duel.DrawFromCandidates([]duel.Candidate{{Id: 7}}, uniform)
		if !ok || got != 7 {
			t.Fatalf("got (%d, %v), want (7, true)", got, ok)
		}
	})

	t.Run("zero-weight candidates are never drawn", func(t *testing.T) {
		candidates := []duel.Candidate{{Id: 1}, {Id: 42}, {Id: 3}}
		weight := func(c duel.Candidate) float64 {
			if c.Id == 42 {
				return 1.0
			}
			return 0.0
		}
		for range 200 {
			got, ok := duel.DrawFromCandidates(candidates, weight)
			if !ok || got != 42 {
				t.Fatalf("got (%d, %v), want (42, true)", got, ok)
			}
		}
	})

	t.Run("all-zero weights still draw a candidate", func(t *testing.T) {
		candidates := []duel.Candidate{{Id: 1}, {Id: 2}}
		got, ok := duel.DrawFromCandidates(candidates, func(duel.Candidate) float64 { return 0.0 })
		if !ok || (got != 1 && got != 2) {
			t.Fatalf("got (%d, %v), want one of the candidates and true", got, ok)
		}
	})

	t.Run("draws are proportional to weight", func(t *testing.T) {
		candidates := []duel.Candidate{{Id: 1}, {Id: 2}, {Id: 7}}
		weight := func(c duel.Candidate) float64 { return float64(c.Id) }

		const n = 20000
		counts := map[int32]int{}
		for range n {
			got, ok := duel.DrawFromCandidates(candidates, weight)
			if !ok {
				t.Fatal("got ok=false, want true")
			}
			counts[got]++
		}

		// Expected proportions 0.1/0.2/0.7; ±0.05 is ~15 standard errors at
		// n=20000, so a failure means a broken sampler, not bad luck.
		for _, c := range candidates {
			want := float64(c.Id) / 10.0
			got := float64(counts[c.Id]) / n
			if got < want-0.05 || got > want+0.05 {
				t.Errorf("candidate %d drawn %.3f of the time, want %.2f ± 0.05", c.Id, got, want)
			}
		}
	})
}
