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

func (v *VoteMetrics) VoteRecorded(vote string) {
	v.total.WithLabelValues(vote).Inc()
}
