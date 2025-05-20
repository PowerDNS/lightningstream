// Package config implements the YAML config file parser
package config

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/PowerDNS/lightningstream/lmdbenv/dbiflags"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/PowerDNS/lightningstream/config/logger"
	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lightningstream/status/healthtracker"
	"github.com/PowerDNS/lightningstream/status/starttracker"
)

const (
	// DefaultLMDBLogStatsInterval is the default interval for logging LMDB stats
	DefaultLMDBLogStatsInterval = 30 * time.Minute

	// DefaultLMDBPollInterval is the minimum time between checking for new LMDB
	// transactions. The check itself is fast, but this also serves to rate limit
	// the creation of new snapshots.
	DefaultLMDBPollInterval = time.Second

	// DefaultStoragePollInterval is the minimum time between polling the storage
	// backend for new snapshots.
	DefaultStoragePollInterval = time.Second

	// DefaultStorageRetryInterval is interval to retry a storage operation
	// after failure.
	DefaultStorageRetryInterval = 5 * time.Second

	// DefaultStorageRetryCount is the number of times to retry a storage operation
	// after failure, before giving up.
	DefaultStorageRetryCount = 100

	// DefaultStorageForceSnapshotInterval is the default interval at which we
	// write a snapshot even if no local changes were detected.
	DefaultStorageForceSnapshotInterval = 4 * time.Hour

	// DefaultMemoryDownloadedSnapshots is the number of downloaded compressed
	// snapshots we can keep in memory.
	DefaultMemoryDownloadedSnapshots = 2

	// DefaultMemoryDecompressedSnapshots is the number of decompressed snapshots
	// we can keep in memory.
	DefaultMemoryDecompressedSnapshots = 3
)

var (
	// DefaultHealthStorageList is the default set of thresholds used by healthz to determine health of storage list operations
	DefaultHealthStorageList = healthtracker.HealthConfig{
		// ErrorDuration is the duration after which a failing List operation will report 'error' to healthz
		ErrorDuration: 5 * time.Minute,
		// WarnDuration is the duration after which a failing List operation will report 'warning' to healthz
		WarnDuration: 1 * time.Minute,
		// EvaluationInterval is the interval between healthz evaluation of the List operation
		EvaluationInterval: 5 * time.Second,
	}

	// DefaultHealthStorageLoad is the default set of thresholds used by healthz to determine health of storage load operations
	DefaultHealthStorageLoad = healthtracker.HealthConfig{
		// ErrorDuration is the duration after which a failing Load operation will report 'error' to healthz
		ErrorDuration: 5 * time.Minute,
		// WarnDuration is the duration after which a failing Load operation will report 'warning' to healthz
		WarnDuration: 1 * time.Minute,
		// EvaluationInterval is the interval between healthz evaluation of the Load operation
		EvaluationInterval: 5 * time.Second,
	}

	// DefaultHealthStorageStore is the default set of thresholds used by healthz to determine health of storage store operations
	DefaultHealthStorageStore = healthtracker.HealthConfig{
		// ErrorDuration is the duration after which a failing Store operation will report 'error' to healthz
		ErrorDuration: 5 * time.Minute,
		// WarnDuration is the duration after which a failing Store operation will report 'warning' to healthz
		WarnDuration: 1 * time.Minute,
		// EvaluationInterval is the interval between healthz evaluation of the Store operation
		EvaluationInterval: 5 * time.Second,
	}

	// DefaultHealthStart is the default set of thresholds used by healthz to determine whether the startup phase has completed successfully
	DefaultHealthStart = starttracker.StartConfig{
		// ErrorDuration is the duration after which a failing startup sequence will report 'error' to healthz
		ErrorDuration: 5 * time.Minute,
		// WarnDuration is the duration after which a failing startup sequence will report 'warning' to healthz
		WarnDuration: 1 * time.Minute,
		// EvaluationInterval is the interval between healthz evaluation of the startup sequence
		EvaluationInterval: 1 * time.Second,
		// ReportHealthz controls whether or not a failing startup sequence will be included in healthz's overall status
		// This can be used to prevent unwanted activity before Lightning Stream has completed an initial sync
		ReportHealthz: false,
		// ReportHealthz controls whether or not healthz's 'startup_[db name]' metadata field will be used to store the status of the startup sequence for each db
		// This can be used to prevent unwanted activity before Lightning Stream has completed an initial sync
		ReportMetadata: true,
	}
)

