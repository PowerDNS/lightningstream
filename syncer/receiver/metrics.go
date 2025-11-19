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
	// called when we receive a new snapshot and download it to the instance
	metricSnapshotsLastReceivedSeconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_receiver_snapshots_last_received_seconds_diff",
			Help: "Seconds since last received snapshot by instance",
		},
		[]string{"lmdb", "syncer_instance"},
	)

	// Called when download a snapshot from storage. This is called all the time, since it's in the syncLoop
	metricSnapshotsLastDownloadedSeconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_receiver_snapshots_last_downloaded_seconds_diff",
			Help: "Seconds since last downloaded snapshot by instance",
		},
		[]string{"lmdb", "syncer_instance"},
	)

	metricSnapshotsTimeToDownloadFromStorage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_receiver_snapshots_time_to_download_from_storage_seconds",
			Help: "Time taken to download snapshot from storage by instance",
		},
		[]string{"lmdb", "syncer_instance"},
	)

	metricSnapshotsTimeToDownloadTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_receiver_snapshots_time_to_download_total_seconds",
			Help: "Total time taken to download snapshots by instance",
		},
		[]string{"lmdb", "syncer_instance"},
	)

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
			Name: "lightningstream_receiver_snapshots_in_storage_count",
			Help: "Number of snapshots in storage",
		},
		[]string{"lmdb"},
	)
	metricSnapshotsStorageBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_receiver_snapshots_in_storage_bytes",
			Help: "Number of bytes occupied by snapshots in storage",
		},
		[]string{"lmdb"},
	)

	metricSnapshotsTimestampString = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_receiver_snapshots_timestamp_string",
			Help: "String representation of the snapshot timestamp",
		},
		[]string{"lmdb", "syncer_instance", "timestamp_string"},
	)

	metricSnapshotsReceiverGenerationID = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lightningstream_receiver_snapshots_generation_id",
			Help: "ID of the snapshot generation, used to identify it",
		},
		[]string{"lmdb", "syncer_instance", "generation_id"},
	)
)

func init() {
	prometheus.MustRegister(metricSnapshotsLastReceivedTimestamp)
	prometheus.MustRegister(metricSnapshotsLastReceivedSeconds)
	prometheus.MustRegister(metricSnapshotsLastDownloadedSeconds)
	prometheus.MustRegister(metricSnapshotsTimeToDownloadFromStorage)
	prometheus.MustRegister(metricSnapshotsTimeToDownloadTotal)
	prometheus.MustRegister(metricSnapshotsLastReceivedAge)
	prometheus.MustRegister(metricSnapshotsLoadCalls)
	prometheus.MustRegister(metricSnapshotsListCalls)
	prometheus.MustRegister(metricSnapshotsLoadFailed)
	prometheus.MustRegister(metricSnapshotsListFailed)
	prometheus.MustRegister(metricSnapshotsLoadBytes)
	prometheus.MustRegister(metricSnapshotsStorageCount)
	prometheus.MustRegister(metricSnapshotsStorageBytes)
	prometheus.MustRegister(metricSnapshotsTimestampString)
	prometheus.MustRegister(metricSnapshotsReceiverGenerationID)
}
