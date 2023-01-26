package syncer

import (
	"bytes"
	"fmt"

	"powerdns.com/platform/lightningstream/snapshot"
	"powerdns.com/platform/lightningstream/utils"
)

const (
	LMDBMaxKeySize        = 511
	DupSortHackMaxKeySize = 255
)

// dupSortHackEncodeOne encodes DupSort key-values in a way that hopefully makes
// the key unique.
// It appends to the original key:
// - a separator consisting on 0 bytes
// - the value, or part of it if the maximum key size would be exceeded
// - a byte with the original length of the key, up to 255
// The value remains unchanged.
// The original key size is limited to 255 bytes.
// Examples:
//
//	"key" = "val"  --->  "key\0\0\0\0val\x03" = "val"
//	"key" = "<very-long-val>"  --->  "key\0\0\0\0<503-bytes-that-fit>\x03" = "<very-long-val>"
func dupSortHackEncodeOne(e snapshot.KV) (result snapshot.KV, err error) {
	if len(e.Key) == 0 {
		return result, fmt.Errorf("empty key not supported by dupsort_hack")
	}
	if len(e.Key) > DupSortHackMaxKeySize {
		return result, fmt.Errorf(
			"key size exceeds dupsort_hack max size of %d: key %s",
			DupSortHackMaxKeySize,
			utils.DisplayASCII(e.Key),
		)
	}
	key := make([]byte, 0, LMDBMaxKeySize)
	key = append(key, e.Key...)
	key = append(key, 0, 0, 0, 0)              // separator
	remaining := LMDBMaxKeySize - len(key) - 1 // reserve last byte for length
	if len(e.Value) > remaining {
		key = append(key, e.Value[:remaining]...)
	} else {
		key = append(key, e.Value...)
	}
	key = append(key, uint8(len(e.Key))) // limits DupSortHackMaxKeySize
	result.Key = key
	result.Value = e.Value
	result.Flags = e.Flags
	return result, nil
}

// dupSortHackDecodeOne does the opposite of dupSortHackEncodeOne
func dupSortHackDecodeOne(e snapshot.KV) (result snapshot.KV, err error) {
	if len(e.Key) < 6 { // 4 for separator + 1 minimum key size + key length byte
		return result, fmt.Errorf("not a valid dupsort_hack dump (key)")
	}
	keyLen := int(e.Key[len(e.Key)-1]) // last byte
	if len(e.Key) < keyLen+5 {         // 4 for separator, 1 for length byte
		return result, fmt.Errorf("not a valid dupsort_hack dump (key len)")
	}
	if e.Key[keyLen] != 0 || e.Key[keyLen+1] != 0 || e.Key[keyLen+2] != 0 || e.Key[keyLen+3] != 0 {
		return result, fmt.Errorf("not a valid dupsort_hack dump (no separator)")
	}
	key := e.Key[:keyLen]
	result.Key = key
	result.Value = e.Value
	result.Flags = e.Flags
	return result, nil
}

// dupSortHackEncode changes a slice of entries in-place with dupSortHackEncodeOne
func dupSortHackEncode(entries []snapshot.KV) error {
	var prevKey []byte
	for i, e := range entries {
		r, err := dupSortHackEncodeOne(e)
		if err != nil {
			return err
		}
		cmp := bytes.Compare(prevKey, r.Key)
		if cmp == 0 {
			// Can happen if the value difference is in the second part
			return fmt.Errorf(
				"dupsort_hack does not result in unique keys for key %s",
				utils.DisplayASCII(e.Key))
		}
		if cmp > 0 {
			// Can only happen if the separator can be found in keys and beginning
			// of values.
			// The separator itself was chosen a zeros to not affect ordering.
			return fmt.Errorf(
				"dupsort_hack results in reverse sort order for key %s",
				utils.DisplayASCII(e.Key))
		}
		prevKey = r.Key
		entries[i] = r
	}
	return nil
}

// dupSortHackDecode changes a slice of entries in-place with dupSortHackDecodeOne
func dupSortHackDecode(entries []snapshot.KV) error {
	for i, e := range entries {
		r, err := dupSortHackDecodeOne(e)
		if err != nil {
			return err
		}
		entries[i] = r
	}
	return nil
}
