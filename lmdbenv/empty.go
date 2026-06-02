package lmdbenv

import (
	"fmt"

	"github.com/PowerDNS/lmdb-go/lmdb"
)

// IsEmpty checks if a database is empty
func IsEmpty(txn *lmdb.Txn, dbi lmdb.DBI) (bool, error) {
	c, err := txn.OpenCursor(dbi)
	if err != nil {
		return false, fmt.Errorf("open cursor: %w", err)
	}
	defer c.Close()

	_, _, err = c.Get(nil, nil, lmdb.First)
	if err == nil {
		return false, nil
	}
	if !lmdb.IsNotFound(err) {
		return false, fmt.Errorf("get: %w", err)
	}
	return true, nil
}
