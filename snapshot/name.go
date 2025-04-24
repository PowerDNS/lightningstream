package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/PowerDNS/lightningstream/lmdbenv/header"
)

const (
	timeFormat = "20060102-150405.000000000" // but need to s/./-/
	dotIndex   = 15                          // position of the '.'
)

// NameTimestamp convert a time.Time to a string for embedding in a filename
func NameTimestamp(ts time.Time) string {
	fileTimestamp := strings.Replace(
		ts.UTC().Format(timeFormat),
		".", "-", 1)
	return fileTimestamp
}

// NameTimestampFromNano is NameTimestamp for LS header timestamps
func NameTimestampFromNano(tsNano header.Timestamp) string {
	return NameTimestamp(tsNano.Time())
}

// Name constructs a snapshot name
func Name(syncerName, instanceID, generationID string, ts time.Time) string {
	ni := NameInfo{
		Extension:    DefaultExtension,
		SyncerName:   syncerName,
		InstanceID:   instanceID,
		GenerationID: generationID,
		Timestamp:    ts,
	}
	return ni.BuildName()
}

var (
	// registeredExtensions maps extensions to kind names
	registeredExtensions = map[string]string{}
)

// RegisterExtension registers a valid snapshot file extension with a kind name
func RegisterExtension(extension, kind string) {
	registeredExtensions[extension] = kind
}

const KindSnapshot = "snapshot"

const DefaultExtension = "pb.gz"

func init() {
	RegisterExtension(DefaultExtension, KindSnapshot)
}

// ParseName parses a snapshot filename
func ParseName(name string) (NameInfo, error) {
	var ni, empty NameInfo
	basename, ext, found := strings.Cut(name, ".")
	if !found {
		return empty, fmt.Errorf("invalid name: no dot: %s", name)
	}
	ni.FullName = name
	kind, known := registeredExtensions[ext]
	if !known {
		return empty, fmt.Errorf("unknown extension: %s", name)
	}
	ni.Extension = ext
	ni.Kind = kind
	p := strings.Split(basename, "__")
	if len(p) < 4 {
		return empty, fmt.Errorf("not enough name parts: %s", name)
	}
	ni.SyncerName = p[0]
	ni.InstanceID = p[1]
	ni.TimestampString = p[2]
	ni.GenerationID = p[3]
	for _, extra := range p[4:] {
		ni.Extra = append(ni.Extra, NameExtraItem(extra))
	}
	tss := ni.TimestampString
	if len(tss) != len(timeFormat) || tss[dotIndex] != '-' {
		return empty, fmt.Errorf("invalid timestamp format: %s in %s", tss, name)
	}
	tss = tss[:dotIndex] + "." + tss[dotIndex+1:] // replace second '-' with '.' for parsing
	ts, err := time.Parse(timeFormat, tss)        // returns time in UTC
	if err != nil {
		return empty, fmt.Errorf("timestamp parse error: %s", err)
	}
	ni.Timestamp = ts
	return ni, nil
}

// NameInfo breaks out all information encoded in a snapshot filename
type NameInfo struct {
	FullName        string    // Full filename
	BaseName        string    // Part before the Extension
	Extension       string    // File extension
	Kind            string    // Kind of file based on extension
	SyncerName      string    // Corresponds to the database being synced
	InstanceID      string    // ID of the LS instance that generated it
	GenerationID    string    // Currently unused, for old idea that was abandoned
	TimestampString string    // Timestamp string in filename
	Timestamp       time.Time // Nanosecond precision snapshot timestamp
	Extra           NameExtra // LSE: extra values after the GenerationID
}

// ShortHash returns a short hash of name info to visually distinguish snapshots in logs
func (ni NameInfo) ShortHash() string {
	return ShortHash(ni.InstanceID, ni.TimestampString)
}

// BuildName creates a filename from basic info
func (ni NameInfo) BuildName() string {
	var nb strings.Builder
	nb.WriteString(ni.SyncerName)
	nb.WriteString("__")
	nb.WriteString(ni.InstanceID)
	nb.WriteString("__")
	nb.WriteString(NameTimestamp(ni.Timestamp))
	nb.WriteString("__")
	nb.WriteString(ni.GenerationID)
	for _, extraItem := range ni.Extra {
		nb.WriteString("__")
		nb.WriteString(extraItem.String())
	}
	nb.WriteString(".")
	nb.WriteString(ni.Extension)
	return nb.String()
}

// NameExtra are extra values added to the filename after the GenerationID field.
type NameExtra []NameExtraItem

// Get retrieved on value by type, if it exists.
func (ne NameExtra) Get(extraType byte) (val string, ok bool) {
	for _, nei := range ne {
		if nei.Type() == extraType {
			return nei.Value(), true
		}
	}
	return "", false
}

func (ne NameExtra) Len() int {
	return len(ne)
}

func (ne NameExtra) Less(i, j int) bool {
	return ne[i].Type() < ne[j].Type()
}

func (ne NameExtra) Swap(i, j int) {
	ne[i], ne[j] = ne[j], ne[i]
}

// NameExtraItem represents one NameExtra value, e.g. "X1234".
//
// Requirements for these values:
//
//   - Start with a unique capital ascii letter [A-Z] indicating the type
//   - 'G' is reserved to prevent confusion with the GenerationID.
//   - Followed by zero or more string characters for the value.
//   - These items are separated by "__" in the filename, so they cannot contain
//     this substring.
//   - A type cannot appear more than once.
//   - The items SHOULD appear sorted alphabetically.
type NameExtraItem string

// Type returns the type byte (first letter)
func (nei NameExtraItem) Type() byte {
	return nei[0]
}

// Value returns the value part (after the first letter)
func (nei NameExtraItem) Value() string {
	return string(nei[1:])
}

// String returns the whole value as is.
func (nei NameExtraItem) String() string {
	return string(nei)
}

// ShortHash returns a short hash of name info to visually distinguish snapshots in logs
func ShortHash(instance, timestamp string) string {
	s := sha256.New()
	_, _ = fmt.Fprintf(s, "%s-%s", instance, timestamp)
	return hex.EncodeToString(s.Sum(nil))[:7]
}