// Config is the config root object
type Config struct {
	Instance string          `yaml:"instance"`
	LMDBs    map[string]LMDB `yaml:"lmdbs"`
	Sweeper  Sweeper         `yaml:"sweeper"`
	Storage  Storage         `yaml:"storage"`
	HTTP     HTTP            `yaml:"http"`
	Log      logger.Config   `yaml:"log"`
	Health   Health          `yaml:"health"`

	// LMDBPollInterval is the minimum time between checking for new LMDB
	// transactions. The check itself is fast, but this also serves to rate limit
	// the creation of new snapshots. Checking for actual changes once a new
	// transaction is detected also requires a full database scan, and a merge
	// with shadow databases when schema_tracks_changes is false.
	LMDBPollInterval time.Duration `yaml:"lmdb_poll_interval"`

	// LMDBLogStatsInterval is the interval to log LMDB stats. Set to 0 to disable.
	LMDBLogStatsInterval time.Duration `yaml:"lmdb_log_stats_interval"`

	// StoragePollInterval is the minimum time between polling the storage backend
	// for new snapshots. This can be set quite low, but keep in mind that loading
	// a new snapshot can also trigger writing a new snapshot when
	// schema_tracks_changes is false and shadow databases are used.
	StoragePollInterval time.Duration `yaml:"storage_poll_interval"`

	// StorageRetryInterval is interval to retry a storage operation
	// after failure.
	StorageRetryInterval time.Duration `yaml:"storage_retry_interval"`

	// StorageRetryCount is the number of times to retry a storage operation
	// after failure, before giving up.
	StorageRetryCount int `yaml:"storage_retry_count"`

	// If set, StorageRetryCount will be ignored, and we retry forever
	StorageRetryForever bool `yaml:"storage_retry_forever"`

	// StorageForceSnapshotInterval sets the interval to force a snapshot write
	// even if no LMDB changes were detected, to make sure we occasionally write
	// a fresh snapshot.
	StorageForceSnapshotInterval time.Duration `yaml:"storage_force_snapshot_interval"`

	// MemoryDownloadedSnapshots defines how many downloaded compressed snapshots
	// we are allowed to keep in memory for each database (minimum: 1, default: 3).
	// Setting this higher allows us to keep downloading snapshots for different
	// instances, even if one download is experiencing a hiccup.
	// These will transition to 'memory_decompressed_snapshots' once a slot opens
	// up in there.
	// Increasing this can speed up processing at the cost of memory.
	MemoryDownloadedSnapshots int `yaml:"memory_downloaded_snapshots"`

	// MemoryDecompressedSnapshots defines how many decompressed snapshots
	// we are allowed to keep in memory for each database (minimum: 1, default: 2).
	// Keep in mind that decompressed snapshots are typically 3-10x larger than
	// the downloaded compressed snapshots.
	// Increasing this can speed up processing at the cost of memory.
	MemoryDecompressedSnapshots int `yaml:"memory_decompressed_snapshots"`

	// LMDBScrapeSmaps enabled the scraping of /proc/smaps for LMDB stats
	LMDBScrapeSmaps bool `yaml:"lmdb_scrape_smaps"`

	// OnlyOnce requests the program to exit ofter once batch has been completed,
	// e.g. after the initial listing of snapshots have been loaded.
	OnlyOnce bool `yaml:"only_once"`

	// Set to current version by main
	Version string `yaml:"-"`
}

