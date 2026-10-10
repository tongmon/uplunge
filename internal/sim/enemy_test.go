package sim

import (
	"strings"
	"testing"

	"github.com/tongmon/uplunge/internal/level"
	"github.com/tongmon/uplunge/internal/tuning"
)

// enemyWorld returns a world in a tall empty map with one enemy of the given
// kind centred at (ex, ey) and the player's top-left at (px, py).
func enemyWorld(t *testing.T, kind string, ex, ey, px, py int) *World {
	t.Helper()
	m := level.NewTileMap(13, 200, tile)
	m.Spawns = []level.Spawn{{Name: kind, X: ex, Y: ey}}
	w := NewWorld(testTuning(), m, px, py)
	if len(w.Enemies) != 1 {
		t.Fatalf("%d enemies, want 1", len(w.Enemies))
	}
	return w
}

// stillTuning stops Floaters so geometry stays put in a test.
func stillTuning() tuning.Tuning {
	tun := testTuning()
	for i := range tun.Enemies {
		tun.Enemies[i].Speed = 0
	}
	return tun
}

func stillEnemyWorld(t *testing.T, kind string, ex, ey, px, py int) *World {
	t.Helper()
	m := level.NewTileMap(13, 200, tile)
	m.Spawns = []level.Spawn{{Name: kind, X: ex, Y: ey}}
	return NewWorld(stillTuning(), m, px, py)
}

func TestEnemySpawnsCentred(t *testing.T) {
	w := enemyWorld(t, "Floater", 100, 1000, 96, 0)
	e := w.Enemies[0]
	if e.Body.X != 93 || e.Body.Y != 994 || e.Body.W != 14 || e.Body.H != 12 || e.HP != 2 || e.VX != 30 {
		t.Fatalf("enemy %+v, want a 14x12 Floater at (93, 994) with 2 HP flying right at 30", e)
	}
}

func TestStomp(t *testing.T) {
	tun := stillTuning()
	// Player falling from 40 px above a Floater whose top is at y 1000.
	w := stillEnemyWorld(t, "Floater", 104, 1006, 98, 1000-20-40)
	w.Player.Fuel = 1
	for i := 0; i < Hz && len(w.Enemies) == 1; i++ {
		w.Step(Input{})
	}
	if len(w.Enemies) != 0 {
		t.Fatal("falling onto a Floater did not stomp it")
	}
	if w.Player.VY != -tun.Player.StompSpeed || w.Player.Fuel != tun.Gun.Magazine || w.Player.HP != tun.Player.MaxHP {
		t.Fatalf("after the stomp VY=%v Fuel=%d HP=%d, want -StompSpeed, a full magazine, no damage",
			w.Player.VY, w.Player.Fuel, w.Player.HP)
	}
	// The stomp happens after the step's move, so the bounce drives the
	// next StompHoldTime of moves at StompSpeed, button or not.
	held := 0
	for i := 0; i < Hz; i++ {
		y := w.Player.Body.Y
		w.Step(Input{})
		if w.Player.VY != -tun.Player.StompSpeed || y-w.Player.Body.Y < 4 {
			break
		}
		held++
	}
	if held != 12 { // StompHoldTime 0.2 s at 60 Hz
		t.Fatalf("bounce drove %d moves at StompSpeed, want 12", held)
	}
}

func TestStompBouncesHigherThanAJump(t *testing.T) {
	w := stillEnemyWorld(t, "Floater", 104, 1006, 98, 1000-20-40)
	for i := 0; i < Hz && len(w.Enemies) == 1; i++ {
		w.Step(Input{})
	}
	y0, top := w.Player.Body.Y, w.Player.Body.Y
	for i := 0; i < Hz; i++ {
		w.Step(Input{})
		top = min(top, w.Player.Body.Y)
	}
	if rise, jump := y0-top, jumpHeight(t, Hz); rise <= jump {
		t.Fatalf("stomp bounce rose %d px, want more than a held jump's %d", rise, jump)
	}
}

