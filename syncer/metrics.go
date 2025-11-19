package syncer

import (
	"github.com/PowerDNS/lightningstream/lmdbenv/stats"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	lmdbCollector *stats.Collector

	metricSnapshotsLoaded = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lightningstream_syncer_snapshots_generated_total",
			Help: "Number of snapshots generated",
		},
		[]string{"lmdb"},
	)
	metricSnapshotsLastTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_syncer_snapshots_generated_last_unix_seconds",
			Help: "UNIX timestamp of last generated snapshot",
		},
		[]string{"lmdb"},
	)
	metricSnapshotsLastAge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_syncer_last_snapshot_age_in_seconds",
			Help: "Age of last generated snapshot in seconds",
		},
		[]string{"lmdb"},
	)
	metricSnapshotsLastSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_syncer_snapshots_generated_last_size_bytes",
			Help: "Size of last generated snapshot in bytes",
		},
		[]string{"lmdb"},
	)
	metricSnapshotsStoreFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lightningstream_syncer_snapshots_store_failed_attempts_total",
			Help: "Number of snapshot failed store attempts",
		},
		[]string{"lmdb"},
	)
	metricSnapshotsStoreFailedPermanently = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lightningstream_syncer_snapshots_store_failed_permanently_total",
			Help: "Number of permanent snapshot store failures",
		},
		[]string{"lmdb"},
	)
	metricSnapshotsStoreCalls = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lightningstream_syncer_snapshots_store_calls_total",
			Help: "Number of snapshot store calls",
		},
	)
	metricSnapshotsStoreBytes = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lightningstream_syncer_snapshots_store_bytes_total",
			Help: "Number of bytes stored successfully",
		},
	)
	metricSnapshotsTimeStamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_syncer_snapshots_timestamp",
			Help: "Snapshot timestamp, used to identify it",
		},
		[]string{"lmdb", "syncer_instance", "timestamp_string"},
	)
	metricSnapshotsSyncerGenerationID = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_syncer_snapshots_generation_id",
			Help: "Snapshot generation ID, used to identify it",
		},
		[]string{"lmdb", "syncer_instance", "generation_id"},
	)
)

func init() {
	lmdbCollector = stats.NewCollector(false)
	prometheus.MustRegister(lmdbCollector)

	prometheus.MustRegister(metricSnapshotsLoaded)
	prometheus.MustRegister(metricSnapshotsLastTimestamp)
	prometheus.MustRegister(metricSnapshotsLastAge)
	prometheus.MustRegister(metricSnapshotsLastSize)
	prometheus.MustRegister(metricSnapshotsStoreFailed)
	prometheus.MustRegister(metricSnapshotsStoreFailedPermanently)
	prometheus.MustRegister(metricSnapshotsStoreCalls)
	prometheus.MustRegister(metricSnapshotsStoreBytes)
	prometheus.MustRegister(metricSnapshotsTimeStamp)
	prometheus.MustRegister(metricSnapshotsSyncerGenerationID)
}
