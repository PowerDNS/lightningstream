package lmdbenv

import (
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
)

// IsEmpty checks if a database is empty
func IsEmpty(txn *lmdb.Txn, dbi lmdb.DBI) (bool, error) {
	c, err := txn.OpenCursor(dbi)
	if err != nil {
		return false, errors.Wrap(err, "open cursor")
	}
	defer c.Close()

	_, _, err = c.Get(nil, nil, lmdb.First)
	if err == nil {
		return false, nil
	}
	if !lmdb.IsNotFound(err) {
		return false, errors.Wrap(err, "get")
	}
	return true, nil
}
