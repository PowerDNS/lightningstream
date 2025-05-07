package snapshot

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/CrowdStrike/csproto"
)

// Protobuf field numbers
const (
	FieldDBIName      = 1
	FieldDBIEntries   = 2
	FieldDBIFlags     = 3
	FieldDBITransform = 4

	// TagSize0To15 is the number of bytes taken by a key with tag 1-15
	TagSize0To15 = 1
)

// NewDBI creates a new empty DBI
func NewDBI() *DBI {
	return &DBI{}
}

// NewDBISize creates a new empty DBI and pre-allocates memory for the protobuf
// data to avoid future reallocs. The size is given in bytes.
func NewDBISize(size int) *DBI {
	return &DBI{
		data: make([]byte, 0, size),
	}
}

// NewDBIFromData creates a new DBI from protobuf data
func NewDBIFromData(data []byte) (*DBI, error) {
	d := DBI{
		data:    data,
		flushed: true, // do not allow setting top-level fields
	}
	if err := d.indexData(); err != nil {
		return nil, err
	}
	return &d, nil
}

// DBI describes the contents of a single DBI.
// The top-level fields (name, flags and transform) can only be set
// before any KV data is written with Append(KV).
// If loaded from an existing protobuf, the top-level fields are read-only.
type DBI struct {
	// Accessed through methods
	name      string
	flags     uint64
	transform string

	// data contains the written protobuf data. DBI only ever appends to this.
	data []byte

	// To keep track of the write state
	dirty   bool // non-KV fields have been modified
	flushed bool // indicates that data have already been written

	cur int // current read offset

	// Some statistics for logging (not persisted)
	NumWrittenEntries int64 // Only incremented when writing
}

func (d *DBI) Name() string {
	return d.name
}

func (d *DBI) Flags() uint64 {
	return d.flags
}

func (d *DBI) Transform() string {
	return d.transform
}

func (d *DBI) SetName(s string) {
	if d.flushed {
		panic("not allowed after fields have been flushed")
	}
	d.name = s
	d.dirty = true
}

func (d *DBI) SetFlags(v uint64) {
	if d.flushed {
		panic("not allowed after fields have been flushed")
	}
	d.flags = v
	d.dirty = true
}

func (d *DBI) SetTransform(s string) {
	if d.flushed {
		panic("not allowed after fields have been flushed")
	}
	d.transform = s
	d.dirty = true
}

func (d *DBI) flushFields() {
	d.flushed = true
	if !d.dirty {
		return
	}
	d.dirty = false
	d.doFlushFields()
}

func (d *DBI) doFlushFields() {
	// This is large enough for our basic fields
	// - name will not be longer than 511 bytes (LMDB key size)
	// - transform is something we set, should never be larger than 50 chars
	// - flags is a varint
	// - a few extra bytes for tag and length varints
	b := make([]byte, 1000)
	offset := 0

	if len(d.name) > 0 {
		offset += csproto.EncodeTag(b[offset:], FieldDBIName, csproto.WireTypeLengthDelimited)
		offset += csproto.EncodeVarint(b[offset:], uint64(len(d.name)))
		offset += copy(b[offset:], d.name)
	}
	if d.flags > 0 {
		offset += csproto.EncodeTag(b[offset:], FieldDBIFlags, csproto.WireTypeVarint)
		offset += csproto.EncodeVarint(b[offset:], d.flags)
	}
	if len(d.transform) > 0 {
		offset += csproto.EncodeTag(b[offset:], FieldDBITransform, csproto.WireTypeLengthDelimited)
		offset += csproto.EncodeVarint(b[offset:], uint64(len(d.transform)))
		offset += copy(b[offset:], d.transform)
	}

	d.data = append(d.data, b[:offset]...)
}

// ResetCursor resets the read cursor to the beginning of the buffer
func (d *DBI) ResetCursor() {
	d.cur = 0
}

// Marshal returns the currently written protobuf data.
// This implicitly calls flushFields, which will prevent further changes to
// the top-level fields.
// Careful, this does not make a copy.
func (d *DBI) Marshal() []byte {
	d.flushFields()
	l := len(d.data)
	return d.data[:l:l]
}

