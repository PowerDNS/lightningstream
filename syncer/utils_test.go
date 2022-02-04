package syncer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_displayASCII1(t *testing.T) {
	tests := []struct {
		name string
		b    []byte
		want string
	}{
		{"empty", []byte{}, ""},
		{"nil", nil, ""},
		{"safe", []byte("abc"), "abc"},
		{"space", []byte("abc def"), "abc def"},
		{"newline", []byte("abc\ndef"), "abc.def [61 62 63 0a 64 65 66]"},
		{"control", []byte("\x01abc"), ".abc [01 61 62 63]"},
		{"zero", []byte("\x00abc"), ".abc [00 61 62 63]"},
		{"high", []byte("\xF0abc"), ".abc [f0 61 62 63]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, displayASCII(tt.b), "displayASCII(%v)", tt.b)
		})
	}
}
