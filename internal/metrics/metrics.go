package metrics

import (
	"fmt"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry    *prometheus.Registry
	HttpMetrics *HttpMetrics
	VoteMetrics *VoteMetrics
}

func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return &Metrics{
		Registry:    reg,
		HttpMetrics: NewHttp(reg),
		VoteMetrics: NewVote(reg),
	}
}

func (m *Metrics) Mount(router *chi.Mux, root string) {
	router.Use(m.HttpMetrics.MetricsMiddleware)
	router.Handle(fmt.Sprintf("%s/metrics", root), promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))
}
