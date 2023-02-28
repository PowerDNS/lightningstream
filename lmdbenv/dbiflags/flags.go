package dbiflags

import (
	"fmt"
	"strconv"
	"strings"
)

// Flags is bitmask of LMDB DBI flags.
// It implements the TextMarshaler and TextUnmarshaler interfaces and supports
// '|'-separated names of LMDB C constants (e.g. MDB_DUPSORT), so that it can
// be conveniently used in config files.
type Flags uint16

const (
	ReverseKey Flags = 0x02
	DupSort    Flags = 0x04
	IntegerKey Flags = 0x08
	DupFixed   Flags = 0x10
	IntegerDup Flags = 0x20
	ReverseDup Flags = 0x40
)

func (f Flags) String() string {
	var sb strings.Builder
	for b := 0x02; b <= 0x40; b <<= 1 {
		bf := Flags(b)
		if f&bf == 0 {
			continue
		}
		f ^= bf // clear flag from local copy
		if sb.Len() > 0 {
			sb.WriteByte('|')
		}
		sb.WriteString(flagToName[bf])
	}
	if f != 0 {
		// Unknown leftover flags
		if sb.Len() > 0 {
			sb.WriteByte('|')
		}
		sb.WriteString("UNKNOWN:0x")
		sb.WriteString(strconv.FormatUint(uint64(f), 16))
	}
	return sb.String()
}

func (f Flags) MarshalText() (text []byte, err error) {
	return []byte(f.String()), nil
}

func (f *Flags) UnmarshalText(text []byte) error {
	*f = 0 // reset
	s := string(text)

	// Accept additional separator characters, in addition to '|'.
	s = strings.ReplaceAll(s, ",", "|")
	s = strings.ReplaceAll(s, "+", "|")
	s = strings.ReplaceAll(s, " ", "|")
	parts := strings.Split(s, "|")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.ToUpper(p)
		if len(p) == 0 {
			continue
		}
		if flag, ok := nameToFlag[p]; ok {
			*f |= flag
			continue
		}
		// Also accept the flags without "MDB_" prefix
		if flag, ok := nameToFlag["MDB_"+p]; ok {
			*f |= flag
			continue
		}
		// Allow numbers as flags. Parse as 16 bits to avoid MDB_CREATE
		// and other non-persistent flags.
		if strings.HasPrefix(p, "0X") {
			v, err := strconv.ParseUint(p[2:], 16, 16)
			if err != nil {
				return fmt.Errorf("invalid persistent DBI flag: %s", p)
			}
			vf := Flags(v)
			if vf&allValid != vf {
				return fmt.Errorf("invalid persistent DBI flag: %s", p)
			}
			*f |= vf
			continue
		}
		v, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid persistent DBI flag: %s", p)
		}
		vf := Flags(v)
		if vf&allValid != vf {
			return fmt.Errorf("invalid persistent DBI flag: %s", p)
		}
		*f |= vf
	}

	return nil
}

// nameToFlag maps LMDB DBI flag constant names (C names, like "MDB_DUPSORT") to
// Flags values.
// MDB_CREATE is omitted here, as it is not a persistent DBI flag.
var nameToFlag = map[string]Flags{
	"MDB_REVERSEKEY": ReverseKey,
	"MDB_DUPSORT":    DupSort,
	"MDB_INTEGERKEY": IntegerKey,
	"MDB_DUPFIXED":   DupFixed,
	"MDB_INTEGERDUP": IntegerDup,
	"MDB_REVERSEDUP": ReverseDup,
}

// flagToName is the inverse of nameToFLag
var flagToName map[Flags]string

var allValid Flags

func init() {
	flagToName = make(map[Flags]string)
	for name, flag := range nameToFlag {
		flagToName[flag] = name
		allValid |= flag
	}
}
