package strategy

import (
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/PowerDNS/lmdb-go/lmdb"
	"github.com/pkg/errors"

	"powerdns.com/platform/lightningstream/lmdbenv"
)

type TestIterator struct {
	kvs []lmdbenv.KVString
	idx int
	ns  byte
}

func NewTestIterator(kvs []lmdbenv.KVString, ns byte) *TestIterator {
	return &TestIterator{
		kvs: kvs,
		idx: -1,
		ns:  ns,
	}
}

func (it *TestIterator) Next() (key []byte, err error) {
	it.idx++
	if it.idx >= len(it.kvs) {
		return nil, io.EOF
	}
	item := it.kvs[it.idx]
	return []byte(item.Key), nil
}

func (it *TestIterator) Merge(oldval []byte) (ret []byte, err error) {
	if len(oldval)%2 != 0 {
		return nil, fmt.Errorf("merge: len(oldval) must be even")
	}

	if it.ns == 0 {
		if oldval != nil {
			return nil, fmt.Errorf("merge: oldval is not nil but no namespace given")
		}
		return []byte(it.kvs[it.idx].Val), nil
	}

	var m [256]map[byte]bool
	for i := 0; i < len(oldval); i += 2 {
		ns := oldval[i]
		if m[ns] == nil {
			m[ns] = make(map[byte]bool)
		}
		m[ns][oldval[i+1]] = true
	}
	m[it.ns] = make(map[byte]bool)
	val := it.kvs[it.idx].Val
	//log.Printf("oldval is '%s'\n", oldval)
	//log.Printf("val is '%s'\n", val)
	for i := 0; i < len(val); i++ {
		m[it.ns][val[i]] = true
	}
	// We want to keep things in a specific order so the randomized range map does not hurt us.
	// This is a bit of a stupid loop, but it's only test code.
	for i := 0; i < 256; i++ {
		if len(m[i]) > 0 {
			for j := 0; j < 256; j++ {
				if m[i][byte(j)] {
					ret = append(ret, byte(i))
					ret = append(ret, byte(j))
				}
			}
		}
	}
	//log.Printf("ret is %s '%s'\n", it.kvs[it.idx].Key, ret)
	return ret, nil
}

func (it *TestIterator) Clean(oldval []byte) (ret []byte, err error) {
	if len(oldval)%2 != 0 {
		return nil, fmt.Errorf("merge: len(oldval) must be even")
	}

	if it.ns == 0 {
		if oldval != nil {
			return nil, fmt.Errorf("merge: oldval is not nil but no namespace given")
		}
		return []byte(it.kvs[it.idx].Val), nil
	}

	var m [256]map[byte]bool
	for i := 0; i < len(oldval); i += 2 {
		ns := oldval[i]
		if m[ns] == nil {
			m[ns] = make(map[byte]bool)
		}
		m[ns][oldval[i+1]] = true
	}
	m[it.ns] = nil
	// We want to keep things in a specific order so the randomized range map does not hurt us.
	// This is a bit of a stupid loop, but it's only test code.
	for i := 0; i < 256; i++ {
		if len(m[i]) > 0 {
			for j := 0; j < 256; j++ {
				if m[i][byte(j)] {
					ret = append(ret, byte(i))
					ret = append(ret, byte(j))
				}
			}
		}
	}
	//log.Printf("ret is %s '%s'\n", it.kvs[it.idx].Key, ret)
	return ret, nil
}

func prefillDBI(txn *lmdb.Txn, dbi lmdb.DBI, items []lmdbenv.KVString) error {
	it := NewTestIterator(items, 0)
	err := Put(txn, dbi, it)
	if err != nil {
		return errors.Wrap(err, "prefill: Put")
	}

	dbItems, err := lmdbenv.ReadDBIString(txn, dbi)
	if err != nil {
		return errors.Wrap(err, "prefill: ReadDBIString")
	}

	if len(dbItems) == 0 {
		dbItems = nil
	}
	if len(items) == 0 {
		items = nil
	}
	if !reflect.DeepEqual(dbItems, items) {
		return fmt.Errorf("prepare: expected items to be equal:\nExp: %v\nGot: %v", items, dbItems)
	}

	return nil
}

// Values in lmdb are formatted in alternating NS byte and data byte values.
// The values we get from the iterator all belong to a specified namespace.
// By default, we use the 'X' namespace.
// Th expected value are read form the lmdb, so they include a namespace byte
//
// So oldItems values are formatted as  k = NVMW... (where N and M are namespaces and V and W values)
// items are formatted as k = VW... (only value bytes)
// expItems are formatted the same way as oldItems.
func doStrategyTestNS(t *testing.T, f Func, oldItems, items, expItems []lmdbenv.KVString, ns byte) {
	err := lmdbenv.TestTxn(func(txn *lmdb.Txn, dbi lmdb.DBI) error {
		// First insert some old data
		if oldItems != nil {
			if err := prefillDBI(txn, dbi, oldItems); err != nil {
				t.Fatalf("prepare: %v", err)
			}
		}

		it := NewTestIterator(items, ns)
		err := f(txn, dbi, it) // call strategy
		if err != nil {
			t.Fatalf("Strategy returned error: %v", err)
		}

		dbItems, err := lmdbenv.ReadDBIString(txn, dbi)
		if err != nil {
			t.Fatalf("ReadDBIString returned error: %v", err)
		}

		if !reflect.DeepEqual(dbItems, expItems) {
			t.Fatalf("Expected items to be equal:\nExp: %v\nGot: %v", expItems, dbItems)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("TestTxn error: %v", err)
	}
}

// By default we process the items as if they ware in the space (' ') namespace
func doStrategyTest(t *testing.T, f Func, oldItems, items, expItems []lmdbenv.KVString) {
	doStrategyTestNS(t, f, oldItems, items, expItems, 'X')
}
