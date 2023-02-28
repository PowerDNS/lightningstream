package lmdbenv

import "github.com/PowerDNS/lmdb-go/lmdb"

func DBIExists(txn *lmdb.Txn, dbiName string) (bool, error) {
	_, err := txn.OpenDBI(dbiName, 0)
	if err != nil {
		if lmdb.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
