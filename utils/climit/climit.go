package climit

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// New creates a new ConcurrencyLimit with a given limit.
// The name is used for Prometheus metrics.
func New(dbname, name string, limit int, logger logrus.FieldLogger) *ConcurrencyLimit {
	if logger == nil {
		lr := logrus.New()
		lr.SetLevel(logrus.PanicLevel) // never reached
		logger = lr
	}
	logger = logger.WithField("limit_name", name).WithField("dbname", dbname)
	if limit < 1 {
		logger.Warnf(
			"Increasing concurrency limit from configured %d to minimum of 1", limit)
		limit = 1
	}
	l := &ConcurrencyLimit{
		dbname: dbname,
		name:   name,
		labels: prometheus.Labels{
			"lmdb":       dbname,
			"limit_name": name,
		},
		ch:  make(chan internalToken, limit),
		log: logger,
	}
	for i := 0; i < limit; i++ {
		l.ch <- internalToken{}
	}
	metricLimit.With(l.labels).Set(float64(limit))
	return l
}

// ConcurrencyLimit enforce a concurrency limit with tokens that need to be held
// by routines.
// A Token can be acquired by calling Acquire(), and MUST be released by
// calling Token.Release().
type ConcurrencyLimit struct {
	dbname string
	name   string
	labels prometheus.Labels
	ch     chan internalToken
	log    logrus.FieldLogger
}

type internalToken struct{}

// Acquire acquires a Token. It will block until a free Token is available.
// You MUST call Token.Release() when you are done with the operation.
func (cl *ConcurrencyLimit) Acquire() *Token {
	cl.log.Debug("Acquiring token")
	metricWaiting.With(cl.labels).Inc()
	t0 := time.Now()
	it := <-cl.ch
	dt := time.Since(t0)

	metricWaiting.With(cl.labels).Dec()
	metricActive.With(cl.labels).Inc()
	metricAcquiredTotal.With(cl.labels).Inc()
	metricWaitingSeconds.With(cl.labels).Observe(dt.Seconds())

	token := &Token{
		cl:              cl,
		token:           it,
		time:            time.Now(),
		acquireDuration: dt,
	}
	cl.log.WithField("time_to_acquire", dt).Debug("Acquired token")
	return token
}

// Token represents the token that allows the caller to proceed with a limited
// operation.
type Token struct {
	cl              *ConcurrencyLimit
	time            time.Time
	acquireDuration time.Duration

	mu       sync.Mutex
	released bool
	token    internalToken
}

// Release releases the Token.
// It can safely be called more than once, even from different goroutines.
// It returns how long the Token was held, or 0 if it had already been released.
func (t *Token) Release() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.released {
		return 0
	}
	t.cl.ch <- t.token
	t.released = true
	dt := time.Since(t.time)
	metricActive.With(t.cl.labels).Dec()
	metricActiveSeconds.With(t.cl.labels).Observe(dt.Seconds())
	t.cl.log.Debug("Released token")
	t.cl = nil
	return dt
}
