package strategy

import (
	"bytes"
	"io"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
)

// Append implements the Append-strategy.
//
// Use the MDB_APPEND flag to directly append items to the database. This is the fastest
// way to insert items and is great for the very first snapshot load.
//
// Prerequisites:
//
// - Database is empty
// - Sorted input
//
// Uses: make the very first snapshot load fast.
//
func Append(txn *lmdb.Txn, dbi lmdb.DBI, it Iterator) error {
	prevKey := make([]byte, 0, LMDBMaxKeySize)
	for {
		// Get key
		key, err := it.Next()
		if err != nil {
			if err == io.EOF {
				return nil // done
			}
			return errors.Wrap(err, "next")
		}

		// Check to ensure the keys are in insert order
		if bytes.Compare(prevKey, key) >= 0 {
			return ErrNotSorted
		}
		prevKey = prevKey[:len(key)]
		copy(prevKey, key)

		// Get val
		val, err := it.Merge(nil)
		if err != nil {
			return errors.Wrap(err, "merge")
		}
		if len(val) == 0 {
			continue
		}

		// Append
		err = txn.Put(dbi, key, val, lmdb.Append)
		if err != nil {
			return errors.Wrap(err, "append")
		}
	}
}
