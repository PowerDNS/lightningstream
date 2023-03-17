package climit

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricLimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_climit_limit",
			Help: "COnfigured maximum number of tokens that can be active at once",
		},
		[]string{"lmdb", "limit_name"},
	)
	metricWaiting = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_climit_waiting",
			Help: "Number of tasks waiting to acquire the token",
		},
		[]string{"lmdb", "limit_name"},
	)
	metricActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_climit_active",
			Help: "Number of tasks currently active with the token",
		},
		[]string{"lmdb", "limit_name"},
	)
	metricAcquiredTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lightningstream_climit_acquired_total",
			Help: "Total number of times the token has been acquired",
		},
		[]string{"lmdb", "limit_name"},
	)
	metricActiveSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "lightningstream_climit_active_seconds",
			Help: "Histogram of how long tasks held the token",
		},
		[]string{"lmdb", "limit_name"},
	)
	metricWaitingSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "lightningstream_climit_waiting_seconds",
			Help: "Histogram of how long tasks have had to wait for the token",
		},
		[]string{"lmdb", "limit_name"},
	)
)

func init() {
	prometheus.MustRegister(metricLimit)
	prometheus.MustRegister(metricWaiting)
	prometheus.MustRegister(metricActive)
	prometheus.MustRegister(metricAcquiredTotal)
	prometheus.MustRegister(metricActiveSeconds)
	prometheus.MustRegister(metricWaitingSeconds)
}
