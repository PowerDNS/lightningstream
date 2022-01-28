package strategy

import (
	"testing"

	"powerdns.com/platform/lightningstream/lmdbenv"
)

func TestIterPut_empty(t *testing.T) {
	// Only replace exiting items
	oldItems := []lmdbenv.KVString{}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_replace(t *testing.T) {
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
		{"aaa", "X1"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_replace_same(t *testing.T) {
	// Only replace exiting items
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "a"},
		{"zzz", "z"},
	}
	doStrategyTest(t, IterPut, oldItems, items, oldItems)
}

func TestIterPut_replace_with_before(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aa", "Xa"},
		{"aaa", "Xa"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_replace_with_after(t *testing.T) {
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
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_replace_with_prepend(t *testing.T) {
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
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_replace_with_append(t *testing.T) {
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
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_insert(t *testing.T) {
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
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_del(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"ccc", "Xc"},
		{"ddd", "Xd"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_del_insert_del(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"ccc", "Xc"},
		{"eee", "Xe"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"ddd", "3"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"ddd", "X3"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_del_put_del(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"ccc", "Xc"},
		{"ddd", "Xd"},
		{"eee", "Xe"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"ddd", "3"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"ddd", "X3"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_insert_del_insert(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"ddd", "Xd"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"ccc", "2"},
		{"eee", "3"},
		{"zzz", "4"},
	}
	exp := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"ccc", "X2"},
		{"eee", "X3"},
		{"zzz", "X4"},
	}
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_insert_del_pure(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"ddd", "Xd"},
	}
	items := []lmdbenv.KVString{
		{"ccc", "2"},
	}
	exp := []lmdbenv.KVString{
		{"ccc", "X2"},
	}
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_wider(t *testing.T) {
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
	doStrategyTest(t, IterPut, oldItems, items, exp)
}

func TestIterPut_narrower(t *testing.T) {
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
	doStrategyTest(t, IterPut, oldItems, items, exp)
}
