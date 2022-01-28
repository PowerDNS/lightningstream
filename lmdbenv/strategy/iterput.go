package strategy

import (
	"bytes"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
)

// IterPut implements the IterPut strategy.
//
// Iter over all existing keys and update all URLs we are processing:
//
// - Add new urls and replace values of existing, just like in the `Put` strategy.
// - But also remove keys that are not present in the snapshot.
//
// Once we have reached the end of the database, we use `Append` instead of `Put`.
//
// Prerequisites:
//
// - Sorted input
func IterPut(txn *lmdb.Txn, dbi lmdb.DBI, it Iterator) error {
	c, err := txn.OpenCursor(dbi)
	if err != nil {
		return errors.Wrap(err, "open cursor")
	}
	defer c.Close()

	err = iterBoth(it, c, func(itKey, dbKey, dbVal []byte, itEOF, dbEOF bool) error {
		//log.Printf("@@@ args: %s, %s, %s, %v, %v", string(itKey), string(dbKey), string(dbVal), itEOF, dbEOF)

		// Database cursor behind
		if itEOF || itKey == nil {
			// Delete current key from LMDB
			//log.Printf("@@@ Del %s %v", string(dbKey), itEOF)
			err := c.Del(0) // Does not affect cursor position, safe
			if err != nil {
				return errors.Wrap(err, "del current")
			}
			return nil
		}

		// We need the value for all the code below
		val, err := it.Merge(nil)
		if err != nil {
			return errors.Wrap(err, "merge")
		}

		// At end of database, append any new item
		if dbEOF {
			// Append all remaining keys
			err = c.Put(itKey, val, lmdb.Append) // Safe to use cursor, because at end
			//log.Printf("@@@ Append %s", string(itKey))
			if err != nil {
				return errors.Wrap(err, "lmdb append")
			}
			return nil
		}

		// Iterator behind database cursor, add new keys
		if dbKey == nil {
			// Insert new key with normal Put
			//log.Printf("@@@ Put %s", string(itKey))
			err = txn.Put(dbi, itKey, val, 0) // Do not touch cursor
			if err != nil {
				return errors.Wrap(err, "put")
			}
			return nil
		}

		// Both keys the same, overwrite value if different
		if bytes.Equal(val, dbVal) {
			//log.Printf("@@@ Same value")
			return nil // Already the same, no need to Put
		}
		//log.Printf("@@@ Put overwrite %s", string(itKey))
		err = txn.Put(dbi, itKey, val, 0) // Do not touch cursor
		if err != nil {
			return errors.Wrap(err, "put current")
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "iterBoth")
	}

	return nil
}