// Size returns the size of the protobuf message
// This implicitly calls flushFields, which will prevent further changes to
// the top-level fields.
func (d *DBI) Size() (n int) {
	d.flushFields()
	return len(d.data)
}

// Next decodes the next KV from the data.
// BenchmarkDBI_Next benchmarks this, locally it takes about 40ns per entry
// (or 40ms for 1 million entries), which makes it a fine replacement for
// looping over a slice, given that we avoid the allocations.
func (d *DBI) Next() (kv KV, err error) {
	// Only update d.cur at the end to never point into the middle of a message
	// if something went wrong before.
	offset := d.cur

	var tag int
	var wireType csproto.WireType
	for tag != FieldDBIEntries {
		if offset >= len(d.data) {
			return kv, io.EOF
		}

		// Get the tag and type
		v, n, err := csproto.DecodeVarint(d.data[offset:])
		if err != nil {
			return kv, err
		}
		offset += n
		tag = int(v >> 3)
		wireType = csproto.WireType(v & 0x7)

		if tag != FieldDBIEntries {
			// Skip the tag if not the one we want
			n, err := skipTag(d.data[offset:], wireType)
			if err != nil {
				return kv, err
			}
			offset += n
		}
	}

	// Check wire type
	if err := expectWT(tag, wireType, csproto.WireTypeLengthDelimited); err != nil {
		return kv, err
	}
	// Get the length
	v, n, err := csproto.DecodeVarint(d.data[offset:])
	size := int(v)
	if err != nil {
		return kv, err
	}
	offset += n
	if len(d.data)-offset < size {
		return kv, fmt.Errorf("remaining data to short for indicated size")
	}

	// Unmarshal the data
	b := d.data[offset : offset+size : offset+size]
	offset += size
	d.cur = offset
	err = kv.Unmarshal(b)
	return kv, err
}

// indexData finds all the DBI fields, except for the KV entries
func (d *DBI) indexData() error {
	var offset = 0
	data := d.data
	d.flushed = true // we do not allow changing top-level keys once this is called

	for {
		if offset >= len(data) {
			return nil
		}

		// Get the tag and type
		v, n, err := csproto.DecodeVarint(data[offset:])
		if err != nil {
			return err
		}
		offset += n
		tag := int(v >> 3)
		wireType := csproto.WireType(v & 0x7)

		switch tag {
		case FieldDBIEntries, FieldDBIName, FieldDBITransform:
			// Length delimited fields
			// Check wire type
			if err := expectWT(tag, wireType, csproto.WireTypeLengthDelimited); err != nil {
				return err
			}
			// Get the length
			v, n, err := csproto.DecodeVarint(data[offset:])
			size := int(v)
			if err != nil {
				return err
			}
			offset += n

			// Actual data
			if len(data)-offset < size {
				return fmt.Errorf("remaining data to short for indicated size")
			}
			b := data[offset : offset+size : offset+size]
			switch tag {
			case FieldDBIEntries:
				// ignore
			case FieldDBIName:
				d.name = string(b)
			case FieldDBITransform:
				d.transform = string(b)
			default:
				panic("unhandled tag")
			}
			offset += size
		case FieldDBIFlags:
			// Check wire type
			if err := expectWT(tag, wireType, csproto.WireTypeVarint); err != nil {
				return err
			}
			// Get the length
			v, n, err := csproto.DecodeVarint(data[offset:])
			if err != nil {
				return err
			}
			offset += n
			d.flags = v
		default:
			n, err := skipTag(data, wireType)
			if err != nil {
				return err
			}
			offset += n
		}
	}
}