// LMDB configures the LMDB database
type LMDB struct {
	// Basic LMDB options
	Path    string          `yaml:"path"` // Path to directory holding data.mdb, or mdb file if NoSubdir
	Options lmdbenv.Options `yaml:"options"`

	// Per-DBI options
	DBIOptions map[string]DBIOptions `yaml:"dbi_options"`

	// Both important and dangerous: set to true if the LMDB schema already tracks
	// changes in the exact way that this tool expects. This includes:
	// - Every value is prefixed with an 24+ byte LS header.
	// - Deleted entries are recorded with the same timestamp and a Deleted flag.
	// When enabled, a shadow database is no longer needed to merge snapshots and
	// conflict resolution is both more accurate and more efficient, but do note
	// that THIS MUST BE SUPPORTED IN THE LMDB SCHEMA THE APPLICATION USES!
	SchemaTracksChanges bool `yaml:"schema_tracks_changes"`

	// Enables hacky support for DupSort DBs, with limitations.
	// This will be applied to all dbs marked as DupSort.
	// Not compatible with schema_tracks_changes=true
	DupSortHack bool `yaml:"dupsort_hack"`

	// HeaderExtraPaddingBlock adds an extra 8 all-zero bytes to the LS header
	// to make it 32 bytes. This is useful to test an application's handling of
	// the numExtra header field. This does not apply to shadow tables.
	HeaderExtraPaddingBlock bool `yaml:"header_extra_padding_block"`
}

// Sweeper settings for the LMDB sweeper that removed deleted entries after
// a while, also known as the "tomb sweeper".
//
// The key consideration for these settings is how long instance can be
// expected to be disconnected from the storage (out of sync) before
// rejoining. If the retention interval is set too low, old records that
// have been removed during the downtime can reappear, which can cause
// major issues.
//
// When picking a value, also take into account development, testing and
// migration systems that only occasionally come online.
//
//	     TODO: Consider if we want to write a marker to keep track of the last
//			   sync, and reject sync once we have passed this interval.
//			   This would be the first instance of retained LS state. Up until
//			   now LS operates in a stateless way.
type Sweeper struct {
	// Enabled controls if the sweeper is enabled.
	// It is DISABLED by default, because of the important consistency
	// considerations that depend on the kind of deployment.
	// When disabled, the deleted entries will never actually be removed.
	Enabled bool `yaml:"enabled"`

	// RetentionDays is the number of DAYS of retention. Unlike in most
	// other places, this is specified in number of days instead of Duration
	// because of the expected length of this.
	// This is a float, so it is possible to use periods shorter than one day,
	// but this is rarely a good idea. Best to set this as high as possible.
	// Default: 370 (days, intentionally on the safe side)
	RetentionDays float32 `yaml:"retention_days"`

	// Interval is the interval between sweeps of the whole database to enforce
	// RetentionDays.
	// As a guideline, on a fast server sweeping 1 million records takes
	// about 1 second.
	// Default: 6h
	Interval time.Duration `yaml:"interval"`

	// FirstInterval is the first Interval immediately after
	// startup, to allow one soon after extended downtime.
	// Default: 10m
	FirstInterval time.Duration `yaml:"first_interval"`

	// LockDuration limits how long the sweeper may hold the exclusive write
	// lock at one time. This effectively controls the maximum latency spike
	// due to the sweeper for API calls that update the LMDB.
	// This is not a hard quota, the sweeper may overrun it slightly.
	// Default: 50ms
	LockDuration time.Duration `yaml:"lock_duration"`

	// ReleaseDuration determines how long the sweeper must sleep before it
	// is allowed to reacquire the exclusive write lock.
	// If this is equal to LockDuration, it means that the sweeper can hold the
	// LMDB at most half the time.
	// Do not set this too high, as every sweep cycle will record a write
	// transaction that can trigger a snapshot generation scan. It is best
	// to get it over with in a short total sweep time.
	// Default: 50ms
	ReleaseDuration time.Duration `yaml:"release_duration"`
}

func (sw Sweeper) RetentionDuration() time.Duration {
	return time.Duration(sw.RetentionDays * float32(24*time.Hour))
}

