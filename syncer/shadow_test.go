package syncer

import (
	"context"
	"testing"
	"time"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/stretchr/testify/assert"
	"powerdns.com/platform/lightningstream/config"
	"powerdns.com/platform/lightningstream/lmdbenv"
	"powerdns.com/platform/lightningstream/lmdbenv/header"
	"powerdns.com/platform/lightningstream/snapshot"
)

func b(s string) []byte {
	return []byte(s)
}

func h(ts header.Timestamp, txnID header.TxnID, flags header.Flags) string {
	buf := make([]byte, header.MinHeaderSize)
	header.PutBasic(buf, ts, txnID, flags)
	return string(buf)
}

func testTS(i int) header.Timestamp {
	tsNano := header.TimestampFromTime(time.Date(2022, 2, i, 3, 4, 5, 123456789, time.UTC))
	return tsNano
}

func TestSyncer_shadow(t *testing.T) {
	v1 := []snapshot.KV{
		{Key: b("a"), Value: b("abc")},
		{Key: b("b"), Value: b("xyz")},
		{Key: b("c"), Value: b("cccccc")},
	}

	ts1 := testTS(1)
	ts2 := testTS(2)
	ts3 := testTS(3)

	s, err := New("test", nil, config.Config{}, config.LMDB{})
	assert.NoError(t, err)

	err = lmdbenv.TestEnv(func(env *lmdb.Env) error {
		// Initial data
		err := env.Update(func(txn *lmdb.Txn) error {
			// First insert the initial data into the main database
			dbi, err := txn.OpenDBI("foo", lmdb.Create)
			assert.NoError(t, err)
			for _, e := range v1 {
				err := txn.Put(dbi, e.Key, e.Value, 0)
				assert.NoError(t, err)
			}

			// Copy to shadow
			err = s.mainToShadow(context.Background(), txn, ts1)
			assert.NoError(t, err)

			// Read shadow DBI
			shadowDBI, err := txn.OpenDBI("_sync_shadow_foo", 0)
			assert.NoError(t, err)
			vals, err := lmdbenv.ReadDBIString(txn, shadowDBI)
			assert.NoError(t, err)

			// Verify contents
			assert.Equal(t, []lmdbenv.KVString{
				{Key: "a", Val: h(ts1, 1, 0) + "abc"},
				{Key: "b", Val: h(ts1, 1, 0) + "xyz"},
				{Key: "c", Val: h(ts1, 1, 0) + "cccccc"},
			}, vals)

			// Reverse sync should not change the original data
			err = s.shadowToMain(context.Background(), txn)
			assert.NoError(t, err)
			dbiMsg, err := s.readDBI(txn, "foo", "foo", true)
			assert.NoError(t, err)
			assert.Equal(t, v1, dbiMsg.Entries)

			return nil
		})
		assert.NoError(t, err)

		// Add and delete something in data and sync again
		err = env.Update(func(txn *lmdb.Txn) error {
			dbi, err := txn.OpenDBI("foo", 0)
			assert.NoError(t, err)
			// Add new 'd'
			err = txn.Put(dbi, b("d"), b("ddd"), 0)
			assert.NoError(t, err)
			// Remove 'b'
			err = txn.Del(dbi, b("b"), nil)
			assert.NoError(t, err)
			// Change 'c'
			err = txn.Put(dbi, b("c"), b("CCC"), 0)
			assert.NoError(t, err)

			// Copy to shadow
			err = s.mainToShadow(context.Background(), txn, ts2)
			assert.NoError(t, err)

			// Read shadow DBI
			shadowDBI, err := txn.OpenDBI("_sync_shadow_foo", 0)
			assert.NoError(t, err)
			vals, err := lmdbenv.ReadDBIString(txn, shadowDBI)
			assert.NoError(t, err)

			// Verify contents
			assert.Equal(t, []lmdbenv.KVString{
				{Key: "a", Val: h(ts1, 1, 0) + "abc"},          // timestamp unchanged
				{Key: "b", Val: h(ts2, 2, header.FlagDeleted)}, // deleted, empty value
				{Key: "c", Val: h(ts2, 2, 0) + "CCC"},          // changed
				{Key: "d", Val: h(ts2, 2, 0) + "ddd"},          // new
			}, vals)
			return nil
		})
		assert.NoError(t, err)

		// No changes in db, so no timestamp changes
		err = env.Update(func(txn *lmdb.Txn) error {
			// Copy to shadow
			err = s.mainToShadow(context.Background(), txn, ts3)
			assert.NoError(t, err)

			// Read shadow DBI
			shadowDBI, err := txn.OpenDBI("_sync_shadow_foo", 0)
			assert.NoError(t, err)
			vals, err := lmdbenv.ReadDBIString(txn, shadowDBI)
			assert.NoError(t, err)

			// Verify contents
			assert.Equal(t, []lmdbenv.KVString{
				{Key: "a", Val: h(ts1, 1, 0) + "abc"},          // timestamp unchanged
				{Key: "b", Val: h(ts2, 2, header.FlagDeleted)}, // deleted, empty value
				{Key: "c", Val: h(ts2, 2, 0) + "CCC"},          // changed
				{Key: "d", Val: h(ts2, 2, 0) + "ddd"},          // new
			}, vals)

			// Reverse sync should not change the original data
			dbi, err := txn.OpenDBI("foo", 0)
			assert.NoError(t, err)
			err = s.shadowToMain(context.Background(), txn)
			assert.NoError(t, err)
			data, err := lmdbenv.ReadDBIString(txn, dbi)
			assert.NoError(t, err)
			assert.Equal(t, []lmdbenv.KVString{
				{Key: "a", Val: "abc"},
				{Key: "c", Val: "CCC"},
				{Key: "d", Val: "ddd"},
			}, data)

			// If we delete some data, it will be restored if we repeat it
			err = txn.Del(dbi, b("a"), nil)
			assert.NoError(t, err)
			err = txn.Put(dbi, b("c"), b("CHANGED!"), 0)
			assert.NoError(t, err)
			err = txn.Put(dbi, b("z"), b("should not be here"), 0)
			assert.NoError(t, err)
			data, err = lmdbenv.ReadDBIString(txn, dbi)
			assert.NoError(t, err)
			assert.Equal(t, []lmdbenv.KVString{
				{Key: "c", Val: "CHANGED!"},
				{Key: "d", Val: "ddd"},
				{Key: "z", Val: "should not be here"},
			}, data)
			// Sync again
			err = s.shadowToMain(context.Background(), txn)
			assert.NoError(t, err)
			data, err = lmdbenv.ReadDBIString(txn, dbi)
			assert.NoError(t, err)
			assert.Equal(t, []lmdbenv.KVString{
				{Key: "a", Val: "abc"},
				{Key: "c", Val: "CCC"},
				{Key: "d", Val: "ddd"},
			}, data)

			return nil
		})
		assert.NoError(t, err)

		return nil
	})
	assert.NoError(t, err)

}
