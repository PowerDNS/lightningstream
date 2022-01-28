package logger

import (
	"flag"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	LogLevels     = []string{"debug", "info", "warning", "error", "fatal"}
	LogFormats    = []string{"human", "logfmt", "json"}
	LogTimestamps = []string{"short", "disable", "full"}
)

// Config configures logging
type Config struct {
	Level     string `yaml:"level"`     // One of LogLevels
	Format    string `yaml:"format"`    // One of LogFormats
	Timestamp string `yaml:"timestamp"` // One of LogTimestamps
}

// DefaultConfig defines the default configuration
var DefaultConfig = Config{
	Level:     "info",
	Format:    "human",
	Timestamp: "short",
}

// FlagConfig captures flag values and defaults to zero values
var FlagConfig = Config{}

// RegisterFlags registers several log flags.
// The default values are set to their zero value to allow detecting when
// the flag has been set. This allows the use of a config file for logging
// and overriding it with these flags.
func RegisterFlags() {
	RegisterFlagsWith(flag.StringVar)
}

// StringVarFlagFunc has the signature of flag.StringVar
type StringVarFlagFunc func(*string, string, string, string)

// RegisterFlagsWith uses a specific function to register the flags with,
// allowing it to be used with different flag packages, like Cobra.
func RegisterFlagsWith(stringVar StringVarFlagFunc) {
	stringVar(&FlagConfig.Level, "log-level", "", "Log level "+
		addDefaults(DefaultConfig.Level, LogLevels))
	stringVar(&FlagConfig.Format, "log-format", "", "Log format "+
		addDefaults(DefaultConfig.Format, LogFormats))
	stringVar(&FlagConfig.Timestamp, "log-timestamp", "", "Log timestamp "+
		addDefaults(DefaultConfig.Timestamp, LogTimestamps))
}

// Check validates a Config instance
func (c Config) Check() error {
	if _, err := logrus.ParseLevel(c.Level); err != nil {
		return fmt.Errorf("log.level: must be one of: %s", strings.Join(LogLevels, ", "))
	}
	if !inList(LogFormats, c.Format) {
		return fmt.Errorf("log.format: must be one of: %s", strings.Join(LogFormats, ", "))
	}
	if c.Timestamp != "" {
		if !inList(LogTimestamps, c.Timestamp) {
			return fmt.Errorf("log.timestamp: must be one of: %s", strings.Join(LogTimestamps, ", "))
		}
	}
	return nil
}

// Merge merges a Config with another Config, returning the new combined Config.
// This is useful for merging in values set by flags.
func (c Config) Merge(o Config) Config {
	if o.Level != "" {
		c.Level = o.Level
	}
	if o.Format != "" {
		c.Format = o.Format
	}
	if o.Timestamp != "" {
		c.Timestamp = o.Timestamp
	}
	return c
}

// Configure configures logrus according to Config
func Configure(c Config) {
	noTimestamp := c.Timestamp == "disable"
	fullTimestamp := c.Timestamp == "full"

	var formatter logrus.Formatter
	switch c.Format {
	case "json":
		formatter = &logrus.JSONFormatter{DisableTimestamp: noTimestamp}
	case "logfmt":
		formatter = &logrus.TextFormatter{
			DisableColors:    true, // this sets logfmt
			DisableTimestamp: noTimestamp,
			FullTimestamp:    fullTimestamp,
		}
	case "human":
		formatter = &NamespaceFormatter{
			Parent: &logrus.TextFormatter{
				DisableColors:    false,
				DisableTimestamp: noTimestamp,
				FullTimestamp:    fullTimestamp,
			},
		}
	}
	logrus.SetFormatter(formatter)

	level, err := logrus.ParseLevel(c.Level)
	if err != nil {
		// Should have been validated before calling this
		logrus.Warnf("Ignoring invalid log level: %s", c.Level)
	} else {
		logrus.SetLevel(level)
	}
}

func addDefaults(def string, options []string) string {
	return fmt.Sprintf("(default: %s; options: %s)", def, strings.Join(options, ", "))
}

func inList(list []string, item string) bool {
	for _, v := range list {
		if item == v {
			return true
		}
	}
	return false
}
