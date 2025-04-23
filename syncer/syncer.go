package syncer

import (
	"fmt"
	"time"

	"github.com/PowerDNS/lightningstream/syncer/cleaner"
	"github.com/PowerDNS/lightningstream/syncer/events"
	"github.com/PowerDNS/lightningstream/syncer/hooks"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/simpleblob"
	"github.com/sirupsen/logrus"

	"github.com/PowerDNS/lightningstream/config"
	"github.com/PowerDNS/lightningstream/status/healthtracker"
	"github.com/PowerDNS/lightningstream/status/starttracker"
)

func New(name string, env *lmdb.Env, st simpleblob.Interface, c config.Config, lc config.LMDB, opt Options) (*Syncer, error) {
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

	ev := opt.Events
	if ev == nil {
		ev = events.New()
	}
	h := opt.Hooks
	if h == nil {
		h = hooks.New()
	}

	s := &Syncer{
		name:               name,
		st:                 st,
		c:                  c,
		lc:                 lc,
		opt:                opt,
		shadow:             true,
		generation:         0,
		env:                env,
		events:             ev,
		hooks:              h,
		lastByInstance:     make(map[string]time.Time),
		lastSnapshotTime:   time.Time{}, // zero
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
	env        *lmdb.Env
	events     *events.Events
	hooks      *hooks.Hooks

	// lastByInstance tracks the last snapshot loaded by instance, so that the
	// cleaner can make safe decisions about when to remove stale snapshots.
	lastByInstance map[string]time.Time

	// lastSnapshotTime is the last time we generated a snapshot, used to force
	// a new one
	lastSnapshotTime time.Time

	// cleaner cleans old snapshots in the background
	cleaner *cleaner.Worker

	// Health trackers
	storageStoreHealth *healthtracker.HealthTracker
	startTracker       *starttracker.StartTracker
}
