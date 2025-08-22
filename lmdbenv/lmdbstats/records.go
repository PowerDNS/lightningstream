package lmdbstats

import "time"

// Record describes the records stored in LMDB if persistence is enabled
type Record struct {
	// Time and Duration define the time span for the transaction
	Time     time.Time     `json:"time"` // start time
	Duration time.Duration `json:"duration"`

	// Success and Error tell you if the action succeeded.
	// Error is only stored when explicitly enabled, because it requires
	// a separate write transaction that may also fail.
	Success bool   `json:"success"`
	Error   string `json:"error"`

	// Stats are the collected stats at the end of the transaction
	Stats Stats `json:"stats"`

	// Optional application info to keep track of what happened
	App      string `json:"app,omitempty"`
	Instance string `json:"instance,omitempty"`
	Action   string `json:"action,omitempty"`
	Details  string `json:"details,omitempty"`
	// Meta are any key-value pairs useful in the context of the application,
	// e.g. original size of data loaded into LMDB, source, etc.
	Meta map[string]any `json:"meta,omitempty"`

	// MonitorStartTime is the time we started monitoring
	MonitorStartTime time.Time `json:"monitor_start_time"`
}
