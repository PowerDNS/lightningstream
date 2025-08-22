package lmdbstats

import (
	"strconv"
	"strings"
)

// getMemoryStats returns the memory stats for the LMDB file as a map.
// 'smaps' is the full content of the proc smaps file.
// 'dbpath' is the full path to the LMDB data.mdb file, as expected in smaps.
func getMemoryStats(smaps, dbpath string) map[string]int64 {
	// This code is a converted from the original Python implementation
	offset := strings.Index(smaps, dbpath)
	if offset <= 0 {
		return nil
	}

	// Get next 1 kB (actual on my system is 540 bytes) and split on lines
	lines := strings.Split(smaps[offset:offset+1024], "\n")

	m := make(map[string]int64)
	for i, line := range lines {
		if i == 0 {
			continue // Skip the first line, as it's the one with the data.mdb path
		}
		if !strings.Contains(line, " kB") {
			break
		}
		p := strings.Fields(line)
		if len(p) < 3 {
			continue
		}
		fullkey := p[0]
		key := fullkey[:len(fullkey)-1] // do not include last ':'
		lkey := strings.ToLower(key)
		v, err := strconv.ParseInt(p[1], 10, 64)
		if err != nil {
			continue
		}
		m[lkey] = v * 1024 // value as bytes
	}
	return m
}
