package snapshot

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeTestSnapshot(dbiEntries int) *Snapshot {
	testDBI := makeTestDBI(dbiEntries)
	return &Snapshot{
		FormatVersion: 1,
		CompatVersion: 2,
		Meta:          makeTestMeta(),
		Databases: []*DBI{
			testDBI,
		},
	}
}

func TestSnapshot_WriteTo(t *testing.T) {
	snap := makeTestSnapshot(10_000)
	if size := snap.Databases[0].Size(); size < 50_000 {
		t.Errorf("expected DBI size at least 50kB, got %d", size)
	}

	buf := bytes.NewBuffer(nil)
	n, err := snap.WriteTo(buf)
	assert.NoError(t, err)
	if buf.Len() != int(n) {
		t.Errorf("expected protobuf size to equal bytes written, but %d != %d", buf.Len(), n)
	}
	if n < 50_000 {
		t.Errorf("expected bytes written of at least 50kB, got %d", n)
	}
}

func TestSnapshot_WriteTo_Unmarshal_roundtrip(t *testing.T) {
	origSnap := makeTestSnapshot(10_000)
	buf := bytes.NewBuffer(nil)
	_, err := origSnap.WriteTo(buf)
	assert.NoError(t, err)

	snap := new(Snapshot)
	err = snap.Unmarshal(buf.Bytes())
	assert.NoError(t, err)

	assert.Equal(t, origSnap.FormatVersion, snap.FormatVersion)
	assert.Equal(t, origSnap.CompatVersion, snap.CompatVersion)
	assert.Equal(t, origSnap.Meta, snap.Meta)
	assert.Equal(t, 1, len(snap.Databases))
	assert.Equal(t, origSnap.Databases[0].Marshal(), snap.Databases[0].Marshal())
}

func BenchmarkSnapshot_WriteTo(b *testing.B) {
	// Since we write to io.Discard and DBIs already contain the data ready as
	// a slice, the write speed here is ridiculously high. The number of
	// entries does not matter at all for this number, as we are just passing
	// a slice around.
	snap := makeTestSnapshot(10_000)
	var bytesWritten int64
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n, err := snap.WriteTo(io.Discard)
		assert.NoError(b, err)
		bytesWritten += n
	}
	b.StopTimer()
	useVar(bytesWritten)
}
