package strategy

import (
	"fmt"

	"github.com/PowerDNS/lmdb-go/lmdb"
)

// EmptyPut firsts empties the DBI, then uses Put to insert all the values.
func EmptyPut(txn *lmdb.Txn, dbi lmdb.DBI, it Iterator) error {
	if err := txn.Drop(dbi, false); err != nil { // empty without deleting the DBI
		return fmt.Errorf("empty dbi: %w", err)
	}
	if err := doPut(txn, dbi, it, true); err != nil {
		return fmt.Errorf("put: %w", err)
	}
	return nil
}
