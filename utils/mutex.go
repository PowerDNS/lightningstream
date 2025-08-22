package utils

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const MonitoredMutexDefaultLimit = time.Second

// MonitoredMutex warns on unlocking when a lock was held too long
type MonitoredMutex struct {
	mu       sync.Mutex
	lockTime time.Time

	Logger logrus.FieldLogger
	Name   string
}

func (m *MonitoredMutex) Lock() {
	m.mu.Lock()
	m.lockTime = time.Now()
}

func (m *MonitoredMutex) Unlock() {
	timeHeld := time.Since(m.lockTime)
	m.lockTime = time.Time{}
	m.mu.Unlock()

	limit := MonitoredMutexDefaultLimit
	if timeHeld > limit {
		// No panic, because time jumps, paused processes and sleep may
		// cause spikes.
		var caller string
		pc, fileName, fileLine, ok := runtime.Caller(1)
		if ok {
			details := runtime.FuncForPC(pc)
			if details != nil {
				funcName := details.Name()
				caller = fmt.Sprintf("%s:%d (%s)", fileName, fileLine, funcName)
			}
		}
		m.logger().WithFields(logrus.Fields{
			"lock_held": timeHeld,
			"limit":     limit,
			"lock_name": m.Name,
			"caller":    caller,
		}).Warn("Lock time limit exceeded")
	}
}

func (m *MonitoredMutex) logger() logrus.FieldLogger {
	if m.Logger != nil {
		return m.Logger
	}
	return logrus.StandardLogger()
}
