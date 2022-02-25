package strategy

import (
	"io"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
)

// Update implements the Update strategy.
//
// - Sorted input not required
func Update(txn *lmdb.Txn, dbi lmdb.DBI, it Iterator) error {
	for {
		// Get key
		key, err := it.Next()
		if err != nil {
			if err == io.EOF {
				return nil // done
			}
			return errors.Wrap(err, "next")
		}

		dbv, err := txn.Get(dbi, key) // Error not a problem, does not need to be there
		if err != nil && !lmdb.IsNotFound(err) {
			return errors.Wrap(err, "get")
		}

		// Get new val by merging existing and iterator's val
		val, err := it.Merge(dbv)
		if err != nil {
			return errors.Wrap(err, "merge")
		}

		if err := setNewVal(txn, dbi, key, dbv, val); err != nil {
			return err
		}
	}
}