// Append appends a new KV to the DBI protobuf.
// The data that KV.Key and KV.Value refer to is copied in the process, so it
// is also safe when they point directly into LMDB pages.
func (d *DBI) Append(kv KV) {
	// FLush top level fields if needed
	if d.dirty {
		d.flushFields()
	}

	d.NumWrittenEntries++

	// Start writing here
	offset := len(d.data)

	// Determine the size of the KV message
	var msgSize = 0
	if len(kv.Key) > 0 {
		msgSize += TagSize0To15
		msgSize += csproto.SizeOfVarint(uint64(len(kv.Key)))
		msgSize += len(kv.Key)
	}
	if len(kv.Value) > 0 {
		msgSize += TagSize0To15
		msgSize += csproto.SizeOfVarint(uint64(len(kv.Value)))
		msgSize += len(kv.Value)
	}
	if kv.Flags > 0 {
		msgSize += TagSize0To15
		msgSize += csproto.SizeOfVarint(uint64(kv.Flags))
	}
	if kv.TimestampNano > 0 {
		msgSize += TagSize0To15 + 8 // fixed
	}
	if msgSize == 0 {
		return // do not write empty messages
	}
	// Size after wrapping as length delimited data
	outerSize := TagSize0To15 + csproto.SizeOfVarint(uint64(msgSize)) + msgSize

	// Grow data buffer if needed
	// TODO: Perhaps use buckets to prevent copying?
	if (cap(d.data) - len(d.data)) < outerSize {
		oldSize := cap(d.data)
		newSize := 2 * oldSize
		if oldSize < 5*MB {
			newSize = 10 * MB
		}
		if oldSize > 1024*MB {
			newSize = oldSize + 512*MB
		}
		newData := make([]byte, len(d.data), newSize+outerSize)
		copy(newData, d.data)
		d.data = newData
	}
	// Expand the data slide to make room for new message
	d.data = d.data[:len(d.data)+outerSize]

	// First write an DBI.Entries tag header and size
	offset += csproto.EncodeTag(d.data[offset:], FieldDBIEntries, csproto.WireTypeLengthDelimited)
	offset += csproto.EncodeVarint(d.data[offset:], uint64(msgSize))

	// Then write the actual KV fields
	if len(kv.Key) > 0 {
		offset += csproto.EncodeTag(d.data[offset:], FieldKVKey, csproto.WireTypeLengthDelimited)
		offset += csproto.EncodeVarint(d.data[offset:], uint64(len(kv.Key)))
		offset += copy(d.data[offset:], kv.Key)
	}
	if len(kv.Value) > 0 {
		offset += csproto.EncodeTag(d.data[offset:], FieldKVValue, csproto.WireTypeLengthDelimited)
		offset += csproto.EncodeVarint(d.data[offset:], uint64(len(kv.Value)))
		offset += copy(d.data[offset:], kv.Value)
	}
	if kv.Flags > 0 {
		offset += csproto.EncodeTag(d.data[offset:], FieldKVFlags, csproto.WireTypeVarint)
		offset += csproto.EncodeVarint(d.data[offset:], uint64(kv.Flags))
	}
	if kv.TimestampNano > 0 {
		offset += csproto.EncodeTag(d.data[offset:], FieldKVTimestampNano, csproto.WireTypeFixed64)
		binary.LittleEndian.PutUint64(d.data[offset:offset+8], kv.TimestampNano)
		offset += 8
	}
	_ = offset // silence linter
}

type KVMapFunc = func(KV) (KV, error)

// Map creates a new DBI with copied and transformed data.
func (d *DBI) Map(transform string, f KVMapFunc) (*DBI, error) {
	newDBI := NewDBISize(len(d.data))
	newDBI.SetName(d.name)
	newDBI.SetFlags(d.flags)
	newDBI.SetTransform(transform)
	var err error
	var kv KV
	d.ResetCursor()
	for {
		kv, err = d.Next()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		newKV, err := f(kv)
		if err != nil {
			return nil, err
		}
		newDBI.Append(newKV)
	}
	return newDBI, nil
}

// AsInefficientKVList returns all KV entries as an inefficient []KV.
// Only use this for tests.
func (d *DBI) AsInefficientKVList() ([]KV, error) {
	var kvList []KV
	d.ResetCursor()
	for {
		kv, err := d.Next()
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		kvList = append(kvList, kv)
	}
	return kvList, nil
}
