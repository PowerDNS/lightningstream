package strategy

import (
	"io"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
)

// Put implements the Put strategy.
//
// Directly put the entries from the iterator into LMDB, overwriting any existing one.
//
// Prerequisites:
//
// - Single namespace
// - Sorted input not required
//
// Uses: currently only for tests
func Put(txn *lmdb.Txn, dbi lmdb.DBI, it Iterator) error {
	for {
		// Get key
		key, err := it.Next()
		if err != nil {
			if err == io.EOF {
				return nil // done
			}
			return errors.Wrap(err, "next")
		}

		// Get val
		val, err := it.Merge(nil)
		if err != nil {
			return errors.Wrap(err, "merge")
		}

		// If the value is empty, we delete the key instead
		if len(val) == 0 {
			err := txn.Del(dbi, key, nil)
			if err != nil && !lmdb.IsNotFound(err) {
				return errors.Wrap(err, "del")
			}
			continue
		}

		// Put
		err = txn.Put(dbi, key, val, 0)
		if err != nil {
			return errors.Wrap(err, "put")
		}
	}
}
