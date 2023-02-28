package syncer

import (
	"fmt"
	"time"

	"github.com/PowerDNS/simpleblob"
	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/syncer/cleaner"

	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/status/healthtracker"
	"powerdns.com/platform/lightningstream/status/starttracker"
)

func New(name string, st simpleblob.Interface, c config.Config, lc config.LMDB, opt Options) (*Syncer, error) {
	l := logrus.WithField("db", name)

	// Start cleaner, but make sure it is disabled if we run in receive-only mode
	var cleanupConf config.Cleanup
	if opt.ReceiveOnly {
		// Disabled when in receive-only mode.
		// This is the default, just setting this for clarity.
		cleanupConf.Enabled = false
	} else {
		cleanupConf = c.Storage.Cleanup
	}
	cl := cleaner.New(name, st, cleanupConf, l)

	s := &Syncer{
		name:               name,
		st:                 st,
		c:                  c,
		lc:                 lc,
		opt:                opt,
		shadow:             true,
		generation:         0,
		lastByInstance:     make(map[string]time.Time),
		cleaner:            cl,
		storageStoreHealth: healthtracker.New(c.Health.StorageStore, fmt.Sprintf("%s_storage_store", name), "write to storage backend"),
		startTracker:       starttracker.New(c.Health.Start, name),
	}
	if s.instanceID() == "" {
		return nil, fmt.Errorf("instance name could not be determined, please provide one with --instance")
	}
	s.l = l.WithField("instance", s.instanceID())
	if !lc.SchemaTracksChanges {
		s.l.Info("This LMDB has schema_tracks_changes disabled and will use " +
			"shadow databases for version tracking.")
	} else {
		s.l.Info("schema_tracks_changes enabled")
	}
	s.l.Info("Initialised syncer")
	return s, nil
}

type Syncer struct {
	name       string // database name
	st         simpleblob.Interface
	c          config.Config
	lc         config.LMDB
	opt        Options
	l          logrus.FieldLogger
	shadow     bool // use shadow database for timestamps?
	generation uint64

	// lastByInstance tracks the last snapshot loaded by instance, so that the
	// cleaner can make safe decisions about when to remove stale snapshots.
	lastByInstance map[string]time.Time

	// cleaner cleans old snapshots in the background
	cleaner *cleaner.Worker

	// Health trackers
	storageStoreHealth *healthtracker.HealthTracker
	startTracker       *starttracker.StartTracker
}
