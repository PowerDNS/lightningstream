package syncer

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/storage"
)

func New(name string, st storage.Interface, c config.Config, lc config.LMDB) (*Syncer, error) {
	s := &Syncer{
		name:       name,
		st:         st,
		c:          c,
		lc:         lc,
		l:          logrus.WithField("db", name),
		shadow:     true,
		generation: 0,
	}
	if s.instanceName() == "" {
		return nil, fmt.Errorf("instance name could not be determined, please provide one with --instance")
	}
	return s, nil
}

type Syncer struct {
	name       string
	st         storage.Interface
	c          config.Config
	lc         config.LMDB
	l          logrus.FieldLogger
	shadow     bool // use shadow database for timestamps?
	generation uint64
}
