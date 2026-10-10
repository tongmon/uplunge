package app

import (
	"testing"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

func TestSpawnPoint(t *testing.T) {
	p := tuning.Player{Width: 12, Height: 20}
	tests := []struct {
		name         string
		rows         []string
		wantX, wantY int
		wantErr      bool
	}{
		{"open top", []string{"#...........#", "#...........#", "#############"}, 98, 0, false},
		{"solid at top centre", []string{"#.....#.....#", "#...........#", "#############"}, 0, 0, true},
		{"too short for the player", []string{"#...........#"}, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := level.ParseRows(16, tt.rows...)
			if err != nil {
				t.Fatal(err)
			}
			x, y, err := spawnPoint(m, p)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("got (%d, %d), want an error", x, y)
				}
				return
			}
			if err != nil || x != tt.wantX || y != tt.wantY {
				t.Fatalf("got (%d, %d) err=%v, want (%d, %d)", x, y, err, tt.wantX, tt.wantY)
			}
		})
	}
}