func TestStompEndsCoyoteTime(t *testing.T) {
	pl := Player{coyoteSteps: 5, jumpHoldSteps: 3}
	pl.stomp(testTuning())
	if pl.coyoteSteps != 0 || pl.jumpHoldSteps != 0 {
		t.Fatalf("coyote %d jump hold %d after a stomp, want both 0", pl.coyoteSteps, pl.jumpHoldSteps)
	}
}

func TestTouchHurts(t *testing.T) {
	tests := []struct {
		name     string
		kind     string
		px, py   int
		wantDir  float64
		hurtFrom string
	}{
		// The enemy's box is (93..106, 994..1005) for a Floater.
		{"Floater from the left", "Floater", 93 - 12, 990, -1, "side"},
		{"Floater from the right", "Floater", 107, 990, 1, "side"},
		{"Floater from below", "Floater", 98, 1006, 1, "below"},
		{"Spiker from above", "Spiker", 98, 993 - 20 - 10, 1, "above"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tun := stillTuning()
			w := stillEnemyWorld(t, tt.kind, 100, 1000, tt.px, tt.py)
			w.Player.Fuel = 0
			// Push the player into the enemy.
			in := Input{}
			switch {
			case tt.hurtFrom == "side" && tt.wantDir < 0:
				in.Right = true
			case tt.hurtFrom == "side":
				in.Left = true
			case tt.hurtFrom == "below":
				w.Player.VY = -200
			}
			for i := 0; i < Hz && w.Player.HP == tun.Player.MaxHP; i++ {
				w.Step(in)
			}
			p := w.Player
			if p.HP != tun.Player.MaxHP-1 {
				t.Fatalf("HP = %d, want one hit", p.HP)
			}
			if p.VX != tt.wantDir*tun.Player.KnockbackX || p.VY != -tun.Player.KnockbackY {
				t.Fatalf("knockback VX=%v VY=%v, want %v, %v", p.VX, p.VY, tt.wantDir*tun.Player.KnockbackX, -tun.Player.KnockbackY)
			}
			if p.Fuel != tun.Gun.Magazine || !p.Invulnerable() || len(w.Enemies) != 1 {
				t.Fatalf("Fuel=%d invulnerable=%v enemies=%d, want refilled, immune, enemy alive",
					p.Fuel, p.Invulnerable(), len(w.Enemies))
			}
		})
	}
}

func TestInvulnerabilityLastsInvulnTime(t *testing.T) {
	// A Spiker right under a player who keeps standing in it: the second
	// hit lands exactly InvulnTime after the first.
	tun := stillTuning()
	tun.Player.KnockbackX, tun.Player.KnockbackY = 0.001, 0.001
	m := level.NewTileMap(13, 200, tile)
	m.Spawns = []level.Spawn{{Name: "Spiker", X: 104, Y: 1010}}
	w := NewWorld(tun, m, 98, 995)
	var hits []int
	for i := 0; i < 3*Hz && len(hits) < 2; i++ {
		hp := w.Player.HP
		w.Step(Input{})
		w.Player.Body.Y, w.Player.VY = 995, 0 // stay inside the Spiker
		if w.Player.HP < hp {
			hits = append(hits, i)
		}
	}
	if len(hits) != 2 || hits[1]-hits[0] != 90 { // InvulnTime 1.5 s
		t.Fatalf("hits at steps %v, want two 90 steps apart", hits)
	}
}

func TestBulletsKillEnemies(t *testing.T) {
	tun := stillTuning()
	// A Spiker (3 HP) right under the player's feet, so every shot hits it.
	w := stillEnemyWorld(t, "Spiker", 104, 1000+20+16, 98, 1000)
	shots := 0
	for i := 0; i < 2*Hz && len(w.Enemies) == 1; i++ {
		fuel := w.Player.Fuel
		w.Step(Input{Button: true})
		if w.Player.Fuel < fuel {
			shots++
		}
	}
	if len(w.Enemies) != 0 || shots != tun.Enemies[1].HP {
		t.Fatalf("enemies %d after %d shots, want the Spiker dead after %d", len(w.Enemies), shots, tun.Enemies[1].HP)
	}
}

