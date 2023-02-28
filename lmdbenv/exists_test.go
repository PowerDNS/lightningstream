package lmdbenv

import (
	"testing"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/stretchr/testify/assert"
)

func TestDBIExists(t *testing.T) {
	err := TestEnv(func(env *lmdb.Env) error {
		err := env.Update(func(txn *lmdb.Txn) error {
			exists, err := DBIExists(txn, "does-not")
			if err != nil {
				return err
			}
			assert.False(t, exists)

			_, err = txn.CreateDBI("foo")
			if err != nil {
				return err
			}

			exists, err = DBIExists(txn, "foo")
			if err != nil {
				return err
			}
			assert.True(t, exists)

			return nil
		})
		if err != nil {
			return err
		}
		return nil
	})
	assert.NoError(t, err)
}
