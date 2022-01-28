package strategy

import (
	"testing"

	"powerdns.com/platform/lightningstream/lmdbenv"
)

func TestPut(t *testing.T) {
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
	doStrategyTest(t, Put, nil, items, exp)
}

func TestPut_existing(t *testing.T) {
	oldItems := []lmdbenv.KVString{
		{"aaa", "Xa"},
		{"zzz", "Xz"},
	}
	items := []lmdbenv.KVString{
		{"aaa", "1"},
		{"bbbb", "2"},
		{"c", "3"},
		{"dddddd", "4"},
	}
	expItems := []lmdbenv.KVString{
		{"aaa", "X1"},
		{"bbbb", "X2"},
		{"c", "X3"},
		{"dddddd", "X4"},
		{"zzz", "Xz"},
	}
	doStrategyTest(t, Put, oldItems, items, expItems)
}
