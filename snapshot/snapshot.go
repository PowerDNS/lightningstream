package snapshot

import (
	"io"

	"github.com/CrowdStrike/csproto"
	"github.com/c2h5oh/datasize"
)

// Protobuf field numbers
const (
	FieldSnapshotFormatVersion = 1
	FieldSnapshotMeta          = 2
	FieldSnapshotDBI           = 3
	FieldSnapshotCompatVersion = 4
)

// MaxFieldLength overrides the default 2GB limit on variable length protobuf
// fields to allow larger dumps with our current schema.
const (
	MaxFieldLength = 100 * uint64(datasize.GB)
)

// Snapshot is the root object in a snapshot protobuf
type Snapshot struct {
	FormatVersion uint32 // version of this snapshot format
	CompatVersion uint32 // compatible with clients that support at least this version
	Meta          Meta
	Databases     []*DBI `json:",omitempty"`
}

func (s *Snapshot) Unmarshal(data []byte) error {
	d := csproto.NewDecoder(data)
	d.SetMode(csproto.DecoderModeFast)
	d.SetMaxFieldLength(MaxFieldLength) // allow DBI dumps larger than 2GB
	for d.More() {
		tag, wireType, err := d.DecodeTag()
		if err != nil {
			return err
		}
		switch tag {
		case FieldSnapshotFormatVersion:
			s.FormatVersion, err = getUInt32(d, tag, wireType)
			if err != nil {
				return err
			}
		case FieldSnapshotCompatVersion:
			s.CompatVersion, err = getUInt32(d, tag, wireType)
			if err != nil {
				return err
			}
		case FieldSnapshotMeta:
			msg, err := getBytes(d, tag, wireType)
			if err != nil {
				return err
			}
			if err := s.Meta.Unmarshal(msg); err != nil {
				return err
			}
		case FieldSnapshotDBI:
			msg, err := getBytes(d, tag, wireType)
			if err != nil {
				return err
			}
			dbi, err := NewDBIFromData(msg)
			if err != nil {
				return err
			}
			s.Databases = append(s.Databases, dbi)
		default:
			if _, err := d.Skip(tag, wireType); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteTo writes all protobuf data to an io.Writer. It does not construct the
// whole protobuf message in the process, it simply streams the data.
func (s *Snapshot) WriteTo(w io.Writer) (nWritten int64, err error) {
	b := make([]byte, 1000) // temp buffer to construct tags
	offset := 0

	// Add top-level fields
	varintFields := []struct {
		tag int
		val uint64
	}{
		{FieldSnapshotFormatVersion, uint64(s.FormatVersion)},
		{FieldSnapshotCompatVersion, uint64(s.CompatVersion)},
	}
	for _, f := range varintFields {
		if f.val > 0 {
			offset += csproto.EncodeTag(b[offset:], f.tag, csproto.WireTypeVarint)
			offset += csproto.EncodeVarint(b[offset:], f.val)
		}
	}

	// Flush temp buffer
	n, err := w.Write(b[:offset])
	nWritten += int64(n)
	if err != nil {
		return nWritten, err
	}
	offset = 0
	_ = offset // silence linter

	// Add Meta
	metaPB := s.Meta.Marshal()
	if len(metaPB) > 0 {
		// Header with tag and length
		offset = 0
		offset += csproto.EncodeTag(b[offset:], FieldSnapshotMeta, csproto.WireTypeLengthDelimited)
		offset += csproto.EncodeVarint(b[offset:], uint64(len(metaPB)))
		// Flush temp buffer
		n, err := w.Write(b[:offset])
		nWritten += int64(n)
		if err != nil {
			return nWritten, err
		}
		offset = 0
		_ = offset // silence linter

		// Write actual Meta message
		n, err = w.Write(metaPB)
		nWritten += int64(n)
		if err != nil {
			return nWritten, err
		}
	}

	// Add DBIs
	for _, dbi := range s.Databases {
		// No actual work is done by this Marshal, it just returns its internal slice
		dbiPB := dbi.Marshal()
		if len(dbiPB) == 0 {
			continue
		}

		// Header with tag and length
		offset = 0
		offset += csproto.EncodeTag(b[offset:], FieldSnapshotDBI, csproto.WireTypeLengthDelimited)
		offset += csproto.EncodeVarint(b[offset:], uint64(len(dbiPB)))
		// Flush temp buffer
		n, err := w.Write(b[:offset])
		nWritten += int64(n)
		if err != nil {
			return nWritten, err
		}
		offset = 0
		_ = offset // silence linter

		// Write actual DBI message
		n, err = w.Write(dbiPB)
		nWritten += int64(n)
		if err != nil {
			return nWritten, err
		}
	}

	return nWritten, nil
}
