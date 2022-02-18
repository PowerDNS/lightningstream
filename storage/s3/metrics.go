package s3

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricLastCallTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "storage_s3_call_timestamp_seconds",
			Help: "UNIX timestamp of last S3 API call by method",
		},
		[]string{"method"},
	)
	metricCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "storage_s3_call_total",
			Help: "S3 API calls by method",
		},
		[]string{"method"},
	)
	metricCallErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "storage_s3_call_error_total",
			Help: "S3 API call errors by method",
		},
		[]string{"method"},
	)
)

func init() {
	prometheus.MustRegister(metricLastCallTimestamp)
	prometheus.MustRegister(metricCalls)
	prometheus.MustRegister(metricCallErrors)
}
