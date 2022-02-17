package strategy

import "testing"

func Test_cmpIntegerLittleEndian(t *testing.T) {
	tests := []struct {
		name string
		a    []byte
		b    []byte
		want int
	}{
		{"same", []byte{1, 1}, []byte{1, 1}, 0},             // 0x0101 == 0x0101
		{"gt", []byte{1, 2}, []byte{2, 1}, 1},               // 0x0201 > 0x0102
		{"lt", []byte{2, 1}, []byte{1, 2}, -1},              // 0x0102 < 0x0201
		{"one-longer", []byte{0, 0, 2, 1}, []byte{1, 2}, 1}, // 0x01020000 > 0x0201
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cmpIntegerLittleEndian(tt.a, tt.b); got != tt.want {
				t.Errorf("cmpIntegerLittleEndian() = %v, want %v", got, tt.want)
			}
		})
	}
}
