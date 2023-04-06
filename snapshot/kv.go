package snapshot

import (
	"encoding/binary"
	"fmt"

	"github.com/CrowdStrike/csproto"
)

// Protobuf field numbers
const (
	FieldKVKey           = 1
	FieldKVValue         = 2
	FieldKVTimestampNano = 3
	FieldKVFlags         = 4
)

type KV struct {
	Key           []byte
	Value         []byte
	TimestampNano uint64
	Flags         uint32
}

func (kv *KV) Unmarshal(data []byte) error {
	// Special purpose parsing code for speed (no pointer allocs with NewDecoder)
	offset := 0
	dataSize := len(data)
	for {
		// Get the tag and type
		v, n, err := csproto.DecodeVarint(data[offset:])
		if err != nil {
			return err
		}
		offset += n
		tag := int(v >> 3)
		wireType := csproto.WireType(v & 0x7)

		// Get the data
		switch tag {
		case FieldKVKey, FieldKVValue:
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
			if dataSize-offset < size {
				return fmt.Errorf("remaining data to short for indicated size")
			}
			b := data[offset : offset+size : offset+size]
			offset += size
			if tag == FieldKVKey {
				kv.Key = b
			} else {
				kv.Value = b
			}
		case FieldKVFlags:
			if err := expectWT(tag, wireType, csproto.WireTypeVarint); err != nil {
				return err
			}
			// Get the varint
			v, n, err := csproto.DecodeVarint(data[offset:])
			if err != nil {
				return err
			}
			offset += n
			kv.Flags = uint32(v)
		case FieldKVTimestampNano:
			if err := expectWT(tag, wireType, csproto.WireTypeFixed64); err != nil {
				return err
			}
			if dataSize-offset < 8 {
				return fmt.Errorf("remaining data to short for fixed64")
			}
			b := data[offset : offset+8 : offset+8]
			offset += 8
			kv.TimestampNano = binary.LittleEndian.Uint64(b)
		default:
			n, err := skipTag(data[offset:], wireType)
			if err != nil {
				return err
			}
			offset += n
		}

		if offset == dataSize {
			break
		}
	}
	return nil
}
