package status

import (
	"context"
	"sync"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/simpleblob"
	"github.com/c2h5oh/datasize"
	"github.com/pkg/errors"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/stats"
)

type info struct {
	mu  sync.Mutex
	dbs []dbs
	st  simpleblob.Interface
}

type dbs struct {
	name string
	env  *lmdb.Env
}

type DBInfo struct {
	Name     string
	Info     *lmdb.EnvInfo
	DBIStats []DBIStat
	Used     datasize.ByteSize
	Err      error
}

type DBIStat struct {
	Name         string
	Stat         *lmdb.Stat
	Used         datasize.ByteSize
	Flags        uint
	FlagsDisplay string
}

var gi info

func (i *info) ListBlobs(ctx context.Context) (simpleblob.BlobList, error) {
	i.mu.Lock()
	st := i.st
	i.mu.Unlock()
	if st == nil {
		return nil, errors.New("no storage registered with status page")
	}
	list, err := st.List(ctx, "")
	return list, err
}

func (i *info) DBInfo() (res []DBInfo) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for _, db := range i.dbs {
		var info DBInfo
		info.Name = db.name
		var err error
		info.Info, err = db.env.Info()
		if err != nil {
			info.Err = err
			continue
		}
		info.Err = db.env.View(func(txn *lmdb.Txn) error {
			dbiNames, err := lmdbenv.ReadDBINames(txn)
			if err != nil {
				return err
			}
			for _, dbiName := range dbiNames {
				dbi, err := txn.OpenDBI(dbiName, 0)
				if err != nil {
					return err
				}
				fl, err := txn.Flags(dbi)
				if err != nil {
					return err
				}
				st, err := txn.Stat(dbi)
				if err != nil {
					return err
				}
				ds := DBIStat{
					Name:         dbiName,
					Stat:         st,
					Used:         datasize.ByteSize(stats.PageUsageBytes(st)),
					Flags:        fl,
					FlagsDisplay: displayFlags(fl),
				}
				info.DBIStats = append(info.DBIStats, ds)
				info.Used += ds.Used
			}
			return nil
		})
		res = append(res, info)
	}
	return res
}

// AddLMDBEnv registers an LMDB Env with the status page
func AddLMDBEnv(name string, env *lmdb.Env) {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	gi.dbs = append(gi.dbs, dbs{
		name: name,
		env:  env,
	})
}

func RemoveLMDBEnv(name string) {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	var dbs []dbs
	for _, db := range gi.dbs {
		if db.name == name {
			continue
		}
		dbs = append(dbs, db)
	}
	gi.dbs = dbs
}

func SetStorage(st simpleblob.Interface) {
	gi.mu.Lock()
	defer gi.mu.Unlock()
	gi.st = st
}
