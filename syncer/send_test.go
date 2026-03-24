package syncer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/PowerDNS/lightningstream/config"
	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/simpleblob/backends/memory"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

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
