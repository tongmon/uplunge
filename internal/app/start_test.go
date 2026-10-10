package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tongmon/uplunge/internal/replay"
	"github.com/tongmon/uplunge/internal/sim"
)

func TestStartOf(t *testing.T) {
	dir := t.TempDir()
	towerRpl := filepath.Join(dir, "tower.rpl")
	chunkRpl := filepath.Join(dir, "chunk.rpl")
	in := []sim.Input{{Right: true}}
	if err := replay.Save(towerRpl, replay.Replay{Tower: true, Seed: 9, Inputs: in}); err != nil {
		t.Fatal(err)
	}
	if err := replay.Save(chunkRpl, replay.Replay{Chunk: "Start", Inputs: in}); err != nil {
		t.Fatal(err)
	}
	labRpl := filepath.Join(dir, "lab.rpl")
	if err := replay.Save(labRpl, replay.Replay{Lab: true, Seed: 5, Inputs: in}); err != nil {
		t.Fatal(err)
	}
	seed := func(v uint64) *uint64 { return &v }

	tests := []struct {
		name    string
		cfg     Config
		want    string // replay.Replay.Where, or "" for any new tower
		wantErr string
	}{
		{"chunk", Config{Chunk: "Blocks"}, `chunk "Blocks"`, ""},
		{"seed", Config{Seed: seed(3)}, "tower 3", ""},
		{"new tower", Config{}, "", ""},
		{"chunk and seed", Config{Chunk: "Blocks", Seed: seed(3)}, "", "takes no -seed or -lab"},
		{"chunk and lab", Config{Chunk: "Blocks", Lab: true}, "", "takes no -seed or -lab"},
		{"lab with seed", Config{Lab: true, Seed: seed(4)}, "lab 4", ""},
		{"tower replay, lab", Config{ReplayPath: towerRpl, Lab: true}, "", "-lab does not match"},
		{"lab replay", Config{ReplayPath: labRpl}, "lab 5", ""},
		{"lab replay, lab", Config{ReplayPath: labRpl, Lab: true}, "lab 5", ""},
		{"tower replay", Config{ReplayPath: towerRpl}, "tower 9", ""},
		{"tower replay, same seed", Config{ReplayPath: towerRpl, Seed: seed(9)}, "tower 9", ""},
		{"tower replay, other seed", Config{ReplayPath: towerRpl, Seed: seed(8)}, "", "-seed 8 does not match"},
		{"tower replay, chunk", Config{ReplayPath: towerRpl, Chunk: "Start"}, "", `-chunk "Start" does not match`},
		{"chunk replay", Config{ReplayPath: chunkRpl}, `chunk "Start"`, ""},
		{"chunk replay, seed", Config{ReplayPath: chunkRpl, Seed: seed(9)}, "", "-seed 9 does not match"},
		{"chunk replay, other chunk", Config{ReplayPath: chunkRpl, Chunk: "Blocks"}, "", `-chunk "Blocks" does not match`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := startOf(tt.cfg)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == "" {
				if !got.Tower {
					t.Fatalf("got %s, want a new tower", got.Where())
				}
				return
			}
			if got.Where() != tt.want {
				t.Fatalf("got %s, want %s", got.Where(), tt.want)
			}
			if tt.cfg.ReplayPath != "" && len(got.Inputs) != 1 {
				t.Fatalf("got %d inputs from the replay, want 1", len(got.Inputs))
			}
		})
	}
}
