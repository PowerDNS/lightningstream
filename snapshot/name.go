package snapshot

import (
	"fmt"
	"strings"
	"time"

	"powerdns.com/platform/lightningstream/utils"
)

const (
	timeFormat = "20060102-150405.000000000" // but need to s/./-/
	dotIndex   = 15                          // position of the '.'
)

func Timestamp(ts time.Time) string {
	fileTimestamp := strings.Replace(
		ts.UTC().Format(timeFormat),
		".", "-", 1)
	return fileTimestamp
}

func TimestampFromNano(tsNano uint64) string {
	ts := time.Unix(0, int64(tsNano))
	return Timestamp(ts)
}

func Name(syncerName, instanceID, generationID string, ts time.Time) string {
	fileTimestamp := Timestamp(ts)
	name := fmt.Sprintf("%s__%s__%s__%s.pb.gz",
		syncerName,
		instanceID,
		fileTimestamp,
		generationID,
	)
	return name
}

func ParseName(name string) (NameInfo, error) {
	var ni, empty NameInfo
	basename, ext, found := utils.Cut(name, ".")
	if !found {
		return empty, fmt.Errorf("invalid name: no dot: %s", name)
	}
	if ext != "pb.gz" {
		return empty, fmt.Errorf("unexpected extension: %s", name)
	}
	ni.FullName = name
	ni.Extension = ext
	p := strings.Split(basename, "__")
	if len(p) < 4 {
		return empty, fmt.Errorf("not enough name parts: %s", name)
	}
	ni.SyncerName = p[0]
	ni.InstanceID = p[1]
	ni.TimestampString = p[2]
	ni.GenerationID = p[3]
	tss := ni.TimestampString
	if len(tss) != len(timeFormat) || tss[dotIndex] != '-' {
		return empty, fmt.Errorf("invalid timestamp format: %s in %s", tss, name)
	}
	tss = tss[:dotIndex] + "." + tss[dotIndex+1:] // replace second '-' with '.' for parsing
	ts, err := time.Parse(timeFormat, tss)        // returns time in UTC
	if err != nil {
		return empty, fmt.Errorf("timestamp parse error: %s", err)
	}
	ni.Timestamp = ts
	return ni, nil
}

type NameInfo struct {
	FullName        string
	Extension       string // "pb.gz"
	SyncerName      string
	InstanceID      string
	GenerationID    string
	TimestampString string
	Timestamp       time.Time
}
