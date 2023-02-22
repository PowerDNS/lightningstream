package syncer

import (
	"fmt"
	"time"

	"github.com/PowerDNS/simpleblob"
	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/syncer/cleaner"

	"powerdns.com/platform/lightningstream/config"
)

func New(name string, st simpleblob.Interface, c config.Config, lc config.LMDB) (*Syncer, error) {
	l := logrus.WithField("db", name)
	cl := cleaner.New(name, st, c.Storage.Cleanup, l)
	s := &Syncer{
		name:           name,
		st:             st,
		c:              c,
		lc:             lc,
		shadow:         true,
		generation:     0,
		lastByInstance: make(map[string]time.Time),
		cleaner:        cl,
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
	l          logrus.FieldLogger
	shadow     bool // use shadow database for timestamps?
	generation uint64

	// lastByInstance tracks the last snapshot loaded by instance, so that the
	// cleaner can make safe decisions about when to remove stale snapshots.
	lastByInstance map[string]time.Time

	// cleaner cleans old snapshots in the background
	cleaner *cleaner.Worker
}
