package syncer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/PowerDNS/simpleblob/backends/memory"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/header"
)

func BenchmarkSyncer_SendOnce_100k(b *testing.B) {
	t := b
	const nRecords = 100_000
	now := time.Now()

	syncer, err := New("test", memory.New(), config.Config{}, config.LMDB{
		SchemaTracksChanges: true,
	})
	require.NoError(t, err)

	l, hook := test.NewNullLogger()
	_ = hook
	syncer.l = l

	// Fixed value
	// We add a header, but we can also benchmark this as all app value
	val := make([]byte, header.MinHeaderSize, 50)
	header.PutBasic(val, header.TimestampFromTime(now), 42, header.NoFlags)
	val = append(val, "TESTING-123456789"...)

	err = lmdbenv.TestEnv(func(env *lmdb.Env) error {
		info, err := env.Info()
		require.NoError(t, err)
		t.Logf("env info: %+v", info)

		// Fill some data to dump
		err = env.Update(func(txn *lmdb.Txn) error {
			// First insert the initial data into the main database
			dbi, err := txn.OpenDBI("foo", lmdb.Create)
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
