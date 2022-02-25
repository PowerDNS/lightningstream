package strategy

import (
	"testing"

	"powerdns.com/platform/lightningstream/lmdbenv"
)

// Test iterator does not do dedup!
// A new code for an existing entry just gets appended.

func TestUpdate(t *testing.T) {
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
	doStrategyTest(t, Update, nil, items, exp)
}

func TestUpdateDup(t *testing.T) {
	items1 := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"bbbb", "X2"},
		{"c", "X3"},
		{"dddddd", "X4"},
	}
	items2 := []lmdbenv.KVString{
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
	doStrategyTest(t, Update, items1, items2, exp)
}

func TestUpdateAdd(t *testing.T) {
	olditems := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"bbbb", "X2"},
		{"c", "X3"},
		{"dddddd", "X4"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "2"},
		{"eee", "5"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X2"},
		{"bbbb", "X2"},
		{"c", "X3"},
		{"dddddd", "X4"},
		{"eee", "X5"},
	}
	doStrategyTest(t, Update, olditems, items, exp)
}

func TestUpdateDel(t *testing.T) {
	olditems := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"bbbb", "Xb"},
		{"c", "X3"},
		{"dddddd", "X4"},
		{"eee", "X5"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "2"},
		{"eee", ""},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X2"},
		{"bbbb", "Xb"},
		{"c", "X3"},
		{"dddddd", "X4"},
	}
	doStrategyTest(t, Update, olditems, items, exp)
}

func TestUpdateNS(t *testing.T) {
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"bbbb", "2"},
		{"c", "3"},
		{"dddddd", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "A1"},
		{"bbbb", "A2"},
		{"c", "A3"},
		{"dddddd", "A4"},
	}
	doStrategyTestNS(t, Update, nil, items, exp, 'A')
}

func TestUpdateDupNS(t *testing.T) {
	items1 := []lmdbenv.KVString{
		{"aaa", "A1"},
		{"bbbb", "C2"},
		{"c", "X3"},
		{"dddddd", "X4"},
	}
	items2 := []lmdbenv.KVString{
		{"aaa", "1"},
		{"bbbb", "2"},
		{"c", "3"},
		{"dddddd", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "A1"},
		{"bbbb", "A2C2"},
		{"c", "A3X3"},
		{"dddddd", "A4X4"},
	}
	doStrategyTestNS(t, Update, items1, items2, exp, 'A')
}
