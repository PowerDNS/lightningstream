package starttracker

import (
	"time"
)

const (
	// MinEvaluationInterval is the minimum interval allowed between healthz evaluation
	MinEvaluationInterval = time.Second

	// MinErrorDuration is the minimum duration before healthz evaluates a tracked item as failing
	MinErrorDuration = 0 * time.Second

	// MinWarnDuration is the minimum duration before healthz evaluates a tracked item as warning
	MinWarnDuration = 0 * time.Second
)

type StartConfig struct {
	EvaluationInterval time.Duration `yaml:"interval"`
	ErrorDuration      time.Duration `yaml:"error_duration"`
	WarnDuration       time.Duration `yaml:"warn_duration"`
	ReportHealthz      bool          `yaml:"report_healthz"`
	ReportMetadata     bool          `yaml:"report_metadata"`
}

func (sc StartConfig) Validated() StartConfig {
	// Enforce MinEvaluationInterval
	if sc.EvaluationInterval < MinEvaluationInterval {
		sc.EvaluationInterval = MinEvaluationInterval
	}

	// Enforce MinErrorDuration
	if sc.ErrorDuration < MinErrorDuration {
		sc.ErrorDuration = MinErrorDuration
	}

	// Enforce MinWarnDuration
	if sc.WarnDuration < MinWarnDuration {
		sc.WarnDuration = MinWarnDuration
	}

	return sc
}
