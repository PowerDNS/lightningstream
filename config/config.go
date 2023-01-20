// Package config implements the YAML config file parser
package config

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"powerdns.com/platform/lightningstream/config/logger"
	"powerdns.com/platform/lightningstream/lmdbenv"
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
)

// Config is the config root object
type Config struct {
	Instance string          `yaml:"instance"`
	LMDBs    map[string]LMDB `yaml:"lmdbs"`
	Storage  Storage         `yaml:"storage"`
	HTTP     HTTP            `yaml:"http"`
	Log      logger.Config   `yaml:"log"`

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

	// LMDBScrapeSmaps enabled the scraping of /proc/smaps for LMDB stats
	LMDBScrapeSmaps bool `yaml:"lmdb_scrape_smaps"`

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
	// - Every value is prefixed with an 8-byte big-endian timestamp containing
	//   the number of *nanoseconds* since the UNIX epoch that it was last modified.
	// - Deleted entries are recorded with the same timestamp and an empty value.
	// When enabled, a shadow database is no longer needed to merge snapshots and
	// conflict resolution is both more accurate and more efficient, but do note
	// that THIS MUST BE SUPPORTED IN THE LMDB SCHEMA THE APPLICATION USES!
	SchemaTracksChanges bool `yaml:"schema_tracks_changes"`

	// Enables hacky support for DupSort DBs, with limitations.
	// This will be applied to all dbs marked as DupSort.
	// Not compatible with schema_tracks_changes=true
	DupSortHack bool `yaml:"dupsort_hack"`

	// Stats logging options
	ScrapeSmaps      bool          `yaml:"scrape_smaps"` // Reading proc smaps can be expensive in some situations
	LogStats         bool          `yaml:"log_stats"`
	LogStatsInterval time.Duration `yaml:"log_stats_interval"`
}

type DBIOptions struct {
	// No longer has any options, but may get them in the future
}

type Storage struct {
	Type    string                 `yaml:"type"`    // "fs", "s3", "memory"
	Options map[string]interface{} `yaml:"options"` // backend specific

	Cleanup Cleanup `yaml:"cleanup"`

	RootPath string `yaml:"root_path,omitempty"` // Deprecated: use options.root_path for fs
}

// Cleanup contains storage cleanup configuration. When enabled, this will clean
// old snapshots for any instance, not just itself.
type Cleanup struct {
	Enabled bool `yaml:"enabled"`

	// Interval determines how often we run a cleaning session.
	// The actual interval is subject to intentional perturbation.
	Interval time.Duration

	// MustKeepInterval determines how long snapshots must be kept after they
	// appear in the bucket, even if a newer snapshot is available.
	// This is a fairly short period (typically less than an hour) that is just
	// long enough to give clients time to download this snapshot after they
	// perform a listing, even if the snapshot is large and the connection slow.
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
		if l.LogStats && l.LogStatsInterval < 100*time.Millisecond {
			return fmt.Errorf("lmdb.log_stats_interval: too short interval")
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
	if c.StorageRetryCount < 1 {
		return fmt.Errorf("storage_retry_count: positive number required")
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

		LMDBScrapeSmaps:      true,
		LMDBPollInterval:     DefaultLMDBPollInterval,
		LMDBLogStatsInterval: DefaultLMDBLogStatsInterval,
		StoragePollInterval:  DefaultStoragePollInterval,
		StorageRetryInterval: DefaultStorageRetryInterval,
		StorageRetryCount:    DefaultStorageRetryCount,

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
