package metrics

import "github.com/prometheus/client_golang/prometheus"

type DuelMetrics struct {
	created prometheus.Counter
	pending prometheus.Gauge
}

func NewDuelMetrics(reg prometheus.Registerer) *DuelMetrics {
	m := &DuelMetrics{
		created: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "filmmash_duels_created_total",
			Help: "Created duels.",
		}),
		pending: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "filmmash_pending_duels",
			Help: "Duels without a registered vote.",
		}),
	}
	reg.MustRegister(m.created, m.pending)
	return m
}

func (m *DuelMetrics) DuelCreated() {
	m.created.Inc()
	m.pending.Inc()
}

func (m *DuelMetrics) DuelResolved() {
	m.pending.Dec()
}

func (m *DuelMetrics) SeedPending(n int) {
	m.pending.Set(float64(n))
}
