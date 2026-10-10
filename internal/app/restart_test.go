package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/replay"
	"github.com/tongmon/uplunge/internal/sim"
	"github.com/tongmon/uplunge/internal/tuning"
)

// restartGame returns a game that has begun start with the replay testdata
// tuning and chunks, as Run sets one up.
func restartGame(t *testing.T, start replay.Replay, newSeeds bool) (*game, tuning.Tuning) {
	t.Helper()
	tun, err := tuning.Load("../replay/testdata/tuning.json")
	if err != nil {
		t.Fatal(err)
	}
	chunks, err := level.LoadLDtk("../replay/testdata/chunks.ldtk")
	if err != nil {
		t.Fatal(err)
	}
	g := &game{chunks: chunks, recording: true, newSeeds: newSeeds}
	if err := g.begin(start, tun); err != nil {
		t.Fatal(err)
	}
	return g, tun
}

// endRun records a few steps, breaking nothing in particular, and ends the
// run.
func endRun(g *game) {
	for i := 0; i < 5; i++ {
		in := sim.Input{Right: true}
		g.recorded = append(g.recorded, in)
		g.world.Step(in)
	}
	g.world.Player.HP = 0
	g.world.Step(sim.Input{})
}

func TestRestart(t *testing.T) {
	tests := []struct {
		name     string
		start    replay.Replay
		newSeeds bool
		sameSeed bool
	}{
		{"new tower", replay.Replay{Tower: true, Seed: 1}, true, false},
		{"fixed seed", replay.Replay{Tower: true, Seed: 1}, false, true},
		{"chunk", replay.Replay{Chunk: "Enemies"}, false, true},
		{"new lab", replay.Replay{Lab: true, Seed: 1}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, tun := restartGame(t, tt.start, tt.newSeeds)
			endRun(g)
			if !g.world.Over {
				t.Fatal("run did not end")
			}
			g.restart()
			if g.world.Over || g.world.Tick != 0 || len(g.recorded) != 0 {
				t.Fatalf("after restart: over %v tick %d recorded %d, want a fresh run with an empty recording",
					g.world.Over, g.world.Tick, len(g.recorded))
			}
			if g.start.Tower != tt.start.Tower || g.start.Lab != tt.start.Lab || g.start.Chunk != tt.start.Chunk ||
				(g.start.Seed == tt.start.Seed) != tt.sameSeed {
				t.Fatalf("next run starts in %s, from %s with newSeeds %v", g.start.Where(), tt.start.Where(), tt.newSeeds)
			}
			// The saved recording replays the new run from where it started.
			_, m, err := replay.Start(g.start, tun, g.chunks)
			if err != nil {
				t.Fatal(err)
			}
			if m.Fingerprint() != g.startMap.Fingerprint() {
				t.Fatal("the recorded start map is not the one the next run started on")
			}
		})
	}
}

func TestRestartFailureKeepsTheEndedRun(t *testing.T) {
	g, tun := restartGame(t, replay.Replay{Chunk: "Enemies"}, false)
	endRun(g)
	tun.Enemies = nil // the Enemies chunk places Floaters and a Spiker
	if err := g.world.SetTuning(tun); err != nil {
		t.Fatal(err)
	}
	old := g.world
	g.restart()
	if g.world != old || !g.world.Over {
		t.Fatal("a restart that could not start replaced the ended run")
	}
}

func TestRestartReadsTuningSavedJustBefore(t *testing.T) {
	g, tun := restartGame(t, replay.Replay{Chunk: "Enemies"}, false)
	path := filepath.Join(t.TempDir(), "tuning.json")
	g.reloadPath, g.reloadWait = path, reloadPollSteps // a poll is not due yet
	endRun(g)
	tun.Enemies[0].Speed = 77
	writeTuning(t, path, tun)
	g.restart()
	for _, e := range g.world.Enemies {
		if e.Def.Name == "Floater" && e.VX != 77 {
			t.Fatalf("Floater VX %v in the next run, want the speed saved before the restart, 77", e.VX)
		}
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestRestartAfterAClear(t *testing.T) {
	g, _ := restartGame(t, replay.Replay{Lab: true, Seed: 4}, false)
	g.recorded = append(g.recorded, sim.Input{Button: true})
	g.world.Cleared, g.world.ClearTick = true, 10
	g.restart()
	if g.world.Cleared || g.world.ClearTick != 0 || len(g.recorded) != 0 || g.start.Seed != 4 || !g.start.Lab {
		t.Fatalf("after restart: cleared %v tick %d recorded %d start %s, want a fresh lab 4",
			g.world.Cleared, g.world.ClearTick, len(g.recorded), g.start.Where())
	}
}

func TestLabRestartsAnytime(t *testing.T) {
	lab, _ := restartGame(t, replay.Replay{Lab: true, Seed: 4}, false)
	tower, _ := restartGame(t, replay.Replay{Tower: true, Seed: 4}, false)
	if !lab.canRestart() || tower.canRestart() {
		t.Fatalf("mid-run restart: lab %v tower %v, want only the lab", lab.canRestart(), tower.canRestart())
	}
	tower.world.Over = true
	if !tower.canRestart() {
		t.Fatal("a tower run that ended cannot restart")
	}
	tower.replaying = true
	if tower.canRestart() {
		t.Fatal("a replay can restart")
	}
}
