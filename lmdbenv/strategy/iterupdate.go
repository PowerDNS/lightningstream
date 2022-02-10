package strategy

import (
	"bytes"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
)

// IterUpdate implements the IterUpdate strategy.
//
// Iter over all existing keys and update all URLs we are processing:
//
// - Add new keys and data, just like in the `update` strategy.
// - But also remove codes from urls that have disappeared for this namespace.
//
// Once we have reached the end of the database, we use `Update` instead of `Append`.
//
// Prerequisites:
//
// - Sorted input
//
// Uses: merging data
func IterUpdate(txn *lmdb.Txn, dbi lmdb.DBI, it Iterator) error {
	c, err := txn.OpenCursor(dbi)
	if err != nil {
		return errors.Wrap(err, "open cursor")
	}
	defer c.Close()

	integerKey := false
	flags, err := txn.Flags(dbi)
	if err != nil {
		return errors.Wrap(err, "get flags")
	}
	if flags&LMDBIntegerKeyFlag > 0 {
		integerKey = true
	}

	err = iterBoth(it, c, integerKey, func(itKey, dbKey, dbVal []byte, itEOF, dbEOF bool) error {
		//log.Printf("@@@ args: itkey=%s, dbkey=%s, dbVal=%s, itEOF=%v, dbEOF=%v", string(itKey), string(dbKey), string(dbVal), itEOF, dbEOF)

		if itEOF || itKey == nil {
			val, err := it.Clean(dbVal)
			if err != nil {
				return errors.Wrap(err, "clean")
			}
			if val == nil {
				//log.Printf("@@@ Del itKey nil")
				err = c.Del(0)
				if err != nil {
					return errors.Wrap(err, "del cleaned")
				}
				return nil
			}
			if bytes.Equal(val, dbVal) {
				//log.Printf("@@@ Same value")
				return nil // Already the same, no need to Put
			}
			err = txn.Put(dbi, dbKey, val, 0) // Do not touch cursor
			if err != nil {
				return errors.Wrap(err, "put cleaned")
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
			if len(val) == 0 {
				return nil
			}
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
			if len(val) == 0 {
				return nil
			}
			// Insert new key with normal Put
			//log.Printf("@@@ Put %s", string(itKey))
			err = txn.Put(dbi, itKey, val, 0) // Do not touch cursor
			if err != nil {
				return errors.Wrap(err, "put")
			}
			return nil
		}

		val, err = it.Merge(dbVal)
		if err != nil {
			return errors.Wrap(err, "merge")
		}
		// Both keys the same, overwrite value if different
		if len(val) == 0 {
			//log.Printf("@@@ Del")
			err = c.Del(0)
			if err != nil {
				return errors.Wrap(err, "del")
			}
			return nil
		}
		if bytes.Equal(val, dbVal) {
			//log.Printf("@@@ Same value")
			return nil // Already the same, no need to Put
		}
		//log.Printf("@@@ Put %v", val)
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
