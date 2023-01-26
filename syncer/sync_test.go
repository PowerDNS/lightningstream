package syncer

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/simpleblob"
	"github.com/PowerDNS/simpleblob/backends/memory"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/config/logger"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/header"
	"powerdns.com/platform/lightningstream/snapshot"
)

const testLMDBName = "default"
const testDBIName = "test"
const tick = 100 * time.Millisecond

func TestSyncer_Sync_startup(t *testing.T) {
	t.Run("with-timestamped-schema", func(t *testing.T) {
		doTest(t, true)
	})
	t.Run("with-shadow", func(t *testing.T) {
		doTest(t, false)
	})
}

func doTest(t *testing.T, withHeader bool) {
	// This test reproduces a bug where an instance ends up with an old version
	// after a restart when not using a native timestamped schema.

	st := memory.New()

	// This test starts two Syncer instances, "a" and "b".
	syncerA, envA := createInstance(t, "a", st, withHeader)
	syncerB, envB := createInstance(t, "b", st, withHeader)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctxA, cancelA := context.WithCancel(ctx)
	ctxB, cancelB := context.WithCancel(ctx)

	setKey(t, envA, "foo", "v1", withHeader)

	// Start syncer A with one key
	t.Log("Starting syncer A")
	go runSync(ctxA, syncerA)

	t.Log("----------")
	time.Sleep(2 * tick)
	t.Log("----------")

	// Expecting one snapshot on startup, because not empty
	logSnapshotList(t, st)
	entries := listInstanceSnapshots(st, "a")
	assert.Len(t, entries, 1, "A")

	// Start syncer B with an empty LMDB
	t.Log("Starting syncer B")
	go runSync(ctxB, syncerB)

	t.Log("----------")
	time.Sleep(2 * tick)
	t.Log("----------")

	// No snapshot, because empty
	logSnapshotList(t, st)
	entries = listInstanceSnapshots(st, "b")
	assert.Len(t, entries, 0, "B")

	assertKey(t, envB, "foo", "v1", withHeader)

	// Now set something in B
	setKey(t, envB, "foo", "v2", withHeader)

	t.Log("----------")
	time.Sleep(3 * tick)
	t.Log("----------")

	// New snapshot in B, no new one in A
	logSnapshotList(t, st)
	entries = listInstanceSnapshots(st, "b")
	assert.Len(t, entries, 1, "B")
	entries = listInstanceSnapshots(st, "a")
	assert.Len(t, entries, 1, "A")

	// New value should be present in A
	assertKey(t, envB, "foo", "v2", withHeader)

	// Restart syncer for A
	t.Log("Restarting syncer A")
	cancelA()
	ctxA, cancelA = context.WithCancel(ctx)
	t.Log("----------")
	go runSync(ctxA, syncerA)

	t.Log("----------")
	time.Sleep(3 * tick)
	t.Log("----------")

	// Check is the contents of A are still correct after restart
	assertKey(t, envA, "foo", "v2", withHeader)
	entries = listInstanceSnapshots(st, "a")
	// None should have been created on startup, because there already exist
	// snapshots and no new data was added that could implicitly trigger one.
	assert.Len(t, entries, 1, "A")

	cancelA()
	cancelB()
	t.Log("Done")
}

func createInstance(t *testing.T, name string, st simpleblob.Interface, timestamped bool) (*Syncer, *lmdb.Env) {
	env, tmp, err := createLMDB(t)
	assert.NoError(t, err)

	c := createConfig(name, tmp, timestamped)
	syncer, err := New("default", st, c, c.LMDBs[testLMDBName])
	assert.NoError(t, err)

	return syncer, env
}

func logSnapshotList(t *testing.T, st simpleblob.Interface) {
	ctx := context.Background()
	entries, _ := st.List(ctx, "")
	var lines []string
	for _, e := range entries {
		name := e.Name
		data, _ := st.Load(ctx, name)
		msg, _ := snapshot.LoadData(data)
		for _, se := range msg.Databases[0].Entries {
			lines = append(lines,
				fmt.Sprintf("%s : %s = %q @ %s",
					name, se.Key, se.Value,
					snapshot.NameTimestampFromNano(header.Timestamp(se.TimestampNano))),
			)
		}
	}
	t.Logf("Snapshots in storage:\n%s", strings.Join(lines, "\n"))
}

