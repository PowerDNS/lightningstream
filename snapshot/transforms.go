package snapshot

import (
	"fmt"

	"github.com/PowerDNS/lmdb-go/lmdb"
)

const (
	// TransformDupSortHackV1 is the 'transform' field for the current
	// dupsort_hack key-value transformation.
	TransformDupSortHackV1 = "dupsort_hack_v1"
	// TransformNone indicates no transformation
	TransformNone = ""
)

// TransformSupported checks if a given transform is supported.
func TransformSupported(transform string) bool {
	switch transform {
	case TransformNone:
		return true
	case TransformDupSortHackV1:
		return true
	default:
		return false
	}
}

// ValidateTransform checks if the transform field is set to a supported value.
func (d *DBI) ValidateTransform(formatVersion uint32, nativeSchema bool) error {
	dbiName := d.Name()
	flags := uint(d.Flags())
	transform := d.Transform()

	if !TransformSupported(transform) {
		return fmt.Errorf("snapshot dbi %q: transform %q not supported",
			dbiName, transform)
	}
	if nativeSchema && transform != TransformNone {
		return fmt.Errorf("snapshot dbi %q: no transforms supported "+
			"for native schema, got %q", dbiName, transform)
	}
	// First formatVersion that has the transform field
	if formatVersion >= 3 {
		flagsDupSort := flags&lmdb.DupSort > 0
		transformDupSort := transform == TransformDupSortHackV1
		if flagsDupSort && !transformDupSort {
			return fmt.Errorf("snapshot dbi %q: dupsort DBI flag without "+
				"expected transform (got %q, expected %q)",
				dbiName, transform, TransformDupSortHackV1)
		}
		if !flagsDupSort && transformDupSort {
			return fmt.Errorf("snapshot dbi %q: non-dupsort DBI flags with "+
				"unexpected dupsort transform (got %q)",
				dbiName, transform)
		}
	}
	return nil
}
