package healthtracker

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wojas/go-healthz"
	"go.uber.org/atomic"
)

type HealthTracker struct {
	Config   HealthConfig
	sequence atomic.Uint32
	since    atomic.Time
	lastErr  atomic.String
	prefix   string
	activity string
	logger   logrus.FieldLogger
}

func New(hc HealthConfig, prefix string, activity string) *HealthTracker {
	ht := &HealthTracker{
		Config:   hc.Validated(),
		prefix:   prefix,
		activity: activity,
		logger:   logrus.WithField("healthtracker", prefix),
	}

	ht.sequence.Store(0)

	// Register duration tracker to healthz
	ht.RegisterDuration()

	return ht
}

func (ht *HealthTracker) RegisterDuration() {
	// Register healthz
	healthz.Register(fmt.Sprintf("%s_failed_duration", ht.prefix), ht.Config.EvaluationInterval, func() error {
		// Current values
		conseqFails := ht.sequence.Load()
		failingSince := ht.since.Load()
		lastErr := ht.lastErr.Load()

		// Calculated values
		failingFor := time.Since(failingSince)

		if conseqFails > 0 {
			if failingFor >= ht.Config.ErrorDuration {
				ht.logger.Warnf("failure for %s is violating the error threshold (%s)", failingFor.Round(time.Second), ht.Config.ErrorDuration)

				return fmt.Errorf("failed to %s for %s - last error: '%s'", ht.activity, failingFor.Round(time.Second), lastErr)
			} else if failingFor >= ht.Config.WarnDuration {
				ht.logger.Warnf("failure for %s is violating the warning threshold (%s)", failingFor.Round(time.Second), ht.Config.WarnDuration)

				return healthz.Warnf("failed to %s for %s - last error: '%s'", ht.activity, failingFor.Round(time.Second), lastErr)
			}
		}

		return nil
	})

	ht.logger.Info("registered tracker for failure duration")
}

func (ht *HealthTracker) AddFailure(err error) {
	ht.lastErr.Store(err.Error())

	failures := ht.sequence.Load()
	if failures == 0 {
		ht.since.Store(time.Now())
	}

	ht.sequence.Inc()

	ht.logger.Debugf("tracked failed attempt")
}

func (ht *HealthTracker) AddSuccess() {
	ht.sequence.Store(0)
	ht.lastErr.Store("")

	ht.logger.Debug("tracked successful attempt")
}
