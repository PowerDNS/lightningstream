package strategy

import (
	"fmt"
	"io"

	"github.com/PowerDNS/lmdb-go/lmdb"
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
			return fmt.Errorf("next: %w", err)
		}

		dbv, err := txn.Get(dbi, key) // Error not a problem, does not need to be there
		if err != nil && !lmdb.IsNotFound(err) {
			return fmt.Errorf("get: %w", err)
		}

		// Get new val by merging existing and iterator's val
		val, err := it.Merge(dbv)
		if err != nil {
			return fmt.Errorf("merge: %w", err)
		}

		if err := setNewVal(txn, dbi, key, dbv, val); err != nil {
			return err
		}
	}
}
