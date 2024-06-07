package strategy

import (
	"testing"

	"github.com/PowerDNS/lightningstream/lmdbenv"
	"github.com/PowerDNS/lmdb-go/lmdb"
)

func TestAppend(t *testing.T) {
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"bbbb", "2"},
		{"c", "3"},
		{"dddddd", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"bbbb", "X2"},
		{"c", "X3"},
		{"dddddd", "X4"},
	}
	doStrategyTest(t, Append, nil, items, exp)
}

func TestAppend_unsorted(t *testing.T) {
	items := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"c", "X3"},
		{"bbbb", "X2"}, // Note: not sorted
		{"dddddd", "X4"},
	}

	err := lmdbenv.TestTxn(func(txn *lmdb.Txn, dbi lmdb.DBI) error {
		it := NewTestIterator(items, ' ')
		err := Append(txn, dbi, it)
		if err != nil {
			if err == ErrNotSorted {
				return nil // Excellent
			}
			t.Fatalf("Strategy returned unexpected error: %v", err)
		}
		t.Fatalf("Strategy returned no error, expected: %v", ErrNotSorted)
		return nil
	})
	if err != nil {
		t.Fatalf("TestTxn error: %v", err)
	}
}
