package limitscanner

import (
	"fmt"
	"testing"
	"time"

	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimitScanner(t *testing.T) {
	err := lmdbenv.TestEnv(func(env *lmdb.Env) error {
		var dbi lmdb.DBI

		// Init
		err := env.Update(func(txn *lmdb.Txn) error {
			var err error
			dbi, err = txn.OpenDBI("test", lmdb.Create)
			require.NoError(t, err)

			for i := 1; i <= 250; i++ {
				key := fmt.Sprintf("key-%05d", i)
				val := fmt.Sprintf("val-%05d", i)
				err := txn.Put(dbi, []byte(key), []byte(val), 0)
				require.NoError(t, err)
			}
			return nil
		})
		require.NoError(t, err)

		var last LimitCursor
		t.Run("limited-scan", func(t *testing.T) {
			err = env.View(func(txn *lmdb.Txn) error {
				ls, err := NewLimitScanner(Options{
					Txn:          txn,
					DBI:          dbi,
					LimitRecords: 100,
				})
				assert.NoError(t, err)
				defer ls.Close()

				count := 0
				for ls.Scan() {
					count++
				}
				assert.Equal(t, 100, count)

				last = ls.Last()
				assert.Equal(t, "key-00100", string(last.key))
				assert.Equal(t, "val-00100", string(last.val))

				return ls.Err()
			})
			require.NoError(t, err)
			require.False(t, last.IsZero(), "expected a limited scan")
		})

		t.Run("limited-scan-continued", func(t *testing.T) {
			// Limited scan - continue
			err = env.View(func(txn *lmdb.Txn) error {
				ls, err := NewLimitScanner(Options{
					Txn:          txn,
					DBI:          dbi,
					LimitRecords: 100,
					Last:         last, // Note this added cursor
				})
				assert.NoError(t, err)
				defer ls.Close()

				count := 0
				for ls.Scan() {
					count++
				}
				assert.Equal(t, 100, count)

				last = ls.Last()
				assert.Equal(t, "key-00200", string(last.key))
				assert.Equal(t, "val-00200", string(last.val))

				return ls.Err()
			})
			require.False(t, last.IsZero(), "expected a limited scan")
		})

		t.Run("limited-scan-continued-deleted", func(t *testing.T) {
			// Limited scan - continue when the last entry has been deleted
			err = env.Update(func(txn *lmdb.Txn) error {
				return txn.Del(dbi, last.key, last.val)
			})
			require.NoError(t, err)
			err = env.View(func(txn *lmdb.Txn) error {
				ls, err := NewLimitScanner(Options{
					Txn:          txn,
					DBI:          dbi,
					LimitRecords: 10,
					Last:         last,
				})
				assert.NoError(t, err)
				defer ls.Close()

				count := 0
				for ls.Scan() {
					count++
				}
				assert.Equal(t, 10, count)

				last = ls.Last()
				assert.Equal(t, "key-00210", string(last.key))
				assert.Equal(t, "val-00210", string(last.val))

				return ls.Err()
			})
			require.NoError(t, err)
			require.False(t, last.IsZero(), "expected a limited scan")
		})

		t.Run("limited-scan-final", func(t *testing.T) {
			// Limited scan - final chunk of 40
			err = env.View(func(txn *lmdb.Txn) error {
				ls, err := NewLimitScanner(Options{
					Txn:          txn,
					DBI:          dbi,
					LimitRecords: 100,
					Last:         last,
				})
				assert.NoError(t, err)
				defer ls.Close()

				count := 0
				for ls.Scan() {
					count++
				}
				assert.Equal(t, 40, count)

				last = ls.Last()
				assert.Nil(t, last.key)
				assert.Nil(t, last.val)

				return ls.Err()
			})
			require.NoError(t, err)
			require.True(t, last.IsZero(), "unexpected limited scan")
		})

		t.Run("limited-by-time", func(t *testing.T) {
			// Rescan limited by time
			err = env.View(func(txn *lmdb.Txn) error {
				ls, err := NewLimitScanner(Options{
					Txn:                     txn,
					DBI:                     dbi,
					LimitDuration:           time.Nanosecond, // very short
					LimitDurationCheckEvery: 50,              // check time every 50 records
				})
				assert.NoError(t, err)
				defer ls.Close()

				count := 0
				for ls.Scan() {
					count++
				}
				// Because we check the time every 50 records, we will fetch 50
				// before we realize we passed the short deadline.
				assert.Equal(t, 50, count)

				last = ls.Last()
				assert.Equal(t, "key-00050", string(last.key))
				assert.Equal(t, "val-00050", string(last.val))

				return ls.Err()
			})
			require.NoError(t, err)
			require.False(t, last.IsZero(), "expected a limited scan")
		})

		t.Run("limited-by-plenty-of-time", func(t *testing.T) {
			// Rescan limited by time, but with plenty of time
			err = env.View(func(txn *lmdb.Txn) error {
				ls, err := NewLimitScanner(Options{
					Txn:                     txn,
					DBI:                     dbi,
					LimitDuration:           time.Second, // an eternity
					LimitDurationCheckEvery: 50,          // check time every 50 records
				})
				assert.NoError(t, err)
				defer ls.Close()

				count := 0
				for ls.Scan() {
					count++
				}
				// Now we will manage to scan all records before the deadline
				// (note that we deleted one before)
				assert.Equal(t, 249, count)

				last = ls.Last()
				assert.Nil(t, last.key)
				assert.Nil(t, last.val)

				return ls.Err()
			})
			require.NoError(t, err)
			require.True(t, last.IsZero(), "unexpected limited scan")
		})

		return nil
	})
	require.NoError(t, err)
}
