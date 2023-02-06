package starttracker

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wojas/go-healthz"
	"go.uber.org/atomic"
)

type StartTracker struct {
	Config                StartConfig
	initialListing        atomic.Bool
	initialStore          atomic.Bool
	initialReceiveAndLoad atomic.Bool
	since                 atomic.Time
	prefix                string
	metaFieldStartup      string
	logger                logrus.FieldLogger
}

func New(sc StartConfig, prefix string) *StartTracker {
	st := &StartTracker{
		Config:           sc.Validated(),
		prefix:           prefix,
		metaFieldStartup: fmt.Sprintf("startup_%s", prefix),
		logger:           logrus.WithField("starttracker", prefix),
	}

	// Default startup state to false (not finished)
	st.initialListing.Store(false)
	st.initialStore.Store(false)
	st.initialReceiveAndLoad.Store(false)

	// Set current time as begin of startup phase
	st.since.Store(time.Now())

	// Register tracker to healthz
	st.RegisterTracker()

	return st
}

func (st *StartTracker) RegisterTracker() {
	if st.Config.ReportMetadata {
		healthz.SetMeta(st.metaFieldStartup, false)
	}

	trackerName := fmt.Sprintf("%s_startup_in_progress", st.prefix)

	// Register healthz
	healthz.Register(trackerName, st.Config.EvaluationInterval, func() error {
		// Calculated values
		failingFor := time.Since(st.since.Load())

		// Tests for pending activities
		if !st.initialListing.Load() || !st.initialStore.Load() || !st.initialReceiveAndLoad.Load() {
			if st.Config.ReportHealthz {
				if failingFor >= st.Config.ErrorDuration {
					st.logger.Debugf("successful startup pending after %s is violating the error threshold (%s)", failingFor.Round(time.Second), st.Config.ErrorDuration)

					return fmt.Errorf("successful startup pending after %s", failingFor.Round(time.Second))
				} else if failingFor >= st.Config.WarnDuration {
					st.logger.Debugf("successful startup pending after %s is violating the warning threshold (%s)", failingFor.Round(time.Second), st.Config.WarnDuration)

					return healthz.Warnf("successful startup pending after %s", failingFor.Round(time.Second))
				}
			}

			return nil
		}

		// Handle finished startup phase
		if st.Config.ReportMetadata {
			healthz.SetMeta(st.metaFieldStartup, true)
		}

		st.logger.Info("startup phase completed successfully")

		// Deregister the tracker - startup phase is irrelevant after passing once
		healthz.Deregister(trackerName)

		return nil
	})

	st.logger.Info("registered tracker for startup phase")
}

func (st *StartTracker) SetPassedInitialListing() {
	st.initialListing.Store(true)

	st.logger.Debug("tracked successful initial listing")
}

func (st *StartTracker) SetPassedInitialStore() {
	st.initialStore.Store(true)

	st.logger.Debug("tracked successful initial snapshot store")
}

func (st *StartTracker) SetPassedInitialReceiveAndLoad() {
	if !st.initialReceiveAndLoad.Load() {
		st.initialReceiveAndLoad.Store(true)

		st.logger.Debug("tracked successful initial receive & load")
	}
}
