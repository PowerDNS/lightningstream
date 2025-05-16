package sweeper

import (
	"math"
	"time"

	"github.com/sirupsen/logrus"
)

type stats struct {
	nEntries  int // total number of entries (including deleted)
	nDeleted  int // total number of entries that are still marked as deleted
	nCleaned  int // total number of deleted entries that have been cleaned in this run
	nTxn      int // number of transactions
	timeTaken time.Duration
}

func (s stats) logFields() logrus.Fields {
	return logrus.Fields{
		"total_entries":    s.nEntries,
		"deleted_entries":  s.nDeleted,
		"deleted_fraction": s.deletedFraction(),
		"cleaned_entries":  s.nCleaned,
		"transactions":     s.nTxn,
		"time_taken":       s.timeTaken.Round(time.Millisecond),
	}
}

func (s stats) deletedFraction() float64 {
	var deletedFraction float64
	if s.nEntries > 0 {
		deletedFraction = float64(s.nDeleted) / float64(s.nEntries)
		deletedFraction = math.Round(deletedFraction*1000) / 1000
	}
	return deletedFraction
}
