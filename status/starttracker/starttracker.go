package starttracker

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wojas/go-healthz"
	"go.uber.org/atomic"
)

type StartTracker struct {
	Config         StartConfig
	initialListing atomic.Bool
	initialStore   atomic.Bool
	initialReceive atomic.Bool
	initialLoad    atomic.Bool
	since          atomic.Time
	prefix         string
	logger         logrus.FieldLogger
}

func New(sc StartConfig, prefix string) *StartTracker {
	st := &StartTracker{
		Config: sc.Validated(),
		prefix: prefix,
		logger: logrus.WithField("starttracker", prefix),
	}

	// Default startup state to false (not finished)
	st.initialListing.Store(false)
	st.initialStore.Store(false)
	st.initialReceive.Store(false)
	st.initialLoad.Store(false)

	// Set current time as begin of startup phase
	st.since.Store(time.Now())

	// Register tracker to healthz
	st.RegisterTracker()

	return st
}

func (st *StartTracker) RegisterTracker() {
	if st.Config.ReportMetadata {
		healthz.SetMeta("startupCompleted", false)
	}

	trackerName := fmt.Sprintf("%s_startup_in_progress", st.prefix)

	// Register healthz
	healthz.Register(trackerName, st.Config.EvaluationInterval, func() error {
		// Calculated values
		failingFor := time.Since(st.since.Load())

		if !st.initialListing.Load() || !st.initialStore.Load() || !st.initialReceive.Load() || !st.initialLoad.Load() {
			if st.Config.ReportHealthz {
				if failingFor >= st.Config.ErrorDuration {
					st.logger.Debugf("succesful startup pending after %s is violating the error threshold (%s)", failingFor.Round(time.Second), st.Config.ErrorDuration)

					return fmt.Errorf("succesful startup pending after %s", failingFor.Round(time.Second))
				} else if failingFor >= st.Config.WarnDuration {
					st.logger.Debugf("succesful startup pending after %s is violating the warning threshold (%s)", failingFor.Round(time.Second), st.Config.WarnDuration)

					return healthz.Warnf("succesful startup pending after %s", failingFor.Round(time.Second))
				}
			} else {
				return nil
			}
		}

		if st.Config.ReportMetadata {
			healthz.SetMeta("startupCompleted", true)
		}

		st.logger.Info("startup phase completed succesfully")

		// Deregister the tracker - startup phase is irrelevant after passing once
		healthz.Deregister(trackerName)

		return nil
	})

	st.logger.Info("registered tracker for startup phase")
}

func (st *StartTracker) SetPassedInitialListing() {
	st.initialListing.Store(true)

	st.logger.Debug("tracked succesful initial listing")
}

func (st *StartTracker) SetPassedInitialStore() {
	st.initialStore.Store(true)

	st.logger.Debug("tracked succesful initial snapshot store")
}

func (st *StartTracker) SetPassedInitialReceive() {
	st.initialReceive.Store(true)

	st.logger.Debug("tracked succesful initial receive")
}

func (st *StartTracker) SetPassedInitialLoad() {
	st.initialLoad.Store(true)

	st.logger.Debug("tracked succesful initial load")
}
