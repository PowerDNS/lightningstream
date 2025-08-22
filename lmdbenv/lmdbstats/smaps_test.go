package lmdbstats

import (
	"testing"
)

func TestGetMemoryStats(t *testing.T) {
	m := getMemoryStats(testSmaps, "/lmdbdata/db/data.mdb")
	t.Logf("smaps data: %+v", m)

	tt := []struct {
		key string
		val int64
	}{
		{"size", 10485760 * 1024},
		{"rss", 6232 * 1024},
		{"shared_clean", 1892 * 1024},
		{"shared_dirty", 0 * 1024},
		{"private_clean", 4340 * 1024},
		{"private_dirty", 0 * 1024},
		{"referenced", 6232 * 1024},
	}

	for _, item := range tt {
		if m[item.key] != item.val {
			t.Errorf("key %s: expected %d, got %d", item.key, item.val, m[item.key])
		}
	}
}

// testSmaps is a sample of /proc/<pid>/smaps output with an LMDB mapping in it.
// This one was collected from Linux kernel 4.15.
var testSmaps = `
...
7f6ff5b7d000-7f6ff5b7e000 ---p 00000000 00:00 0
Size:                  4 kB
KernelPageSize:        4 kB
MMUPageSize:           4 kB
Rss:                   0 kB
Pss:                   0 kB
Shared_Clean:          0 kB
Shared_Dirty:          0 kB
Private_Clean:         0 kB
Private_Dirty:         0 kB
Referenced:            0 kB
Anonymous:             0 kB
LazyFree:              0 kB
AnonHugePages:         0 kB
ShmemPmdMapped:        0 kB
Shared_Hugetlb:        0 kB
Private_Hugetlb:       0 kB
Swap:                  0 kB
SwapPss:               0 kB
Locked:                0 kB
VmFlags: mr mw me sd
7f6ff5b7e000-7f6ff637e000 rw-p 00000000 00:00 0
Size:               8192 kB
KernelPageSize:        4 kB
MMUPageSize:           4 kB
Rss:                  36 kB
Pss:                  36 kB
Shared_Clean:          0 kB
Shared_Dirty:          0 kB
Private_Clean:         0 kB
Private_Dirty:        36 kB
Referenced:           36 kB
Anonymous:            36 kB
LazyFree:              0 kB
AnonHugePages:         0 kB
ShmemPmdMapped:        0 kB
Shared_Hugetlb:        0 kB
Private_Hugetlb:       0 kB
Swap:                  0 kB
SwapPss:               0 kB
Locked:                0 kB
VmFlags: rd wr mr mw me ac sd
7f6ff637e000-7f727637e000 r--s 00000000 103:02 1968305                   /lmdbdata/db/data.mdb
Size:           10485760 kB
KernelPageSize:        4 kB
MMUPageSize:           4 kB
Rss:                6232 kB
Pss:                4959 kB
Shared_Clean:       1892 kB
Shared_Dirty:          0 kB
Private_Clean:      4340 kB
Private_Dirty:         0 kB
Referenced:         6232 kB
Anonymous:             0 kB
LazyFree:              0 kB
AnonHugePages:         0 kB
ShmemPmdMapped:        0 kB
Shared_Hugetlb:        0 kB
Private_Hugetlb:       0 kB
Swap:                  0 kB
SwapPss:               0 kB
Locked:                0 kB
VmFlags: rd sh mr mw me ms sd
7f727637e000-7f72766c0000 rw-p 00000000 00:00 0
Size:               3336 kB
KernelPageSize:        4 kB
MMUPageSize:           4 kB
Rss:                 264 kB
Pss:                 264 kB
Shared_Clean:          0 kB
Shared_Dirty:          0 kB
Private_Clean:         0 kB
Private_Dirty:       264 kB
Referenced:          264 kB
Anonymous:           264 kB
LazyFree:              0 kB
AnonHugePages:         0 kB
ShmemPmdMapped:        0 kB
Shared_Hugetlb:        0 kB
Private_Hugetlb:       0 kB
Swap:                  0 kB
SwapPss:               0 kB
Locked:                0 kB
VmFlags: rd wr mr mw me ac sd
...
`
