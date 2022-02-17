package syncer

import (
	"github.com/prometheus/client_golang/prometheus"
	"powerdns.com/platform/lightningstream/lmdbenv/stats"
)

var (
	lmdbCollector *stats.Collector
)

func init() {
	lmdbCollector = stats.NewCollector(false)
	prometheus.MustRegister(lmdbCollector)
}