func listInstanceSnapshots(st simpleblob.Interface, instance string) simpleblob.BlobList {
	prefix := testLMDBName + "__" + instance + "__"
	entries, _ := st.List(context.Background(), prefix)
	return entries
}

func runSync(ctx context.Context, syncer *Syncer) {
	err := syncer.Sync(ctx)
	if err != nil && err != context.Canceled {
		logrus.WithError(err).WithField("syncer", syncer.name).Error("Syncer Sync error")
	}
}

func assertKey(t *testing.T, env *lmdb.Env, key, val string, withHeader bool) {
	kv, err := dumpData(env, withHeader)
	assert.NoError(t, err)
	assert.Equal(t, val, kv[key])
}

func setKey(t *testing.T, env *lmdb.Env, key, val string, withHeader bool) {
	err := env.Update(func(txn *lmdb.Txn) error {
		dbi, err := txn.OpenDBI(testDBIName, lmdb.Create)
		if err != nil {
			return err
		}
		valb := []byte(val)
		if withHeader {
			var b [header.MinHeaderSize]byte
			header.PutBasic(b[:],
				header.TimestampFromTime(time.Now()),
				header.TxnID(txn.ID()),
				header.NoFlags)
			binary.BigEndian.PutUint64(b[:], uint64(time.Now().UnixNano()))
			valb = append(b[:], valb...)
		}
		err = txn.Put(dbi, []byte(key), valb, 0)
		return err
	})
	assert.NoError(t, err)
}

func dumpData(env *lmdb.Env, withHeader bool) (map[string]string, error) {
	data := make(map[string]string)
	err := env.View(func(txn *lmdb.Txn) error {
		dbi, err := txn.OpenDBI(testDBIName, lmdb.Create)
		if err != nil {
			return err
		}
		kvs, err := lmdbenv.ReadDBIString(txn, dbi)
		if err != nil {
			return err
		}
		for _, kv := range kvs {
			if withHeader {
				val, err := header.Skip([]byte(kv.Val))
				if err != nil {
					return err
				}
				data[kv.Key] = string(val)
			} else {
				data[kv.Key] = kv.Val
			}
		}
		return err
	})
	return data, err
}

func createConfig(instance string, tmpdir string, withHeader bool) config.Config {
	c := config.Config{
		Instance:             instance,
		LMDBs:                make(map[string]config.LMDB),
		Storage:              config.Storage{}, // directly provided
		HTTP:                 config.HTTP{},
		Log:                  logger.Config{},
		LMDBPollInterval:     tick,
		LMDBLogStatsInterval: 0,
		StoragePollInterval:  tick,
		StorageRetryInterval: tick,
		StorageRetryCount:    1,
		LMDBScrapeSmaps:      false,
		Version:              "",
	}
	c.LMDBs[testLMDBName] = config.LMDB{
		Path:                tmpdir,
		Options:             lmdbenv.Options{},
		DBIOptions:          nil,
		SchemaTracksChanges: withHeader,
		DupSortHack:         false,
		ScrapeSmaps:         false,
		LogStats:            false,
		LogStatsInterval:    0,
	}
	return c
}

func createLMDB(t *testing.T) (env *lmdb.Env, tmpdir string, err error) {
	tmpdir, err = os.MkdirTemp("", "lmdbtest_")
	if err != nil {
		return nil, "", err
	}
	t.Cleanup(func() {
		if tmpdir == "" {
			panic("Empty tmpdir")
		}
		_ = os.RemoveAll(tmpdir)
	})

	env, err = lmdbenv.New(tmpdir, 0)
	if err != nil {
		return nil, "", err
	}

	return env, tmpdir, nil
}
