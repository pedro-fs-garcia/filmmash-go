package metrics

import "github.com/prometheus/client_golang/prometheus"

type VoteMetrics struct {
	total *prometheus.CounterVec
}

func NewVote(reg prometheus.Registerer) *VoteMetrics {
	v := &VoteMetrics{
		total: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "filmmash_votes_total",
				Help: "Total votes recorded",
			},
			[]string{"vote"},
		),
	}
	reg.MustRegister(v.total)
	return v
}

func (m *VoteMetrics) VoteRegistered(vote string) {
	m.total.WithLabelValues(vote).Inc()
}

func (m *VoteMetrics) SeedTotal(n int) {
	m.total.WithLabelValues("preexisting votes").Add(float64(n))
}