type DBIOptions struct {
	// OverrideCreateFlags can override DBI create flags when loading a
	// snapshot and the DBI does not create yet.
	// By default, we use the flags stored in the snapshot.
	// Snapshot with formatVersion < 3 did not store the right flags when
	// shadow tables were in use and may require this override.
	//
	// ONLY USE THIS WHEN YOU ARE SURE YOU NEED IT!
	OverrideCreateFlags *dbiflags.Flags `yaml:"override_create_flags"`
}

type Storage struct {
	Type    string                 `yaml:"type"`    // "fs", "s3", "memory"
	Options map[string]interface{} `yaml:"options"` // backend specific

	// FIXME: Configure per LMDB instead, since we run a cleaner per LMDB?
	Cleanup Cleanup `yaml:"cleanup"`

	RootPath string `yaml:"root_path,omitempty"` // Deprecated: use options.root_path for fs
}

// Cleanup contains storage cleanup configuration. When enabled, this will clean
// old snapshots for any instance, not just itself.
type Cleanup struct {
	Enabled bool `yaml:"enabled"`

	// Interval determines how often we run a cleaning session.
	// The actual interval is subject to intentional perturbation.
	Interval time.Duration `yaml:"interval"`

	// MustKeepInterval determines how long snapshots must be kept after they
	// appear in the bucket, even if a newer snapshot is available.
	// This is a fairly short period (typically less than an hour) that is just
	// long enough to give clients time to download this snapshot after they
	// perform a listing, even if the snapshot is large and the connection is slow.
	// Note that the latest snapshot will be retained as long as no new one
	// comes in for the same instance.
	MustKeepInterval time.Duration `yaml:"must_keep_interval"`

	// RemoveOldInstancesInterval determines when an instance is considered
	// stale and the latest snapshot for the instance can be considered for
	// removal. The actual removal only happens when the current instance has
	// loaded and merged that snapshot, and written its own snapshot with this
	// data, to ensure that this is also safe after extended downtime.
	RemoveOldInstancesInterval time.Duration `yaml:"remove_old_instances_interval"`
}

// HTTP configures the HTTP server with Prometheus metrics and status page
type HTTP struct {
	Address string `yaml:"address"` // Address like ":8000"
}

// Health configures the healthz error & warn thresholds
type Health struct {
	StorageList  healthtracker.HealthConfig `yaml:"storage_list"`
	StorageLoad  healthtracker.HealthConfig `yaml:"storage_load"`
	StorageStore healthtracker.HealthConfig `yaml:"storage_store"`
	Start        starttracker.StartConfig   `yaml:"start"`
}

// Check validates a Config instance
func (c Config) Check() error {
	if err := c.Log.Check(); err != nil {
		return err
	}
	if len(c.LMDBs) < 1 {
		return fmt.Errorf("no LMDBs configured")
	}
	for name, l := range c.LMDBs {
		prefix := fmt.Sprintf("lmdb %q", name)
		if l.Path == "" {
			return fmt.Errorf("%s: no path configured", prefix)
		}
		if l.Options.FileMask > 0777 { // decimal 511
			return fmt.Errorf("lmdb.options.file_mask: too large value, possible use of decimal (%d) instead of octal (%#o)",
				l.Options.FileMask, l.Options.FileMask)
		}
		if l.Options.DirMask > 0777 { // decimal 511
			return fmt.Errorf("lmdb.options.dir_mask: too large value, possible use of decimal (%d) instead of octal (%#o)",
				l.Options.DirMask, l.Options.DirMask)
		}
		if l.SchemaTracksChanges && l.DupSortHack {
			return fmt.Errorf("lmdb.schema_tracks_changes: cannot be used together with the dupsort_hack option")
		}
	}
	if c.HTTP.Address != "" {
		if _, _, err := net.SplitHostPort(c.HTTP.Address); err != nil {
			return fmt.Errorf("http.address: %v", err)
		}
	}
	if c.LMDBPollInterval < 100*time.Millisecond {
		return fmt.Errorf("lmdb_poll_interval: too short interval")
	}
	if c.StoragePollInterval < 100*time.Millisecond {
		return fmt.Errorf("storage_poll_interval: too short interval")
	}
	if c.StorageRetryInterval < 100*time.Millisecond {
		return fmt.Errorf("storage_retry_interval: too short interval")
	}
	if dt := c.StorageForceSnapshotInterval; dt != 0 && dt < time.Minute {
		return fmt.Errorf("storage_force_snapshot_interval: too short interval (minimum 1m if enabled)")
	}
	if c.StorageRetryCount < 1 {
		return fmt.Errorf("storage_retry_count: positive number required")
	}
	if c.MemoryDownloadedSnapshots < 1 {
		return fmt.Errorf("memory_downloaded_snapshots: positive number required")
	}
	if c.MemoryDecompressedSnapshots < 1 {
		return fmt.Errorf("memory_decompressed_snapshots: positive number required")
	}
	return nil
}

