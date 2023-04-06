package syncer

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/simpleblob"
	"github.com/PowerDNS/simpleblob/backends/memory"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/config/logger"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/header"
	"powerdns.com/platform/lightningstream/snapshot"
)

const testLMDBName = "default"
const testDBIName = "test"
const tick = 10 * time.Millisecond

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

	// For some reason trying to close the envs in this test segfaults on Linux (not on macOS).
	// It appears like this is caused by the syncer still running after cancellation
	// (and thus after the env is closed), but I did not get to the bottom of this yet.
	// [signal SIGSEGV: segmentation violation code=0x1 addr=0x7fd015132090 pc=0x8dc942]
	//defer envA.Close()
	//defer envB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctxA, cancelA := context.WithCancel(ctx)
	ctxB, cancelB := context.WithCancel(ctx)

	setKey(t, envA, "foo", "v1", withHeader)

	// Start syncer A with one key
	t.Log("Starting syncer A")
	goRunSync(ctxA, syncerA)

	t.Log("----------")

	// Expecting one snapshot on startup, because not empty
	requireSnapshotsLenWait(t, st, 1, "A")

	// Start syncer B with an empty LMDB
	// Starting with an empty LMDB is a special case that will not trigger any
	// local snapshot.
	t.Log("Starting syncer B")
	goRunSync(ctxB, syncerB)

	t.Log("----------")

	// Wait until the data from A was synced to B
	assertKeyWait(t, envB, "foo", "v1", withHeader)

	// No snapshot made by B, because we started empty
	requireSnapshotsLenWait(t, st, 0, "B")

	// Now set something in B
	setKey(t, envB, "foo", "v2", withHeader)

	t.Log("----------")

	// New snapshot in B, no new one in A
	requireSnapshotsLenWait(t, st, 1, "B")
	requireSnapshotsLenWait(t, st, 1, "A")

	// New value should be present in A
	assertKeyWait(t, envB, "foo", "v2", withHeader)

	// Restart syncer for A
	t.Log("Restarting syncer A")
	cancelA()
	ctxA, cancelA = context.WithCancel(ctx)
	t.Log("----------")
	goRunSync(ctxA, syncerA)

	t.Log("----------")

	// Check is the contents of A are still correct after restart
	assertKeyWait(t, envA, "foo", "v2", withHeader)
	// A new snapshot should always be created on startup, in case the LMDB
	// was modified while it was down.
	requireSnapshotsLenWait(t, st, 2, "A")

	// Stopping syncer for A
	t.Log("Stopping syncer A")
	cancelA()

	// Now set something in A while its syncer is down
	t.Log("Modifying data in A while the syncer is down")
	setKey(t, envA, "new", "hello", withHeader)

	t.Log("----------")
	t.Log("Starting syncer A again")
	ctxA, cancelA = context.WithCancel(ctx)
	goRunSync(ctxA, syncerA)
	t.Log("----------")

	// New value in A should get synced to B
	assertKeyWait(t, envB, "new", "hello", withHeader)
	// Check if the contents of A are still correct after restart
	assertKeyWait(t, envA, "new", "hello", withHeader)
	requireSnapshotsLenWait(t, st, 3, "A")

	cancelA()
	cancelB()
	t.Log("Done")
}

func createInstance(t *testing.T, name string, st simpleblob.Interface, timestamped bool) (*Syncer, *lmdb.Env) {
	env, tmp, err := createLMDB(t)
	require.NoError(t, err)

	c := createConfig(name, tmp, timestamped)
	syncer, err := New("default", env, st, c, c.LMDBs[testLMDBName], Options{})
	require.NoError(t, err)

	return syncer, env
}

func LogSnapshotList(t *testing.T, st simpleblob.Interface) {
	ctx := context.Background()
	entries, _ := st.List(ctx, "")
	var lines []string
	for _, e := range entries {
		name := e.Name
		data, _ := st.Load(ctx, name)
		msg, _ := snapshot.LoadData(data)
		dbiMsg := msg.Databases[0]
		dbiMsg.ResetCursor()
		for {
			se, err := dbiMsg.Next()
			if err != nil {
				if err != io.EOF {
					t.Errorf("Unexpected error from dbimsg.Next(): %v", err)
				}
				break
			}
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

func requireSnapshotsLenWait(t *testing.T, st simpleblob.Interface, expLen int, instance string) {
	var list simpleblob.BlobList
	// Retry until it succeeds
	var i int
	const maxIter = 150
	const sleepTime = 10 * time.Millisecond
	defer func() {
		t.Logf("Waited %d/%d iterations for the expected snapshot length", i, maxIter)
	}()
	for i = 0; i < maxIter; i++ {
		list = listInstanceSnapshots(st, strings.ToLower(instance))
		l := len(list)
		if l == expLen {
			return
		}
		time.Sleep(sleepTime)
	}
	// This one is actually expected to fail, call it for the formatting
	t.Logf("Gave up on waiting for the expected snapshot length")
	require.Len(t, list, expLen, instance)
}

// Ensure that we there are never two Sync goroutines running at the same time,
// because this can cause a data race.
// Protected by mutex, just in case the test is ever run in parallel mode.
var (
	runningSyncersMu sync.Mutex
	runningSyncers   = map[*Syncer]*sync.WaitGroup{}
)

func goRunSync(ctx context.Context, syncer *Syncer) {
	runningSyncersMu.Lock()
	wg, exists := runningSyncers[syncer]
	if !exists {
		wg = &sync.WaitGroup{}
		runningSyncers[syncer] = wg
	}
	runningSyncersMu.Unlock()
	go func() {
		logrus.Info("Wait for any previous Syncer instance to exit")
		wg.Wait()
		logrus.Info("Wait for any previous Syncer done")
		wg.Add(1)
		defer wg.Done()
		runSync(ctx, syncer)
	}()
}

func runSync(ctx context.Context, syncer *Syncer) {
	err := syncer.Sync(ctx)
	if err != nil && err != context.Canceled {
		logrus.WithError(err).WithField("syncer", syncer.name).Error("Syncer Sync error")
	}
}

func assertKeyWait(t *testing.T, env *lmdb.Env, key, val string, withHeader bool) {
	var kv map[string]string
	var err error
	var i int
	const maxIter = 150
	const sleepTime = 10 * time.Millisecond
	defer func() {
		t.Logf("Waited %d/%d iterations for the expected key", i, maxIter)
	}()
	for i = 0; i < maxIter; i++ {
		kv, err = dumpData(env, withHeader)
		if err != nil && !lmdb.IsNotFound(err) {
			require.NoError(t, err)
		}
		if kv[key] == val {
			return
		}
		time.Sleep(sleepTime)
	}
	// Expected to fail now, called for formatting
	t.Logf("Gave up on waiting for the expected key")
	require.Equal(t, val, kv[key])
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
	require.NoError(t, err)
}

func dumpData(env *lmdb.Env, withHeader bool) (map[string]string, error) {
	data := make(map[string]string)
	err := env.View(func(txn *lmdb.Txn) error {
		dbi, err := txn.OpenDBI(testDBIName, 0)
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
