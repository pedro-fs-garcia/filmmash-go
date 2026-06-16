package metrics

import "github.com/prometheus/client_golang/prometheus"

type FilmMetrics struct {
	total prometheus.Gauge
}

func NewFilmMetrics(reg prometheus.Registerer) *FilmMetrics {
	m := &FilmMetrics{
		total: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "filmmash_films_total",
			Help: "Total films in catalog.",
		}),
	}
	reg.MustRegister(m.total)
	return m
}

func (m *FilmMetrics) FilmInserted() {
	m.total.Inc()
}

func (m *FilmMetrics) SeedCurrentTotal(n int64) {
	m.total.Set(float64(n))
}
