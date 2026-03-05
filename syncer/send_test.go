package syncer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/PowerDNS/lightningstream/config"
	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lightningstream/snapshot"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/simpleblob/backends/memory"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSyncer_SendOnce_GenerationIDStable verifies that the GenerationID embedded
// in snapshot filenames does not change across multiple SendOnce calls on the
// same Syncer instance. It must stay stable for the lifetime of the process so
// that peers can identify which syncer produced each snapshot.
func TestSyncer_SendOnce_GenerationIDStable(t *testing.T) {
	l, _ := test.NewNullLogger()
	lc := config.LMDB{SchemaTracksChanges: true}
	now := time.Now()

	val := make([]byte, header.MinHeaderSize, 50)
	header.PutBasic(val, header.TimestampFromTime(now), 42, header.NoFlags)
	val = append(val, "test-value"...)

	st := memory.New()

	err := lmdbenv.TestEnv(func(env *lmdb.Env) error {
		s, err := New("test", env, st, config.Config{StorageRetryCount: 1}, lc, Options{})
		require.NoError(t, err)
		s.l = l

		ctx := context.Background()

		// Write distinct data between sends so each SendOnce produces a new snapshot.
		for i := 0; i < 3; i++ {
			err = env.Update(func(txn *lmdb.Txn) error {
				dbi, err := txn.OpenDBI("foo", lmdb.Create)
				require.NoError(t, err)
				return txn.Put(dbi, []byte(fmt.Sprintf("key-%d", i)), val, 0)
			})
			require.NoError(t, err)

			_, err = s.SendOnce(ctx, env)
			require.NoError(t, err)
		}
		return nil
	})
	require.NoError(t, err)

	blobs, err := st.List(context.Background(), "")
	require.NoError(t, err)

	var generationIDs []string
	for _, name := range blobs.Names() {
		ni, err := snapshot.ParseName(name)
		if err != nil {
			continue // skip non-snapshot files
		}
		generationIDs = append(generationIDs, string(ni.GenerationID))
	}

	require.NotEmpty(t, generationIDs, "expected at least one snapshot to be stored")

	first := generationIDs[0]
	for i, gid := range generationIDs[1:] {
		assert.Equal(t, first, gid, "snapshot %d has a different GenerationID", i+1)
	}
}

func BenchmarkSyncer_SendOnce_native_100k(b *testing.B) {
	doBenchmarkSyncerSendOnce(b, true, false)
}

func BenchmarkSyncer_SendOnce_shadow_100k(b *testing.B) {
	doBenchmarkSyncerSendOnce(b, false, false)
}

func BenchmarkSyncer_SendOnce_shadow_dupsort_100k(b *testing.B) {
	doBenchmarkSyncerSendOnce(b, false, true)
}

func doBenchmarkSyncerSendOnce(b *testing.B, native, dupsort bool) {
	t := b
	const nRecords = 100_000
	now := time.Now()

	var extraDBIFlags uint
	if dupsort {
		extraDBIFlags = lmdb.DupSort
	}

	l, hook := test.NewNullLogger()
	_ = hook
	lc := config.LMDB{
		SchemaTracksChanges: native,
		DupSortHack:         dupsort,
	}

	// Fixed value
	// We add a header, but we can also benchmark this as all app value
	val := make([]byte, header.MinHeaderSize, 50)
	header.PutBasic(val, header.TimestampFromTime(now), 42, header.NoFlags)
	val = append(val, "TESTING-123456789"...)

	err := lmdbenv.TestEnv(func(env *lmdb.Env) error {
		info, err := env.Info()
		require.NoError(t, err)
		t.Logf("env info: %+v", info)

		syncer, err := New("test", env, memory.New(), config.Config{}, lc, Options{})
		require.NoError(t, err)
		syncer.l = l

		// Fill some data to dump
		err = env.Update(func(txn *lmdb.Txn) error {
			// First insert the initial data into the main database
			dbi, err := txn.OpenDBI("foo", lmdb.Create|extraDBIFlags)
			require.NoError(t, err)
			for i := 0; i < nRecords; i++ {
				key := []byte(fmt.Sprintf("key-%020d", i))
				err := txn.Put(dbi, key, val, 0)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Actual benchmark
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := syncer.SendOnce(ctx, env)
			require.NoError(b, err)
			syncer.st = memory.New()
		}

		return err
	})
	require.NoError(b, err)
}
