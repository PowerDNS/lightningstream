// Package config implements the YAML config file parser
package config

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"powerdns.com/platform/lightningstream/config/logger"
	"powerdns.com/platform/lightningstream/lmdbenv"
)

// DefaultLMDBLogStatsInterval is the default interval for logging LMDB stats
const DefaultLMDBLogStatsInterval = time.Minute

// DefaultFetchTimeout is the default timeout for an HTTP fetch.
// This timeout is a very conservative protection against a stuck
// download. In practise this should never happen, because Go also
// uses TCP keepalive for HTTP TCP connections, but it prevents
// a hung process if the server keeps the TCP connection
// open but stops sending data for some reason.
// At 4 hours and a snapshot size of 1.4 GB, it this assumes
// a transfer speed of at least 100 kB/s (800 kbit/s).
const DefaultFetchTimeout = 4 * time.Hour

// Config is the config root object
type Config struct {
	RunOnce bool            `yaml:"run_once"` // Exit after a single run // FIXME: needed?
	LMDBs   map[string]LMDB `yaml:"lmdbs"`
	Storage Storage         `yaml:"storage"`
	HTTP    HTTP            `yaml:"http"`
	Log     logger.Config   `yaml:"log"`

	DefaultPollInterval time.Duration `yaml:"default_poll_interval"` // FIXME: move

	// Set to current version by main
	Version string `yaml:"-"`
}

// LMDB configures the LMDB database
type LMDB struct {
	Path             string          `yaml:"path"` // Path to directory holding data.mdb, or mdb file if NoSubdir
	Options          lmdbenv.Options `yaml:"options"`
	ScrapeSmaps      bool            `yaml:"scrape_smaps"` // Reading proc smaps can be expensive in some situations
	LogStats         bool            `yaml:"log_stats"`
	LogStatsInterval time.Duration   `yaml:"log_stats_interval"`
}

type Storage struct {
	Type     string `yaml:"type"`
	RootPath string `yaml:"root_path,omitempty"` // for the 'fs' backend
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
	}
	if c.HTTP.Address != "" {
		if _, _, err := net.SplitHostPort(c.HTTP.Address); err != nil {
			return fmt.Errorf("http.address: %v", err)
		}
	}
	if c.DefaultPollInterval < 100*time.Millisecond {
		return fmt.Errorf("default_poll_interval: too short interval")
	}
	return nil
}

// String returns the config as a YAML string with passwords masked.
func (c Config) String() string {
	y, err := yaml.Marshal(c)
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
	contents, err := ioutil.ReadFile(fpath)
	if err != nil {
		return errors.Wrap(err, "open yaml file")
	}
	return c.LoadYAML(contents, expandEnv)
}

// Default returns a Config with default settings
func Default() Config {
	return Config{
		Log: logger.DefaultConfig,

		// Default poll intervals
		DefaultPollInterval: 5 * time.Second,
	}
}
