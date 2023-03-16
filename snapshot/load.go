package snapshot

import (
	"bytes"
	"io"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/klauspost/compress/gzip"
)

// LoadData loads snapshot file contents that are gzipped protobufs
func LoadData(data []byte) (*Snapshot, error) {
	// Uncompress
	dataReader := bytes.NewReader(data)
	g, err := gzip.NewReader(dataReader)
	if err != nil {
		return nil, err
	}
	// For buffer sizing, assume 1:10 best case compression. Better to overestimate
	// than to underestimate the size needed, because reallocs are expensive.
	pbBuf := bytes.NewBuffer(make([]byte, 0, 10*len(data)))
	//pbData, err := io.ReadAll(g)
	_, err = io.Copy(pbBuf, g)
	if err != nil {
		return nil, err
	}
	pbData := pbBuf.Bytes()
	if err := g.Close(); err != nil {
		return nil, err
	}

	// Load protobuf
	msg := new(Snapshot)
	if err := msg.Unmarshal(pbData); err != nil {
		return nil, err
	}

	return msg, nil
}

// DumpData returns a compressed Snapshot.
func DumpData(msg *Snapshot) ([]byte, DumpDataStats, error) {
	var stat DumpDataStats
	t0 := time.Now()

	// Streaming compression
	// For buffer sizing, assume 1:2 worst case compression. Better to overestimate
	// than to underestimate the size needed, because reallocs are expensive.
	var estimatedSize int
	for _, d := range msg.Databases {
		estimatedSize += d.Size()
	}
	out := bytes.NewBuffer(make([]byte, 0, estimatedSize/2))
	gw, err := gzip.NewWriterLevel(out, gzip.BestSpeed)
	if err != nil {
		return nil, stat, err
	}

	// Marshal and write to gzip writer
	// The marshalling itself takes almost no time, since all the DBI data is
	// already marshaled.
	pbSize, err := msg.WriteTo(gw)
	if err != nil {
		return nil, stat, err
	}
	stat.ProtobufSize = datasize.ByteSize(pbSize)

	if err = gw.Close(); err != nil {
		return nil, stat, err
	}
	tCompressed := time.Now()
	stat.TCompressed = tCompressed.Sub(t0)

	compressedData := out.Bytes()
	stat.CompressedSize = datasize.ByteSize(len(compressedData))
	return compressedData, stat, nil
}

type DumpDataStats struct {
	TCompressed    time.Duration     // time it took to marshal (near 0) and compress
	ProtobufSize   datasize.ByteSize // uncompressed protobuf size
	CompressedSize datasize.ByteSize // uncompressed protobuf size
}
