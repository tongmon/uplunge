package app

import (
	"slices"
	"strings"
	"testing"
)

func TestParseShotTicks(t *testing.T) {
	tests := []struct {
		in      string
		want    []uint64
		wantErr string
	}{
		{"120", []uint64{120}, ""},
		{"0,30,120", []uint64{0, 30, 120}, ""},
		{"120, 0 ,30", []uint64{0, 30, 120}, ""},
		{"", nil, "bad shot tick"},
		{"30,,60", nil, "bad shot tick"},
		{"-1", nil, "bad shot tick"},
		{"1.5", nil, "bad shot tick"},
		{"30,30", nil, "listed twice"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseShotTicks(tt.in)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, %v; want %v", got, err, tt.want)
			}
		})
	}
}
