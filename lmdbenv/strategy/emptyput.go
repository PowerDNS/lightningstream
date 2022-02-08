package strategy

import (
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
)

// EmptyPut firsts empties the DBI, then uses Put to insert all the values.
func EmptyPut(txn *lmdb.Txn, dbi lmdb.DBI, it Iterator) error {
	if err := txn.Drop(dbi, false); err != nil { // empty without deleting the DBI
		return errors.Wrap(err, "empty dbi")
	}
	if err := Put(txn, dbi, it); err != nil {
		return errors.Wrap(err, "put")
	}
	return nil
}
