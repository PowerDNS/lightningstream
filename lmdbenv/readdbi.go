package lmdbenv

import (
	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"
)

type KV struct {
	Key, Val []byte
}

// ReadDBI reads all values in a DBI and returns them as a slice.
// This is useful for tests.
func ReadDBI(txn *lmdb.Txn, dbi lmdb.DBI) ([]KV, error) {
	c, err := txn.OpenCursor(dbi)
	if err != nil {
		return nil, errors.Wrap(err, "open cursor")
	}
	defer c.Close()

	var entries []KV
	var flag uint = lmdb.First
	for {
		key, val, err := c.Get(nil, nil, flag)
		if err != nil {
			if lmdb.IsNotFound(err) {
				return entries, nil // done
			} else {
				return nil, errors.Wrap(err, "cursor next")
			}
		}
		flag = lmdb.Next
		entries = append(entries, KV{key, val})
	}
}

type KVString struct {
	Key, Val string
}

// ReadDBIString is a wrapper around ReadDBI() that returns all values as strings
// instead of []byte.
func ReadDBIString(txn *lmdb.Txn, dbi lmdb.DBI) ([]KVString, error) {
	kv, err := ReadDBI(txn, dbi)
	if err != nil {
		return nil, err
	}
	var kvs []KVString
	for _, item := range kv {
		kvs = append(kvs, KVString{string(item.Key), string(item.Val)})
	}
	return kvs, nil
}

// ReadDBINames reads all DBI names from the root database
func ReadDBINames(txn *lmdb.Txn) ([]string, error) {
	// Read all database names
	rootDBI, err := txn.OpenRoot(0)
	if err != nil {
		return nil, err
	}
	kvs, err := ReadDBI(txn, rootDBI)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, item := range kvs {
		names = append(names, string(item.Key))
	}
	return names, nil
}
