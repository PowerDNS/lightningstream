package cleaner

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricListCalls = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lightningstream_cleaner_list_calls_total",
			Help: "Number of cleaner list calls",
		},
	)
	metricListFailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lightningstream_cleaner_list_failed_total",
			Help: "Number of cleaner failed list attempts",
		},
	)
	metricDeleteCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lightningstream_cleaner_delete_total",
			Help: "Number of cleaner delete calls",
		},
		[]string{"lmdb", "reason"},
	)
	metricDeleteFailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "lightningstream_cleaner_delete_failed_total",
			Help: "Number of failed cleaner delete calls",
		},
	)
)

func init() {
	prometheus.MustRegister(metricListCalls)
	prometheus.MustRegister(metricListFailed)
	prometheus.MustRegister(metricDeleteCalls)
	prometheus.MustRegister(metricDeleteFailed)
}
