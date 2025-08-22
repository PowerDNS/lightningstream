package lmdbstats

import "github.com/PowerDNS/lmdb-go/lmdb"

// PageUsageBytes estimates bytes of map size used based on used pages
func PageUsageBytes(s *lmdb.Stat) uint64 {
	return uint64(s.PSize) * (s.BranchPages + s.LeafPages + s.OverflowPages)
}
