package sim

import (
	"fmt"
	"testing"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

// testTuning mirrors data/tuning.json so these tests pin the behavior of the
// current hypotheses rather than whatever the file says later.
func testTuning() tuning.Tuning {
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
		Speed: 30, Baseline: 24, MaxLag: 48, MaxMult: 3, SlowRange: 64, MinMult: 0.5,
		StartBelow: 32, Bounce: 400, Retreat: 96, RetreatTime: 0.4, PauseTime: 0.5,
	}, Feel: tuning.Feel{
		StompFreeze: 0.0167, HitFreeze: 0.0167, DrillFreeze: 0.05, ShakeTime: 0.2, ShakeInterval: 0.04, ShakeScale: 20,
		StompShakeTime: 0.16, StompShakeScale: 12,
		JumpX: 0.6, JumpY: 1.4, LandX: 1.6, LandY: 0.4, LandSpeed: 480, Recover: 1.75,
		FullColor: "#ff9a3c", EmptyColor: "#3c78ff", FlashTime: 0.12,
	}, Tower: tuning.Tower{
		Base: "Start", Pool: []string{"Shaft", "Blocks"}, Length: 6,
	}, Lab: tuning.Lab{
		Rows: 300, Enemy: "Floater", Spacing: 96, Jitter: 16,
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

const tile = 16

// testRoom is 13x12 tiles: walls on both sides and a floor whose top is at
// y = 11*16 = 176. The inner space spans x 16..191.
func testRoom(t *testing.T) *level.TileMap {
	t.Helper()
	m, err := level.ParseRows(tile,
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#...........#",
		"#############",
	)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

const floorY = 11 * tile

// standingWorld returns a world with the player standing on the floor.
func standingWorld(t *testing.T) *World {
	t.Helper()
	tun := testTuning()
	w := NewWorld(tun, testRoom(t), 96, floorY-tun.Player.Height)
	w.Step(Input{})
	if !w.Player.OnGround {
		t.Fatal("player does not start on the ground")
	}
	return w
}

func run(w *World, in Input, steps int) {
	for i := 0; i < steps; i++ {
		w.Step(in)
	}
}

func TestStepAdvancesOneTick(t *testing.T) {
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	run(w, Input{}, 3*Hz)
	if got, want := w.Tick, uint64(3*Hz); got != want {
		t.Fatalf("Tick = %d, want %d", got, want)
	}
}

func TestDtMatchesHz(t *testing.T) {
	if got := Dt * Hz; got != 1 {
		t.Fatalf("Dt * Hz = %v, want 1", got)
	}
}

func TestFallsAndLands(t *testing.T) {
	tun := testTuning()
	w := NewWorld(tun, testRoom(t), 96, 0)
	run(w, Input{}, Hz)
	p := w.Player
	if !p.OnGround || p.VY != 0 || p.Body.Y != floorY-tun.Player.Height {
		t.Fatalf("after 1 s: Y=%d VY=%v OnGround=%v, want Y=%d VY=0 on ground",
			p.Body.Y, p.VY, p.OnGround, floorY-tun.Player.Height)
	}
}

func TestFallSpeedIsCapped(t *testing.T) {
	tun := testTuning()
	tall := level.NewTileMap(13, 200, tile)
	w := NewWorld(tun, tall, 96, 0)
	run(w, Input{}, Hz)
	if w.Player.VY != tun.Player.MaxFall {
		t.Fatalf("VY = %v after 1 s of falling, want %v", w.Player.VY, tun.Player.MaxFall)
	}
}

func TestRunReachesRunSpeed(t *testing.T) {
	w := standingWorld(t)
	x0 := w.Player.Body.X
	run(w, Input{Right: true}, Hz/4)
	if w.Player.VX != testTuning().Player.RunSpeed {
		t.Fatalf("VX = %v, want %v", w.Player.VX, testTuning().Player.RunSpeed)
	}
	if w.Player.Body.X <= x0 {
		t.Fatalf("X = %d, want more than %d", w.Player.Body.X, x0)
	}
}

func TestAirAccelIsReduced(t *testing.T) {
	ground := standingWorld(t)
	ground.Step(Input{Right: true})

	air := NewWorld(testTuning(), testRoom(t), 96, 0)
	air.Step(Input{Right: true})

	p := testTuning().Player
	if got, want := ground.Player.VX, p.RunAccel*Dt; got != want {
		t.Errorf("ground VX after one step = %v, want %v", got, want)
	}
	if got, want := air.Player.VX, p.RunAccel*p.AirAccelMult*Dt; got != want {
		t.Errorf("air VX after one step = %v, want %v", got, want)
	}
}

func TestRunStopsAtWall(t *testing.T) {
	w := standingWorld(t)
	run(w, Input{Right: true}, 2*Hz)
	wantX := 12*tile - testTuning().Player.Width
	if w.Player.Body.X != wantX || w.Player.VX != 0 {
		t.Fatalf("X=%d VX=%v, want X=%d VX=0", w.Player.Body.X, w.Player.VX, wantX)
	}
}

// jumpHeight presses the button on a standing player, keeps it down for
// holdSteps, and returns how many pixels the player rose at the peak.
func jumpHeight(t *testing.T, holdSteps int) int {
	w := standingWorld(t)
	y0, top := w.Player.Body.Y, w.Player.Body.Y
	for i := 0; i < 2*Hz; i++ {
		w.Step(Input{Button: i < holdSteps})
		top = min(top, w.Player.Body.Y)
	}
	if !w.Player.OnGround {
		t.Fatal("player did not land within 2 s")
	}
	return y0 - top
}

func TestHeldJumpRisesAboutThreeAndAHalfTiles(t *testing.T) {
	h := jumpHeight(t, Hz)
	t.Logf("held jump height: %d px (%.2f tiles)", h, float64(h)/tile)
	if h < 3*tile || h > 4*tile {
		t.Fatalf("held jump rose %d px, want 3 to 4 tiles (%d..%d)", h, 3*tile, 4*tile)
	}
}

func TestJumpHoldLastsJumpHoldTime(t *testing.T) {
	p := testTuning().Player
	w := standingWorld(t)
	held := 0
	for i := 0; i < Hz; i++ {
		w.Step(Input{Button: true})
		if w.Player.VY == -p.JumpSpeed {
			held++
		}
	}
	// The launch step counts as the first held step.
	if want := int(p.JumpHoldTime * Hz); held != want {
		t.Fatalf("VY stayed at -JumpSpeed for %d steps, want %d", held, want)
	}
}

func TestTapJumpIsLower(t *testing.T) {
	tap, held := jumpHeight(t, 1), jumpHeight(t, Hz)
	t.Logf("tap jump height: %d px", tap)
	if tap <= 0 || tap >= held {
		t.Fatalf("tap jump rose %d px, want between 0 and the held %d px", tap, held)
	}
}

func TestNoJumpInAir(t *testing.T) {
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	w.Step(Input{})
	w.Player.Fuel = 0 // so the press cannot fire either
	w.Step(Input{Button: true})
	if w.Player.VY < 0 {
		t.Fatalf("VY = %v after pressing in the air, want no jump", w.Player.VY)
	}
}

func TestHoldingButtonDoesNotRejump(t *testing.T) {
	w := standingWorld(t)
	run(w, Input{Button: true}, 2*Hz)
	if !w.Player.OnGround {
		t.Fatal("player is not on the ground after holding the button for 2 s")
	}
	run(w, Input{Button: true}, 10)
	if !w.Player.OnGround || w.Player.VY != 0 {
		t.Fatalf("player jumped again without a new press: VY=%v", w.Player.VY)
	}
}

func TestHeadBumpEndsJump(t *testing.T) {
	tun := testTuning()
	m := testRoom(t)
	// A ceiling whose underside is two tiles above the floor.
	for c := 1; c < 12; c++ {
		m.Set(c, 8, level.Solid)
	}
	w := NewWorld(tun, m, 96, floorY-tun.Player.Height)
	w.Step(Input{})
	top := w.Player.Body.Y
	// Hold for less than JumpHoldTime: without the cancel the head would stay
	// pinned to the ceiling for the whole hold.
	for i := 0; i < int(tun.Player.JumpHoldTime*Hz)-1; i++ {
		w.Step(Input{Button: true})
		top = min(top, w.Player.Body.Y)
	}
	if top != 9*tile {
		t.Fatalf("top Y = %d, want head against the ceiling at %d", top, 9*tile)
	}
	if w.Player.Body.Y == top {
		t.Fatal("player still pinned to the ceiling, want the hold cancelled")
	}
}

func TestNewWorldInChunk(t *testing.T) {
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
			m, err := level.ParseRows(tile, tt.rows...)
			if err != nil {
				t.Fatal(err)
			}
			w, err := NewWorldInChunk(testTuning(), m)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("got player at (%d, %d), want an error", w.Player.Body.X, w.Player.Body.Y)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if b := w.Player.Body; b.X != tt.wantX || b.Y != tt.wantY {
				t.Fatalf("player at (%d, %d), want (%d, %d)", b.X, b.Y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestSetTuningAppliesMovement(t *testing.T) {
	w := standingWorld(t)
	tun := testTuning()
	tun.Player.RunSpeed = 90
	if err := w.SetTuning(tun); err != nil {
		t.Fatal(err)
	}
	run(w, Input{Right: true}, Hz/2)
	if w.Player.VX != 90 {
		t.Fatalf("VX = %v after the new tuning, want 90", w.Player.VX)
	}
	if w.Tuning().Player.RunSpeed != 90 {
		t.Fatal("Tuning() does not report the new tuning")
	}
}

func TestSetTuningResizesAroundFeet(t *testing.T) {
	tests := []struct {
		name         string
		w, h         int
		wantX, wantY int
	}{
		{"grow", 16, 24, 94, floorY - 24},
		{"shrink", 8, 10, 98, floorY - 10},
		// The left edge moves by -1.5 px: X 94 plus a 0.5 px remainder.
		{"odd difference", 15, 20, 94, floorY - 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := standingWorld(t) // 12x20 at x 96
			tun := testTuning()
			tun.Player.Width, tun.Player.Height = tt.w, tt.h
			if err := w.SetTuning(tun); err != nil {
				t.Fatal(err)
			}
			b := w.Player.Body
			if b.X != tt.wantX || b.Y != tt.wantY || b.W != tt.w || b.H != tt.h {
				t.Fatalf("body %dx%d at (%d, %d), want %dx%d at (%d, %d)",
					b.W, b.H, b.X, b.Y, tt.w, tt.h, tt.wantX, tt.wantY)
			}
			w.Step(Input{})
			if !w.Player.OnGround {
				t.Fatal("resized player is no longer standing on the floor")
			}
		})
	}
}

func TestSetTuningKeepsSizeWhenBlocked(t *testing.T) {
	tun := testTuning()
	m := testRoom(t)
	// A ceiling whose underside is two tiles (32 px) above the floor.
	for c := 1; c < 12; c++ {
		m.Set(c, 8, level.Solid)
	}
	w := NewWorld(tun, m, 96, floorY-tun.Player.Height)
	w.Step(Input{})

	tall := testTuning()
	tall.Player.Height = 40
	tall.Player.RunSpeed = 90
	if err := w.SetTuning(tall); err == nil {
		t.Fatal("growing into the ceiling did not report an error")
	}
	if b := w.Player.Body; b.H != 20 || b.Y != floorY-20 {
		t.Fatalf("body %dx%d at y %d, want the old 12x20 at y %d", b.W, b.H, b.Y, floorY-20)
	}
	if got := w.Tuning().Player; got.Height != 20 || got.RunSpeed != 90 {
		t.Fatalf("tuning height %d run %v, want the kept height 20 and the new run speed 90",
			got.Height, got.RunSpeed)
	}
}

func TestSetTuningResizeRoundTripDoesNotDrift(t *testing.T) {
	w := standingWorld(t) // 12x20 at x 96
	for _, width := range []int{13, 14, 12, 13, 12} {
		tun := testTuning()
		tun.Player.Width = width
		if err := w.SetTuning(tun); err != nil {
			t.Fatal(err)
		}
	}
	if w.Player.Body.X != 96 {
		t.Fatalf("X = %d after resizing back to 12, want 96", w.Player.Body.X)
	}
}

// ledgeWorld returns a world with the player standing near the right end of
// a ledge (tiles 1..5 of row 9) above a floor at the bottom of a 20-row room.
func ledgeWorld(t *testing.T) *World {
	t.Helper()
	rows := []string{}
	for r := 0; r < 9; r++ {
		rows = append(rows, "#...........#")
	}
	rows = append(rows, "######......#")
	for r := 10; r < 19; r++ {
		rows = append(rows, "#...........#")
	}
	rows = append(rows, "#############")
	m, err := level.ParseRows(tile, rows...)
	if err != nil {
		t.Fatal(err)
	}
	tun := testTuning()
	w := NewWorld(tun, m, 6*tile-tun.Player.Width-2, 9*tile-tun.Player.Height)
	w.Step(Input{})
	if !w.Player.OnGround {
		t.Fatal("player does not start on the ledge")
	}
	return w
}

// walkOff runs right until the player has left the ledge, so the next step is
// the first one taken in the air.
func walkOff(t *testing.T) *World {
	t.Helper()
	w := ledgeWorld(t)
	for i := 0; w.Player.OnGround; i++ {
		if i > Hz {
			t.Fatal("player did not walk off the ledge within 1 s")
		}
		w.Step(Input{Right: true})
	}
	return w
}

func TestCoyoteJump(t *testing.T) {
	p := testTuning().Player
	n := 6 // CoyoteTime 0.1 s at 60 Hz
	for k := 1; k <= n+2; k++ {
		t.Run(fmt.Sprintf("press on air step %d", k), func(t *testing.T) {
			w := walkOff(t)
			run(w, Input{Right: true}, k-1)
			w.Step(Input{Right: true, Button: true})
			jumped := w.Player.VY == -p.JumpSpeed
			if want := k <= n; jumped != want {
				t.Fatalf("jumped = %v (VY %v), want %v with %d coyote steps", jumped, w.Player.VY, want, n)
			}
		})
	}
}

func TestCoyoteTimeEndsWithJump(t *testing.T) {
	p := testTuning().Player
	w := standingWorld(t)
	w.Step(Input{Button: true})
	w.Step(Input{})
	w.Step(Input{Button: true})
	if w.Player.VY == -p.JumpSpeed {
		t.Fatal("a second press right after a jump jumped again")
	}
}

// landingStep drops the player from the top of testRoom with no input and
// returns how many steps it takes until the player reports OnGround.
func landingStep(t *testing.T) int {
	t.Helper()
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	for i := 1; i <= 2*Hz; i++ {
		w.Step(Input{})
		if w.Player.OnGround {
			return i
		}
	}
	t.Fatal("player did not land within 2 s")
	return 0
}

func TestJumpBuffer(t *testing.T) {
	p := testTuning().Player
	n := 5 // JumpBufferTime 0.08 s at 60 Hz, rounded
	land := landingStep(t)
	// Step land+1 is the first one that starts on the ground. A tap j steps
	// earlier is remembered for n steps, counting the step it happened on.
	for j := 0; j <= n+1; j++ {
		t.Run(fmt.Sprintf("tap %d steps early", j), func(t *testing.T) {
			w := NewWorld(testTuning(), testRoom(t), 96, 0)
			w.Player.Fuel = 0 // only a press that cannot fire is buffered
			tapAt := land + 1 - j
			for i := 1; i <= land+1; i++ {
				w.Step(Input{Button: i == tapAt})
			}
			jumped := w.Player.VY == -p.JumpSpeed
			if want := j < n; jumped != want {
				t.Fatalf("jumped = %v (VY %v), want %v with a %d-step buffer", jumped, w.Player.VY, want, n)
			}
		})
	}
}

func TestBufferedJumpIsUsedOnce(t *testing.T) {
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	w.Player.Fuel = 0 // only a press that cannot fire is buffered
	land := landingStep(t)
	for i := 1; i <= land+1; i++ {
		w.Step(Input{Button: i == land})
	}
	if w.Player.VY != -testTuning().Player.JumpSpeed {
		t.Fatal("buffered press did not jump")
	}
	run(w, Input{}, 2*Hz)
	if !w.Player.OnGround || w.Player.VY != 0 {
		t.Fatalf("player jumped again from the same press: VY=%v OnGround=%v", w.Player.VY, w.Player.OnGround)
	}
}

func TestApexGravity(t *testing.T) {
	p := testTuning().Player
	full := p.Gravity * Dt
	tests := []struct {
		name   string
		vy     float64
		button bool
		want   float64
	}{
		{"slow rise held", -50, true, -50 + full*p.ApexGravMult},
		{"slow fall held", 50, true, 50 + full*p.ApexGravMult},
		{"slow rise released", -50, false, -50 + full},
		{"fast rise held", -100, true, -100 + full},
		{"at threshold held", -p.ApexGravThreshold, true, -p.ApexGravThreshold + full},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWorld(testTuning(), testRoom(t), 96, 64)
			w.Player.VY = tt.vy
			w.Player.Fuel = 0 // isolate gravity from the gunjet
			w.Step(Input{Button: tt.button})
			if w.Player.VY != tt.want {
				t.Fatalf("VY = %v, want %v", w.Player.VY, tt.want)
			}
		})
	}
}

func TestReleasedBufferedJumpCannotBeExtended(t *testing.T) {
	p := testTuning().Player
	land := landingStep(t)
	w := NewWorld(testTuning(), testRoom(t), 96, 0)
	w.Player.Fuel = 0 // only a press that cannot fire is buffered
	// Tap and release before landing, so the buffered jump launches with the
	// button up, then press again right after the launch.
	for i := 1; i <= land+1; i++ {
		w.Step(Input{Button: i == land})
	}
	if w.Player.VY != -p.JumpSpeed {
		t.Fatal("buffered press did not jump")
	}
	w.Step(Input{Button: true})
	if w.Player.VY == -p.JumpSpeed {
		t.Fatal("a new press after a released buffered jump extended the jump hold")
	}
}
