package healthtracker

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

type HealthConfig struct {
	EvaluationInterval time.Duration `yaml:"interval"`
	ErrorDuration      time.Duration `yaml:"error_duration"`
	WarnDuration       time.Duration `yaml:"warn_duration"`
}

func (hc HealthConfig) Validated() HealthConfig {
	// Enforce MinEvaluationInterval
	if hc.EvaluationInterval < MinEvaluationInterval {
		hc.EvaluationInterval = MinEvaluationInterval
	}

	// Enforce MinErrorDuration
	if hc.ErrorDuration < MinErrorDuration {
		hc.ErrorDuration = MinErrorDuration
	}

	// Enforce MinWarnDuration
	if hc.WarnDuration < MinWarnDuration {
		hc.WarnDuration = MinWarnDuration
	}

	return hc
}
