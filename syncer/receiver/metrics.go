package receiver

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	m = 60
	h = 60 * m
	d = 24 * h
)

var (
	metricSnapshotsLastReceivedTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_receiver_snapshots_last_received_seconds",
			Help: "UNIX timestamp of last received snapshot by instance",
		},
		[]string{"lmdb", "syncer_instance"},
	)
	metricSnapshotsLastReceivedAge = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "lightningstream_receiver_snapshots_last_received_age_seconds",
			Help: "Age of last received snapshot by instance in seconds",
			Buckets: []float64{
				0.1, 0.2, 0.5, 1, 2, 5, 10, 30, // seconds
				1 * m, 2 * m, 5 * m, 10 * m, 30 * m, // minutes
				1 * h, 2 * h, 6 * h, 12 * h, // hours
				1 * d, 3 * d, 7 * d, 14 * d, 30 * d, // days
			},
		},
		[]string{"lmdb", "syncer_instance"},
	)
	metricSnapshotsLoadCalls = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lightningstream_syncer_snapshots_load_calls_total",
			Help: "Number of snapshot load calls",
		},
	)
	metricSnapshotsListCalls = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lightningstream_syncer_snapshots_list_calls_total",
			Help: "Number of snapshot list calls",
		},
	)
	metricSnapshotsLoadFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lightningstream_syncer_snapshots_load_failed_total",
			Help: "Number of snapshot failed load attempts",
		},
		[]string{"lmdb", "syncer_instance"},
	)
	metricSnapshotsListFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lightningstream_syncer_snapshots_list_failed_total",
			Help: "Number of snapshot failed list attempts",
		},
		[]string{"lmdb"},
	)
	metricSnapshotsLoadBytes = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lightningstream_syncer_snapshots_load_bytes_total",
			Help: "Number of bytes downloaded successfully",
		},
	)
	metricSnapshotsStorageCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_receiver_snapshots_count_in_storage",
			Help: "Number of snapshots in storage",
		},
		[]string{"lmdb"},
	)
	metricSnapshotsStorageBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_receiver_snapshots_last_bytes_in_storage",
			Help: "Number of bytes occupied by snapshots in storage",
		},
		[]string{"lmdb"},
	)
)

func init() {
	prometheus.MustRegister(metricSnapshotsLastReceivedTimestamp)
	prometheus.MustRegister(metricSnapshotsLastReceivedAge)
	prometheus.MustRegister(metricSnapshotsLoadCalls)
	prometheus.MustRegister(metricSnapshotsListCalls)
	prometheus.MustRegister(metricSnapshotsLoadFailed)
	prometheus.MustRegister(metricSnapshotsListFailed)
	prometheus.MustRegister(metricSnapshotsLoadBytes)
	prometheus.MustRegister(metricSnapshotsStorageCount)
	prometheus.MustRegister(metricSnapshotsStorageBytes)
}
