package healthtracker

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wojas/go-healthz"
	"go.uber.org/atomic"
)

type HealthConfig struct {
	ErrorDuration      time.Duration `yaml:"error_duration"`
	WarnDuration       time.Duration `yaml:"warn_duration"`
	ErrorSequence      uint32        `yaml:"error_sequence"`
	WarnSequence       uint32        `yaml:"warn_sequence"`
	EvaluationInterval time.Duration `yaml:"interval"`
}

type HealthTracker struct {
	Config   HealthConfig
	sequence atomic.Uint32
	since    atomic.Time
	prefix   string
	activity string
	logger   logrus.FieldLogger
}

func New(hc HealthConfig, prefix string, activity string) *HealthTracker {
	ht := &HealthTracker{
		Config:   hc,
		prefix:   prefix,
		activity: activity,
		logger:   logrus.WithField("healthtracker", prefix),
	}

	ht.sequence.Store(0)

	ht.RegisterSequence()
	ht.RegisterDuration()

	return ht
}

func (ht *HealthTracker) RegisterSequence() {

	// Register healthz
	healthz.Register(fmt.Sprintf("%s_failed_attempts", ht.prefix), ht.Config.EvaluationInterval, func() error {
		// Current values
		conseqFails := ht.sequence.Load()

		if conseqFails >= ht.Config.ErrorSequence {
			ht.logger.Warnf("%d consecutive failures is violating the error threshold (%d)", conseqFails, ht.Config.ErrorSequence)

			return fmt.Errorf("failed to %s %d consecutive times", ht.activity, conseqFails)
		} else if conseqFails >= ht.Config.WarnSequence {
			ht.logger.Warnf("%d consecutive failures is violating the warning threshold (%d)", conseqFails, ht.Config.WarnSequence)

			return healthz.Warnf("failed to %s %d consecutive times", ht.activity, conseqFails)
		}

		return nil
	})

	ht.logger.Info("registered tracker for consecutive failures")
}

func (ht *HealthTracker) RegisterDuration() {
	// Register healthz
	healthz.Register(fmt.Sprintf("%s_failed_duration", ht.prefix), ht.Config.EvaluationInterval, func() error {
		// Current values
		conseqFails := ht.sequence.Load()
		failingSince := ht.since.Load()

		// Calculated values
		failingFor := time.Since(failingSince)

		if conseqFails > 0 {
			if failingFor >= ht.Config.ErrorDuration {
				ht.logger.Warnf("failure for %s is violating the error threshold (%s)", failingFor.Round(time.Second), ht.Config.ErrorDuration)

				return fmt.Errorf("failed to %s for %s", ht.activity, failingFor.Round(time.Second))
			} else if failingFor >= ht.Config.WarnDuration {
				ht.logger.Warnf("failure for %s is violating the warning threshold (%s)", failingFor.Round(time.Second), ht.Config.WarnDuration)

				return healthz.Warnf("failed to %s for %s", ht.activity, failingFor.Round(time.Second))
			}
		}

		return nil
	})

	ht.logger.Info("registered tracker for failure duration")
}

func (ht *HealthTracker) AddFailure() {
	failures := ht.sequence.Load()
	if failures == 0 {
		ht.since.Store(time.Now())
	}

	ht.sequence.Inc()

	ht.logger.Debugf("incremented consecutive failures to %d", failures)
}

func (ht *HealthTracker) AddSuccess() {
	ht.sequence.Store(0)

	ht.logger.Debug("tracked succesful attempt")
}
