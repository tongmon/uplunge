package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/sim"
	"github.com/tongmon/uplunge/internal/tuning"
)

func baseTuning() tuning.Tuning {
	return tuning.Tuning{Player: tuning.Player{
		Width: 12, Height: 20,
		Gravity: 1800, MaxFall: 320,
		JumpSpeed: 210, JumpHoldTime: 0.2,
		CoyoteTime: 0.1, JumpBufferTime: 0.08,
		ApexGravThreshold: 80, ApexGravMult: 0.5,
		RunSpeed: 180, RunAccel: 2000, AirAccelMult: 0.65,
		OneWayAssist: 80, CornerCorrection: 4, DrillBounce: 240,
		StompSpeed: 280, StompHoldTime: 0.2,
		MaxHP: 4, KnockbackX: 180, KnockbackY: 190, InvulnTime: 1.5,
	}, Gun: tuning.Gun{
		Magazine: 8, FireInterval: 0.1, Thrust: 240,
		BulletSpeed: 480, BulletLife: 0.2, BulletWidth: 4, BulletHeight: 8,
	}, Camera: tuning.Camera{
		Anchor: 0.667, Lookahead: 0.2, RemainPerSecond: 0.01,
	}, Water: tuning.Water{
		Speed: 60, Baseline: 24, MaxLag: 192, MaxMult: 2, SlowRange: 64, MinMult: 0.5,
		StartBelow: 32, Bounce: 400, Retreat: 96, RetreatTime: 0.4, PauseTime: 0.5,
	}, Tower: tuning.Tower{
		Base: "Start", Pool: []string{"Shaft", "Blocks"}, Length: 6,
	}, Blocks: []tuning.Block{
		{Value: 1, Name: "solid", Color: "#707070"},
		{Value: 2, Name: "one_way", OneWay: true, Color: "#a08060"},
		{Value: 3, Name: "drill", Drill: true, Color: "#c06040"},
		{Value: 4, Name: "soft", Drill: true, Bullet: true, Color: "#b0a040"},
		{Value: 5, Name: "bullet_only", Bullet: true, Color: "#4080c0"},
	}, Enemies: []tuning.Enemy{
		{Name: "Floater", Width: 14, Height: 12, HP: 2, Stompable: true, Speed: 30, Color: "#60b060"},
		{Name: "Spiker", Width: 14, Height: 14, HP: 3, Color: "#c04060"},
	}}
}

// fixedTime is used for every write so the tests prove that changes are
// found by content, not by modification time.
var fixedTime = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

func writeTuning(t *testing.T, path string, tun tuning.Tuning) {
	t.Helper()
	data, err := json.Marshal(tun)
	if err != nil {
		t.Fatal(err)
	}
	writeRaw(t, path, string(data))
}

func writeRaw(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, fixedTime, fixedTime); err != nil {
		t.Fatal(err)
	}
}

// reloadWorld returns a player standing on the floor of an open room, and a
// tuning file holding the same values.
func reloadWorld(t *testing.T) (*sim.World, string) {
	t.Helper()
	m, err := level.ParseRows(16,
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#############",
	)
	if err != nil {
		t.Fatal(err)
	}
	w := sim.NewWorld(baseTuning(), m, 96, 4*16-20)
	path := filepath.Join(t.TempDir(), "tuning.json")
	writeTuning(t, path, baseTuning())
	return w, path
}

func TestReloadTuning(t *testing.T) {
	w, path := reloadWorld(t)

	if applied, err := reloadTuning(w, path); applied || err != nil {
		t.Fatalf("unchanged file: applied=%v err=%v, want nothing", applied, err)
	}

	// Same length, same modification time, different value.
	faster := baseTuning()
	faster.Player.RunSpeed = 190
	writeTuning(t, path, faster)
	if applied, err := reloadTuning(w, path); !applied || err != nil || w.Tuning().Player.RunSpeed != 190 {
		t.Fatalf("edit: applied=%v err=%v run=%v, want 190 applied", applied, err, w.Tuning().Player.RunSpeed)
	}
	if applied, err := reloadTuning(w, path); applied || err != nil {
		t.Fatalf("same edit again: applied=%v err=%v, want nothing", applied, err)
	}

	writeRaw(t, path, `{"player": {`)
	if applied, err := reloadTuning(w, path); applied || err == nil || w.Tuning().Player.RunSpeed != 190 {
		t.Fatalf("broken file: applied=%v err=%v, want an error and the values kept", applied, err)
	}
	// Still broken: retried and reported again rather than remembered as seen.
	if _, err := reloadTuning(w, path); err == nil {
		t.Fatal("broken file not retried")
	}

	writeTuning(t, path, baseTuning())
	if applied, err := reloadTuning(w, path); !applied || err != nil || w.Tuning().Player.RunSpeed != 180 {
		t.Fatalf("fixed file: applied=%v err=%v run=%v, want 180 applied", applied, err, w.Tuning().Player.RunSpeed)
	}
}

func TestReloadTuningRetriesBlockedResize(t *testing.T) {
	w, path := reloadWorld(t)
	// Ceiling two tiles above the floor.
	for c := 1; c < 12; c++ {
		w.Map.Set(c, 1, level.Solid)
	}
	tall := baseTuning()
	tall.Player.Height = 40
	tall.Player.RunSpeed = 90
	writeTuning(t, path, tall)

	for i := 0; i < 2; i++ {
		applied, err := reloadTuning(w, path)
		if !applied || err == nil {
			t.Fatalf("poll %d under the ceiling: applied=%v err=%v, want applied with an error", i, applied, err)
		}
		if p := w.Tuning().Player; p.Height != 20 || p.RunSpeed != 90 {
			t.Fatalf("poll %d: height %d run %v, want the kept 20 and the new 90", i, p.Height, p.RunSpeed)
		}
	}

	for c := 1; c < 12; c++ {
		w.Map.Set(c, 1, level.Empty)
	}
	if applied, err := reloadTuning(w, path); !applied || err != nil || w.Player.Body.H != 40 {
		t.Fatalf("with room: applied=%v err=%v height=%d, want 40", applied, err, w.Player.Body.H)
	}
}
