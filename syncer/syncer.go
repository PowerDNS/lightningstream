package syncer

import (
	"fmt"

	"github.com/PowerDNS/simpleblob"
	"github.com/sirupsen/logrus"

	"powerdns.com/platform/lightningstream/config"
)

func New(name string, st simpleblob.Interface, c config.Config, lc config.LMDB) (*Syncer, error) {
	s := &Syncer{
		name:       name,
		st:         st,
		c:          c,
		lc:         lc,
		shadow:     true,
		generation: 0,
	}
	if s.instanceID() == "" {
		return nil, fmt.Errorf("instance name could not be determined, please provide one with --instance")
	}
	s.l = logrus.WithField("db", name).WithField("instance", s.instanceID())
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
	name       string
	st         simpleblob.Interface
	c          config.Config
	lc         config.LMDB
	l          logrus.FieldLogger
	shadow     bool // use shadow database for timestamps?
	generation uint64
}