func TestBulletStopsAtTheEnemyItHits(t *testing.T) {
	w := stillEnemyWorld(t, "Spiker", 104, 1000+20+30, 98, 1000)
	w.Step(Input{Button: true})
	if len(w.Bullets) != 1 {
		t.Fatal("no bullet in flight")
	}
	for i := 0; i < 10 && len(w.Bullets) > 0; i++ {
		w.Step(Input{})
	}
	if len(w.Bullets) != 0 || w.Enemies[0].HP != 2 {
		t.Fatalf("bullets %d enemy HP %d, want the bullet spent on the Spiker", len(w.Bullets), w.Enemies[0].HP)
	}
}

func TestFloaterTurnsAtWalls(t *testing.T) {
	m := level.NewTileMap(13, 10, tile)
	for r := 0; r < 10; r++ {
		m.Set(0, r, level.Solid)
		m.Set(12, r, level.Solid)
	}
	m.Spawns = []level.Spawn{{Name: "Floater", X: 104, Y: 40}}
	w := NewWorld(testTuning(), m, 98, 120)
	minX, maxX := 1000, 0
	for i := 0; i < 20*Hz; i++ {
		w.Step(Input{})
		minX, maxX = min(minX, w.Enemies[0].Body.X), max(maxX, w.Enemies[0].Body.X)
	}
	if minX != tile || maxX != 12*tile-14 {
		t.Fatalf("Floater ranged over x %d..%d, want wall to wall %d..%d", minX, maxX, tile, 12*tile-14)
	}
}

func TestRunEndsAtZeroHP(t *testing.T) {
	w := stillEnemyWorld(t, "Spiker", 104, 1010, 98, 995)
	w.Player.HP = 1
	w.Step(Input{})
	if !w.Over || w.Player.HP != 0 {
		t.Fatalf("Over=%v HP=%d after the last hit, want the run over", w.Over, w.Player.HP)
	}
	p, tick := w.Player, w.Tick
	run(w, Input{Button: true, Right: true}, 10)
	if w.Player != p || w.Tick != tick+10 {
		t.Fatal("the world kept changing after the run ended")
	}
}

func TestUnknownSpawn(t *testing.T) {
	m, err := level.ParseRows(tile, "#...........#", "#...........#", "#############")
	if err != nil {
		t.Fatal(err)
	}
	m.Spawns = []level.Spawn{{Name: "Nope", X: 50, Y: 20}}
	if _, err := NewWorldInChunk(testTuning(), m); err == nil || !strings.Contains(err.Error(), `"Nope"`) {
		t.Fatalf("error = %v, want one about the Nope spawn", err)
	}
}

// touchWorld returns a world with the player falling onto still enemies
// spawned in the given order, the player's feet 2 px above y 1000.
func touchWorld(t *testing.T, spawns ...level.Spawn) *World {
	t.Helper()
	m := level.NewTileMap(13, 200, tile)
	m.Spawns = spawns
	w := NewWorld(stillTuning(), m, 98, 1000-20-2)
	w.Player.VY = 300
	return w
}

func TestStompWinsOverASimultaneousHit(t *testing.T) {
	// A Floater under the left half of the player and a Spiker under the
	// right half, tops level: landing on both stomps the Floater and does
	// not hurt, whichever was spawned first.
	floater := level.Spawn{Name: "Floater", X: 96, Y: 1000 + 6}
	spiker := level.Spawn{Name: "Spiker", X: 112, Y: 1000 + 7}
	for _, order := range [][]level.Spawn{{floater, spiker}, {spiker, floater}} {
		w := touchWorld(t, order...)
		w.Step(Input{})
		if w.Player.HP != w.Tuning().Player.MaxHP || len(w.Enemies) != 1 || w.Enemies[0].Def.Name != "Spiker" {
			t.Fatalf("spawn order %s first: HP %d, enemies %d, want the Floater stomped and no hit",
				order[0].Name, w.Player.HP, len(w.Enemies))
		}
	}
}