func (c Config) Clone() Config {
	y, err := yaml.Marshal(c)
	if err != nil {
		logrus.Panicf("YAML marshal of config failed: %v", err) // Should never happen
	}
	var newConfig Config
	err = yaml.Unmarshal(y, &newConfig)
	if err != nil {
		logrus.Panicf("YAML unmarshal of config failed: %v", err) // Should never happen
	}
	return newConfig
}

// String returns the config as a YAML string with passwords masked.
func (c Config) String() string {
	cc := c.Clone()
	if cc.Storage.Options != nil {
		opt := cc.Storage.Options
		for _, key := range []string{"secret_key", "secret", "password"} {
			iv := opt[key]
			if v, ok := iv.(string); ok && v != "" {
				opt[key] = "***"
			}
		}
	}
	y, err := yaml.Marshal(cc)
	if err != nil {
		logrus.Panicf("YAML marshal of config failed: %v", err) // Should never happen
	}
	return string(y)
}

// LoadYAML loads config from YAML. Any set value overwrites any existing value,
// but omitted keys are untouched.
func (c *Config) LoadYAML(yamlContents []byte, expandEnv bool) error {
	if expandEnv {
		yamlContents = []byte(os.ExpandEnv(string(yamlContents)))
	}
	return yaml.UnmarshalStrict(yamlContents, c)
}

// LoadYAMLFile loads config from a YAML file. Any set value overwrites any existing value,
// but omitted keys are untouched.
func (c *Config) LoadYAMLFile(fpath string, expandEnv bool) error {
	contents, err := os.ReadFile(fpath)
	if err != nil {
		return errors.Wrap(err, "open yaml file")
	}
	return c.LoadYAML(contents, expandEnv)
}

// Default returns a Config with default settings
func Default() Config {
	return Config{
		Log: logger.DefaultConfig,
		Health: Health{
			StorageList:  DefaultHealthStorageList,
			StorageLoad:  DefaultHealthStorageLoad,
			StorageStore: DefaultHealthStorageStore,
			Start:        DefaultHealthStart,
		},

		LMDBScrapeSmaps:              true,
		LMDBPollInterval:             DefaultLMDBPollInterval,
		LMDBLogStatsInterval:         DefaultLMDBLogStatsInterval,
		StoragePollInterval:          DefaultStoragePollInterval,
		StorageRetryInterval:         DefaultStorageRetryInterval,
		StorageRetryCount:            DefaultStorageRetryCount,
		StorageForceSnapshotInterval: DefaultStorageForceSnapshotInterval,
		MemoryDownloadedSnapshots:    DefaultMemoryDownloadedSnapshots,
		MemoryDecompressedSnapshots:  DefaultMemoryDecompressedSnapshots,

		Sweeper: Sweeper{
			Enabled:         false,
			RetentionDays:   370, // days
			Interval:        6 * time.Hour,
			FirstInterval:   10 * time.Minute,
			LockDuration:    50 * time.Millisecond,
			ReleaseDuration: 50 * time.Millisecond,
		},

		Storage: Storage{
			Cleanup: Cleanup{
				Enabled:                    false, // TODO: Enable by default in future
				Interval:                   5 * time.Minute,
				MustKeepInterval:           10 * time.Minute,
				RemoveOldInstancesInterval: 7 * 24 * time.Hour,
			},
		},
	}
}
