package utils

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/sirupsen/logrus"
)

// SleepContext sleeps for given duration. If the context closes in the
// meantime, it returns immediately with a context.Canceled error.
func SleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return context.Canceled
	case <-t.C:
		return nil
	}
}

// SleepContextPerturb sleeps for given duration like SleepContent, but it
// perturbs the duration with a 20% random component to avoid multiple instances
// running at the exact same time.
// If the context closes in the meantime, it returns immediately with a
// context.Canceled error.
func SleepContextPerturb(ctx context.Context, d time.Duration) error {
	r := rand.Intn(400)
	// Random duration between 80% and 120% of original
	d = time.Duration(800+r) * d / 1000
	return SleepContext(ctx, d)
}

// IsCanceled checks if the context has been canceled.
func IsCanceled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// DisplayASCII represents a key as ascii if it only contains safe ascii characters.
// If it contains unsafe characters, these are replaced by '.' and a hex
// representation is added to the output.
// It the first 8 bytes look like a nanosecond UNIX timestamp, that will be shown too.
// TODO: Support full LS header
func DisplayASCII(b []byte) string {
	ret := make([]byte, len(b))
	unsafe := false
	for i, ch := range b {
		if ch < 32 || ch > 126 {
			ret[i] = '.'
			unsafe = true
		} else {
			ret[i] = ch
		}
	}
	var tsString string
	if len(b) >= 8 {
		tsNano := int64(binary.BigEndian.Uint64(b[:8]))
		ts := time.Unix(0, tsNano).UTC()
		isRecentTS := ts.After(time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)) &&
			ts.Before(time.Now().Add(30*24*time.Hour))
		if isRecentTS {
			tsString = ts.Format(time.RFC3339Nano)
		}
	}
	if unsafe || len(b) <= 8 || tsString != "" {
		if tsString != "" {
			return fmt.Sprintf("%s [% 0x] (%s)", string(ret), b, tsString)

		} else {
			return fmt.Sprintf("%s [% 0x]", string(ret), b)
		}
	}
	return string(ret)
}

// Cut cuts s around the first instance of sep,
// returning the text before and after sep.
// The found result reports whether sep appears in s.
// If sep does not appear in s, cut returns s, "", false.
//
// This is a copy of strings.Cut from Go 1.18,
// see https://github.com/golang/go/issues/46336
// TODO: remove when we switch to Go 1.18 and use strings.Cut
func Cut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

// TimeDiff returns the difference between two times, rounded to milliseconds.
func TimeDiff(t1, t0 time.Time) time.Duration {
	return t1.Sub(t0).Round(time.Millisecond)
}

// GC runs the garbage collector and logs some memory stats
func GC() time.Duration {
	var before, after runtime.MemStats
	t0 := time.Now()
	runtime.ReadMemStats(&before)
	runtime.GC()
	runtime.ReadMemStats(&after)
	t1 := time.Now()
	dt := TimeDiff(t1, t0)
	freed := after.Frees - before.Frees
	logrus.WithFields(logrus.Fields{
		"time_gc":      dt,
		"freed":        datasize.ByteSize(freed),
		"alloc_before": datasize.ByteSize(before.Alloc),
		"alloc_after":  datasize.ByteSize(after.Alloc),
	}).Debug("GC stats")
	return dt
}
