package snapshot

import (
	"io"

	"github.com/CrowdStrike/csproto"
)

// Protobuf field numbers
const (
	FieldSnapshotFormatVersion = 1
	FieldSnapshotMeta          = 2
	FieldSnapshotDBI           = 3
	FieldSnapshotCompatVersion = 4
)

// Snapshot is the root object in a snapshot protobuf
type Snapshot struct {
	FormatVersion uint32 // version of this snapshot format
	CompatVersion uint32 // compatible with clients that support at least this version
	Meta          Meta
	Databases     []*DBI
}

func (s *Snapshot) Unmarshal(data []byte) error {
	d := csproto.NewDecoder(data)
	d.SetMode(csproto.DecoderModeFast)
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

func (s *Snapshot) WriteTo(w io.Writer) (n int64, err error) {
	panic("NOT IMPLEMENTED")
}
