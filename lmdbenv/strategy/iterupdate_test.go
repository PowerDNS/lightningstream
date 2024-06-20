package strategy

import (
	"testing"

	"github.com/PowerDNS/lightningstream/lmdbenv"
)

func TestIterUpdate_empty(t *testing.T) {
	// Only replace exiting items
	oldItems := []lmdbenv.KVString{}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"yyy", ""},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdate_merge(t *testing.T) {
	// Only replace exiting items
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "A1Xa"},
		{"zzz", "A4Xz"},
	}
	doStrategyTestNS(t, IterUpdate, oldItems, items, exp, 'A')
}

func TestIterUpdate_merge_same(t *testing.T) {
	// Only replace exiting items
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "a"},
		{"zzz", "z"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"zzz", "Xz"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdate_merge_with_before(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aa", "XA"},
		{"aaa", "XA"},
		{"zzz", "XZ"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdate_merge_with_after(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"zzz", "Xz"},
		{"zzzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdate_merge_with_prepend(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aa", "a"},
		{"aaa", "1"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aa", "Xa"},
		{"aaa", "X1"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdate_merge_with_append(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"zzz", "4"},
		{"zzzz", "z"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"zzz", "X4"},
		{"zzzz", "Xz"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdate_insert(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"ccc", "2"},
		{"ddd", "3"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"ccc", "X2"},
		{"ddd", "X3"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdate_del(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"ccc", ""},
		{"ddd", ""},
		{"zzz", ""},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, nil)
}

func TestIterUpdate_del_insert_del(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", ""},
		{"bbb", "b"},
		{"zzz", ""},
	}
	exp := []lmdbenv.KVString{
		{"bbb", "Xb"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdate_del_put_del(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"ccc", "Xc"},
		{"ddd", "Xd"},
		{"eee", "Xe"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", ""},
		{"ddd", "3"},
		{"zzz", ""},
	}
	exp := []lmdbenv.KVString{
		{"ddd", "X3"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdate_insert_del_insert(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"ddd", "Xd"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"ddd", ""},
		{"eee", "3"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"eee", "X3"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdate_wider(t *testing.T) {
	// Adding items before and after existing set
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"c", "Xc"},
		{"d", "Xd"},
		{"f", "Xf"},
		{"g", "Xg"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"a", "1"},
		{"aaa", "1"},
		{"bbbb", "2"},
		{"c", "3"},
		{"d", "3"},
		{"dddddd", "4"},
		{"zzz", "4"},
		{"zzzzzzzzz", "5"},
	}
	exp := []lmdbenv.KVString{
		{"a", "X1"},
		{"aaa", "X1"},
		{"bbbb", "X2"},
		{"c", "X3"},
		{"d", "X3"},
		{"dddddd", "X4"},
		{"zzz", "X4"},
		{"zzzzzzzzz", "X5"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdate_narrower(t *testing.T) {
	// Only adding items between existing set
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"c", "Xc"},
		{"d", "Xd"},
		{"f", "Xf"},
		{"g", "Xg"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaaa", "1"},
		{"bbbb", "2"},
		{"c", "3"},
		{"d", "3"},
		{"dddddd", "4"},
		{"zz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaaa", "X1"},
		{"bbbb", "X2"},
		{"c", "X3"},
		{"d", "X3"},
		{"dddddd", "X4"},
		{"zz", "X4"},
	}
	doStrategyTest(t, IterUpdate, oldItems, items, exp)
}

func TestIterUpdateNS(t *testing.T) {
	items1 := []lmdbenv.KVString{
		{"aa", "Z1"}, // will not be touched
		{"aaa", "A1Z4"},
		{"aaa3", "A1Z4"}, // will only have A stripped
		{"bbbb", "C2"},
		{"c", "X3"},
		{"cc", "Z3"}, // will not be touched
		{"dddddd", "X4"},
		{"del", "A9"},  // will be deleted
		{"zz", "V1Z4"}, // will not be touched
	}
	items2 := []lmdbenv.KVString{
		{"aaa", "1"},
		{"aaa2", "2"}, // will be added
		{"aaa3", ""},
		{"bbbb", "2"},
		{"c", "3"},
		{"dddddd", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aa", "Z1"},
		{"aaa", "A1Z4"},
		{"aaa2", "A2"},
		{"aaa3", "Z4"},
		{"bbbb", "A2C2"},
		{"c", "A3X3"},
		{"cc", "Z3"},
		{"dddddd", "A4X4"},
		{"zz", "V1Z4"},
	}
	doStrategyTestNS(t, IterUpdate, items1, items2, exp, 'A')
}
