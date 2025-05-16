package sweeper

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricCleanedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lightningstream_sweeper_cleaned_total",
			Help: "Number of stale deletion markers cleaned by the sweeper",
		},
		[]string{"lmdb"},
	)
	metricStatsAvailable = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_sweeper_stats_available",
			Help: "Set to 1 when the stats are available after a sweep run",
		},
		[]string{"lmdb"},
	)
	metricStatsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_sweeper_stats_total_entries",
			Help: "Total entries after last sweeper run, including deleted",
		},
		[]string{"lmdb"},
	)
	metricStatsDeleted = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_sweeper_stats_deleted_entries",
			Help: "Deleted entries after last sweeper run",
		},
		[]string{"lmdb"},
	)
	metricDurationSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "lightningstream_sweeper_duration_seconds",
			Help: "Summary of time taken by sweeper",
		},
		[]string{"lmdb"},
	)
)

func init() {
	prometheus.MustRegister(metricCleanedTotal)
	prometheus.MustRegister(metricStatsAvailable)
	prometheus.MustRegister(metricStatsTotal)
	prometheus.MustRegister(metricStatsDeleted)
	prometheus.MustRegister(metricDurationSummary)
}
