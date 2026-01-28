package sweeper

import (
	"fmt"
	"testing"
	"time"

	"github.com/PowerDNS/lightningstream/config"
	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lightningstream/lmdbenv/header"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestSweeper(t *testing.T) {
	conf := config.Sweeper{
		Enabled:         true,
		RetentionDays:   2,
		Interval:        time.Second,
		FirstInterval:   0,
		LockDuration:    time.Second,
		ReleaseDuration: 0,
	}

	l, _ := test.NewNullLogger()

	err := lmdbenv.TestEnv(func(env *lmdb.Env) error {
		sweeper := New("test", conf, env, l, true)

		t.Run("empty-lmdb", func(t *testing.T) {
			// Completely empty database sweep
			assert.NoError(t, sweeper.sweep(t.Context()))
		})

		createDBI := func(name string) lmdb.DBI {
			var dbi lmdb.DBI
			err := env.Update(func(txn *lmdb.Txn) error {
				var err error
				dbi, err = txn.CreateDBI("test1")
				return err
			})
			assert.NoError(t, err)
			return dbi
		}

		t.Run("empty-dbi", func(t *testing.T) {
			// Empty DBI sweep
			_ = createDBI("empty")
			assert.NoError(t, sweeper.sweep(t.Context()))
		})

		t.Run("mix", func(t *testing.T) {
			now := time.Now()
			past := now.Add(-50 * time.Hour) // longer than RetentionDays=2
			nowTS := header.TimestampFromTime(now)
			pastTS := header.TimestampFromTime(past)

			mix := createDBI("mix")

			assert.NoError(t, env.Update(func(txn *lmdb.Txn) error {
				for i := 0; i < 3000; i++ {
					key := []byte(fmt.Sprintf("key-%08d", i))
					val := make([]byte, header.MinHeaderSize)
					switch i % 3 {
					case 0:
						// Normal entry, not deleted
						header.PutBasic(val, pastTS, 1, 0)
					case 1:
						// Recently deleted entry
						header.PutBasic(val, nowTS, 1, header.FlagDeleted)
					case 2:
						// Deleted longer time ago
						header.PutBasic(val, pastTS, 1, header.FlagDeleted)
					}
					assert.NoError(t, txn.Put(mix, key, val, 0))
				}
				return nil
			}))

			assert.NoError(t, sweeper.sweep(t.Context()))
			assert.Equal(t, 2000, sweeper.lastStats.nEntries)
			assert.Equal(t, 1000, sweeper.lastStats.nDeleted)
			assert.Equal(t, 1000, sweeper.lastStats.nCleaned)
			assert.Equal(t, 0.5, sweeper.lastStats.deletedFraction())
			t.Logf("Cleaning 3000 entries took %s", sweeper.lastStats.timeTaken)
		})

		return nil
	})
	assert.NoError(t, err)
}

func BenchmarkSweeper(b *testing.B) {
	// This benchmark creates b.N entries and sweeps them.
	// Here 1/3 of the entries will be cleaned.

	// TODO: This may create a large LMDB and fail at a certain N due to mapsize.
	//       Find a better way?

	conf := config.Sweeper{
		Enabled:         true,
		RetentionDays:   2,
		Interval:        time.Second,
		FirstInterval:   0,
		LockDuration:    10 * time.Millisecond, // Forces split operation
		ReleaseDuration: 0,
	}

	l, _ := test.NewNullLogger()

	err := lmdbenv.TestEnv(func(env *lmdb.Env) error {
		sweeper := New("test", conf, env, l, true)

		createDBI := func(name string) lmdb.DBI {
			var dbi lmdb.DBI
			err := env.Update(func(txn *lmdb.Txn) error {
				var err error
				dbi, err = txn.CreateDBI(name)
				return err
			})
			assert.NoError(b, err)
			return dbi
		}

		now := time.Now()
		past := now.Add(-3 * 24 * time.Hour) // longer than RetentionDays=2
		nowTS := header.TimestampFromTime(now)
		pastTS := header.TimestampFromTime(past)

		mix := createDBI("bench")

		assert.NoError(b, env.Update(func(txn *lmdb.Txn) error {
			for i := 0; i < b.N; i++ {
				key := []byte(fmt.Sprintf("key-%08d", i))
				val := make([]byte, header.MinHeaderSize)
				switch i % 3 {
				case 0:
					// Normal entry, not deleted
					header.PutBasic(val, pastTS, 1, 0)
				case 1:
					// Recently deleted entry
					header.PutBasic(val, nowTS, 1, header.FlagDeleted)
				case 2:
					// Deleted longer time ago
					header.PutBasic(val, pastTS, 1, header.FlagDeleted)
				}
				assert.NoError(b, txn.Put(mix, key, val, 0))
			}
			return nil
		}))

		// Actual benchmark
		b.ReportAllocs()
		b.ResetTimer()
		err := sweeper.sweep(b.Context())
		b.StopTimer()

		if b.N > 2 {
			assert.NoError(b, env.Update(func(txn *lmdb.Txn) error {
				stat, err := txn.Stat(mix)
				if err != nil {
					return err
				}
				assert.Greater(b, b.N-int(stat.Entries), b.N/4, "less than a 1/4 entries were cleaned")
				return nil
			}))
		}

		// This is the most interesting metric
		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "entries/s")

		return err
	})
	assert.NoError(b, err)
}
