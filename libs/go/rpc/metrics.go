package rpc

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const MetricsPath = "/metrics"

var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rpc_server_requests_total",
		Help: "Connect and gRPC requests handled, by procedure and result code.",
	}, []string{"procedure", "code"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "rpc_server_request_duration_seconds",
		Help:    "Time spent handling a Connect or gRPC request, by procedure.",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"procedure"})
)

func record(procedure, code string, elapsed time.Duration) {
	requestsTotal.WithLabelValues(procedure, code).Inc()
	requestDuration.WithLabelValues(procedure).Observe(elapsed.Seconds())
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
