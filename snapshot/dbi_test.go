package snapshot

import (
	"encoding/binary"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func makeTestDBI(n int) *DBI {
	d := NewDBI()
	// This also tests implicit flushing of these fields when writing a KV
	d.SetName("this-will-be-overridden")
	d.SetName("test-name")
	d.SetTransform("test-transform")
	d.SetFlags(42)
	extra := []byte("TEST1234567890ABCDEF") // 20 extra bytes
	for i := 0; i < n; i++ {
		key := append([]byte{'k', 0, 0, 0, 0}, extra...)
		binary.BigEndian.PutUint32(key[1:5], uint32(i))
		val := append([]byte{'v', byte(i)}, extra...)
		d.Append(KV{
			Key:           key,
			Value:         val,
			Flags:         uint32(i) % 2,
			TimestampNano: uint64(i),
		})
	}
	return d
}

func TestDBI(t *testing.T) {
	pbdata := makeTestDBI(1_000).Marshal()

	d, err := NewDBIFromData(pbdata)
	assert.NoError(t, err)
	assert.Equal(t, "test-name", d.Name())
	assert.Equal(t, "test-transform", d.Transform())
	assert.Equal(t, uint64(42), d.Flags())
	assert.NotContains(t, pbdata, []byte("this-will-be-overridden"))

	// Data that has been loaded cannot have top-level fields modified
	assert.Panics(t, func() {
		d.SetName("not-allowed")
	})
	assert.Panics(t, func() {
		d.SetTransform("not-allowed")
	})
	assert.Panics(t, func() {
		d.SetFlags(12)
	})

	d.ResetCursor()
	for i := 0; i < 1_000; i++ {
		kv, err := d.Next()
		assert.NoError(t, err)
		assert.Equal(t, byte(i), kv.Key[4])
		assert.Equal(t, []byte{'v', byte(i)}, kv.Value[:2])
		assert.Equal(t, uint32(i)%2, kv.Flags)
		assert.Equal(t, uint64(i), kv.TimestampNano)
	}
	_, err = d.Next()
	assert.Equal(t, io.EOF, err)
}

func BenchmarkDBI_Next(b *testing.B) {
	d := makeTestDBI(1_000_000)
	d.ResetCursor()

	var doNotOptimise int
	b.ReportAllocs()
	b.ResetTimer()
	t := time.Now()
	for i := 0; i < b.N; i++ {
		kv, err := d.Next()
		if err == io.EOF {
			d.ResetCursor()
		}
		doNotOptimise += len(kv.Value)
	}
	b.StopTimer()
	dt := time.Since(t)
	useVar(doNotOptimise)

	b.ReportMetric(float64(b.N)/dt.Seconds()/1_000_000, "Mentries/s")
}

func BenchmarkDBI_Next_compare_with_slice(b *testing.B) {
	slice := make([]KV, 1_000_000)

	var doNotOptimise int
	b.ReportAllocs()
	b.ResetTimer()
	t := time.Now()
	for i := 0; i < b.N; i++ {
		kv := slice[i%1_000_000]
		doNotOptimise += len(kv.Value)
	}
	b.StopTimer()
	dt := time.Since(t)
	useVar(doNotOptimise)

	b.ReportMetric(float64(b.N)/dt.Seconds()/1_000_000, "Mentries/s")
}

func BenchmarkDBI_index_1M(b *testing.B) {
	d := makeTestDBI(1_000_000)
	b.Logf("Size for 1M entries: %.1f MB", float64(len(d.Marshal())/MB))
	d.ResetCursor()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.indexData()
	}
}

func useVar(v any) {
	_, _ = fmt.Fprint(io.Discard, v)
}
