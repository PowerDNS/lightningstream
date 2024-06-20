package syncer

import (
	"github.com/PowerDNS/lightningstream/config"
	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/c2h5oh/datasize"
	"github.com/sirupsen/logrus"
)

// OpenEnv opens the LMDB env with the right options
func OpenEnv(l logrus.FieldLogger, lc config.LMDB) (env *lmdb.Env, err error) {
	l.WithField("lmdbpath", lc.Path).Info("Opening LMDB")
	env, err = lmdbenv.NewWithOptions(lc.Path, lc.Options)
	if err != nil {
		return nil, err
	}

	// Print some env info
	info, err := env.Info()
	if err != nil {
		return nil, err
	}
	l.WithFields(logrus.Fields{
		"MapSize":   datasize.ByteSize(info.MapSize).HumanReadable(),
		"LastTxnID": info.LastTxnID,
	}).Info("Env info")

	// TODO: Perhaps check data if SchemaTracksChanges is set. Check if
	//       the timestamp is in a reasonable range or 0.

	return env, nil
}
