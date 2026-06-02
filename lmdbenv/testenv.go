package lmdbenv

import (
	"errors"
	"fmt"
	"os"

	"github.com/PowerDNS/lmdb-go/lmdb"
)

type TestEnvFunc func(env *lmdb.Env) error

// TestEnv creates a temporary LMDB database and calls the given test function
// with the temporary LMDB Env. Any error returned by this function is returned
// unmodified to the caller.
func TestEnv(f TestEnvFunc) error {
	tmpdir, err := os.MkdirTemp("", "lmdbtest_")
	if err != nil {
		return fmt.Errorf("create tempdir: %w", err)
	}
	if tmpdir == "" {
		panic("Empty tmpdir")
	}
	defer os.RemoveAll(tmpdir)

	env, err := New(tmpdir, lmdb.Create)
	if err != nil {
		return fmt.Errorf("new lmdb env: %w", err)
	}

	if err := f(env); err != nil {
		return err
	}
	return nil
}

type TestTxnFunc func(txn *lmdb.Txn, dbi lmdb.DBI) error

// TestTxn creates a temporary LMDB database and calls the given test function
// with a write transaction and a new DBI that will be rolled back on return.
// This is a convenience wrapper around TestEnv().
// Any error returned by this function is returned unmodified to the caller.
func TestTxn(f TestTxnFunc) error {
	noErr := errors.New("no error")
	return TestEnv(func(env *lmdb.Env) error {
		err := env.Update(func(txn *lmdb.Txn) error {
			dbi, err := txn.OpenDBI("tempdbi", lmdb.Create)
			if err != nil {
				return fmt.Errorf("create dbi: %w", err)
			}
			if err := f(txn, dbi); err != nil {
				return err
			}
			return noErr // to rollback the transaction
		})
		if err != nil && err != noErr {
			return err
		}
		return nil
	})
}
