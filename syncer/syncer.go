package syncer

import (
	"github.com/sirupsen/logrus"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/storage"
)

func New(name string, st storage.Interface, c config.Config, lc config.LMDB) *Syncer {
	return &Syncer{
		name:   name,
		st:     st,
		c:      c,
		lc:     lc,
		l:      logrus.WithField("db", name),
		shadow: true,
	}
}

type Syncer struct {
	name   string
	st     storage.Interface
	c      config.Config
	lc     config.LMDB
	l      logrus.FieldLogger
	shadow bool // use shadow database for timestamps?
}