func TestStompKillsEveryEnemyLandedOn(t *testing.T) {
	w := touchWorld(t,
		level.Spawn{Name: "Floater", X: 96, Y: 1000 + 6},
		level.Spawn{Name: "Floater", X: 110, Y: 1000 + 6})
	w.Step(Input{})
	if len(w.Enemies) != 0 {
		t.Fatalf("%d enemies left after landing on two Floaters, want both stomped", len(w.Enemies))
	}
	run(w, Input{}, Hz)
	if w.Player.HP != w.Tuning().Player.MaxHP {
		t.Fatalf("HP %d after the double stomp, want no hit", w.Player.HP)
	}
}

func TestFastBulletHitsTheEnemyItPasses(t *testing.T) {
	// At 1440 px/s a bullet moves 24 px a step, more than an enemy is tall:
	// the bullet's ends go 1016..1023, then 1040..1047, skipping the Spiker
	// at 1025..1038 in between. It must still hit it, and not the block
	// behind it.
	tun := stillTuning()
	tun.Gun.BulletSpeed = 1440
	m := level.NewTileMap(13, 200, tile)
	for c := 0; c < 13; c++ {
		m.Set(c, 66, 4) // soft blocks at y 1056, right behind the enemy
	}
	m.Spawns = []level.Spawn{{Name: "Spiker", X: 104, Y: 1032}}
	w := NewWorld(tun, m, 98, 1000)
	w.Step(Input{Button: true})
	for i := 0; i < 10 && len(w.Bullets) > 0; i++ {
		w.Step(Input{})
	}
	if w.Enemies[0].HP != 2 {
		t.Fatalf("Spiker HP %d, want the fast bullet to hit it", w.Enemies[0].HP)
	}
	if w.Map.At(6, 66) != 4 {
		t.Fatal("the bullet broke the block behind the enemy it hit")
	}
}

func TestBulletHitsTheNearestEnemy(t *testing.T) {
	tun := stillTuning()
	tun.Gun.BulletSpeed = 1440
	for _, order := range [][]string{{"near", "far"}, {"far", "near"}} {
		m := level.NewTileMap(13, 200, tile)
		pos := map[string]int{"near": 1032, "far": 1048}
		for _, n := range order {
			m.Spawns = append(m.Spawns, level.Spawn{Name: "Spiker", X: 104, Y: pos[n]})
		}
		w := NewWorld(tun, m, 98, 1000)
		w.Step(Input{Button: true})
		for i := 0; i < 10 && len(w.Bullets) > 0; i++ {
			w.Step(Input{})
		}
		for _, e := range w.Enemies {
			hit := e.HP < 3
			if near := e.Body.Y < 1035; hit != near {
				t.Fatalf("spawned %v: enemy at y %d hit = %v, want only the nearer one hit", order, e.Body.Y, hit)
			}
		}
	}
}

func TestEnemyInsideAWall(t *testing.T) {
	m, err := level.ParseRows(tile, "#...........#", "#...........#", "#...........#", "#############")
	if err != nil {
		t.Fatal(err)
	}
	m.Spawns = []level.Spawn{{Name: "Spiker", X: 8, Y: 24}}
	if _, err := NewWorldInChunk(testTuning(), m); err == nil || !strings.Contains(err.Error(), "Spiker") {
		t.Fatalf("error = %v, want one about the Spiker inside the wall", err)
	}
}

func TestSetTuningKeepsBlocksThatWouldTrapAnEnemy(t *testing.T) {
	m := level.NewTileMap(13, 200, tile)
	for c := 1; c < 12; c++ {
		m.Set(c, 60, 2) // one-way platforms at y 960
	}
	m.Spawns = []level.Spawn{{Name: "Spiker", X: 104, Y: 962}}
	w := NewWorld(stillTuning(), m, 98, 0)
	tun := stillTuning()
	tun.Blocks[1].OneWay = false
	if err := w.SetTuning(tun); err == nil {
		t.Fatal("making the platform an enemy is inside solid gave no error")
	}
	if w.Map.ShapeAt(6, 60) != level.ShapeOneWay {
		t.Fatal("block definitions changed although they trap an enemy")
	}
}
