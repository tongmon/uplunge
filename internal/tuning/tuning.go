// Package tuning defines the tunable numbers the simulation reads and parses
// them from JSON. It must not depend on Ebitengine.
package tuning

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
)

// Tuning is the full set of tunables. Every value is a prototype hypothesis
// until the developer confirms it in docs/design.md.
type Tuning struct {
	Player Player  `json:"player"`
	Gun    Gun     `json:"gun"`
	Camera Camera  `json:"camera"`
	Water  Water   `json:"water"`
	Feel   Feel    `json:"feel"`
	Tower  Tower   `json:"tower"`
	Blocks []Block `json:"blocks"`
	// Enemies defines every enemy kind, by the name chunks place it with.
	Enemies []Enemy `json:"enemies"`
}

// Enemy defines one enemy kind.
type Enemy struct {
	// Name matches the entity identifier in the LDtk project.
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	// HP is the number of bullet hits it takes. A stomp always kills.
	HP int `json:"hp"`
	// Stompable enemies are safe on top; the rest hurt from every side.
	Stompable bool `json:"stompable,omitempty"`
	// Speed is how fast it flies sideways, turning at walls. 0 keeps it in
	// place.
	Speed float64 `json:"speed"`
	// Color is the grey-box draw color, "#rrggbb".
	Color string `json:"color"`
}

// RGB returns Color as bytes. Parse has already checked its format.
func (e Enemy) RGB() (r, g, b uint8) {
	return Block{Color: e.Color}.RGB()
}

// Camera holds how the view follows the player up the tower.
type Camera struct {
	// Anchor is where the player's centre sits on screen, as a fraction of
	// the view height from the top.
	Anchor float64 `json:"anchor"`
	// Lookahead shows this many seconds of upward travel ahead: the target
	// moves up by -VY * Lookahead while the player rises.
	Lookahead float64 `json:"lookahead"`
	// RemainPerSecond is the fraction of the distance to the target still
	// left after following for one second.
	RemainPerSecond float64 `json:"remainPerSecond"`
}

// Water holds the rising water and its fall safety net. Distances are px,
// speeds px/s, times seconds.
type Water struct {
	// Speed is the base rising speed.
	Speed float64 `json:"speed"`
	// Baseline is how far above the bottom of the view the water aims to be.
	Baseline float64 `json:"baseline"`
	// MaxLag is the furthest the water falls behind below the baseline.
	// Over that distance its speed grows from 1 to MaxMult times Speed.
	MaxLag  float64 `json:"maxLag"`
	MaxMult float64 `json:"maxMult"`
	// Above the baseline, the speed drops from 1 to MinMult times Speed
	// over SlowRange.
	SlowRange float64 `json:"slowRange"`
	MinMult   float64 `json:"minMult"`
	// StartBelow is how far under the map's bottom edge the water starts.
	StartBelow float64 `json:"startBelow"`

	// Catching the player launches it up at Bounce, eases the water down by
	// Retreat over RetreatTime, and stops it rising for PauseTime from the
	// catch.
	Bounce      float64 `json:"bounce"`
	Retreat     float64 `json:"retreat"`
	RetreatTime float64 `json:"retreatTime"`
	PauseTime   float64 `json:"pauseTime"`
}

// Feel holds the effects that sell impacts. FreezeTime is simulation: the
// world stands still for it after a stomp or a drill break. The rest only
// changes how the world is drawn. Times are seconds.
type Feel struct {
	FreezeTime float64 `json:"freezeTime"`

	// Firing shakes the view along the shot for ShakeTime. Every
	// ShakeInterval the offset flips side, ShakeScale px per second of shake
	// left, rounded up.
	ShakeTime     float64 `json:"shakeTime"`
	ShakeInterval float64 `json:"shakeInterval"`
	ShakeScale    float64 `json:"shakeScale"`

	// A jump stretches the drawn player to Jump{X,Y} times its size; a
	// landing squashes it toward Land{X,Y}, fully at LandSpeed or faster.
	// Both ease back to 1 at Recover per second.
	JumpX     float64 `json:"jumpX"`
	JumpY     float64 `json:"jumpY"`
	LandX     float64 `json:"landX"`
	LandY     float64 `json:"landY"`
	LandSpeed float64 `json:"landSpeed"`
	Recover   float64 `json:"recover"`

	// The helmet lamp shows the fuel: FullColor with a full magazine,
	// shading to EmptyColor at none. It flashes white for FlashTime on a
	// refill.
	FullColor  string  `json:"fullColor"`
	EmptyColor string  `json:"emptyColor"`
	FlashTime  float64 `json:"flashTime"`
}

// RGB parses a "#rrggbb" color that Parse has already checked.
func RGB(color string) (r, g, b uint8) {
	return Block{Color: color}.RGB()
}

// MaxValue caps every number in the tuning, so a typo such as 1e308 cannot
// overflow the simulation's arithmetic.
const MaxValue = 1e6

// MaxCameraRemain caps Camera.RemainPerSecond. Closer to 1, a step of
// following moves the camera by less than float precision and it stops.
const MaxCameraRemain = 0.99

// MaxTowerLength matches level.MaxTowerLength, which this package cannot
// import.
const MaxTowerLength = 1000

// Clone returns a copy that shares no slices with t, so a caller cannot
// change a copy someone else keeps.
func (t Tuning) Clone() Tuning {
	t.Blocks = slices.Clone(t.Blocks)
	t.Tower.Pool = slices.Clone(t.Tower.Pool)
	t.Enemies = slices.Clone(t.Enemies)
	return t
}

// Tower holds how a run's tower is stacked from chunks. It is read once,
// when a run starts.
type Tower struct {
	// Base is the bottom chunk, where the player starts.
	Base string `json:"base"`
	// Pool lists the chunks drawn at random above the base.
	Pool []string `json:"pool"`
	// Length is the number of chunks, the base included.
	Length int `json:"length"`
}

// Player holds the player's size and movement numbers. Speeds are px/s,
// accelerations px/s², and times seconds. Speeds are magnitudes; the
// simulation applies the direction (y grows downward).
type Player struct {
	// Width and Height are the hitbox size, applied when the world is created.
	Width  int `json:"width"`
	Height int `json:"height"`

	Gravity float64 `json:"gravity"`
	MaxFall float64 `json:"maxFall"`

	// JumpSpeed is held for up to JumpHoldTime while the button stays down.
	JumpSpeed    float64 `json:"jumpSpeed"`
	JumpHoldTime float64 `json:"jumpHoldTime"`
	// CoyoteTime is how long after leaving the ground a jump is still allowed.
	CoyoteTime float64 `json:"coyoteTime"`
	// JumpBufferTime is how long a press is remembered until a jump is allowed.
	JumpBufferTime float64 `json:"jumpBufferTime"`
	// Gravity is scaled by ApexGravMult while the button is held and the
	// vertical speed is below ApexGravThreshold.
	ApexGravThreshold float64 `json:"apexGravThreshold"`
	ApexGravMult      float64 `json:"apexGravMult"`

	RunSpeed float64 `json:"runSpeed"`
	RunAccel float64 `json:"runAccel"`
	// AirAccelMult scales RunAccel while airborne.
	AirAccelMult float64 `json:"airAccelMult"`

	// OneWayAssist is the extra upward speed while rising through a one-way
	// platform, so a jump that barely reaches it still gets on top.
	OneWayAssist float64 `json:"oneWayAssist"`
	// CornerCorrection is how many pixels sideways the head may slide past
	// a ceiling corner instead of bumping into it.
	CornerCorrection int `json:"cornerCorrection"`
	// DrillBounce caps the upward speed after the head breaks a block:
	// VY = min(VY, -DrillBounce).
	DrillBounce float64 `json:"drillBounce"`

	// StompSpeed is held as the upward speed for StompHoldTime after
	// stomping an enemy, whether or not the button is down.
	StompSpeed    float64 `json:"stompSpeed"`
	StompHoldTime float64 `json:"stompHoldTime"`

	// MaxHP is the HP a run starts with. A hit takes 1, refills the
	// magazine, knocks the player up by KnockbackY and away by up to
	// KnockbackX (scaled by how much the hit came from the side), and makes
	// the player immune to hits for InvulnTime.
	MaxHP      int     `json:"maxHP"`
	KnockbackX float64 `json:"knockbackX"`
	KnockbackY float64 `json:"knockbackY"`
	InvulnTime float64 `json:"invulnTime"`
}

// Block defines one non-empty value of the chunks' Collision layer.
type Block struct {
	// Value is the Collision IntGrid value, 1 to 255.
	Value int `json:"value"`
	// Name matches the value's identifier in the LDtk project.
	Name string `json:"name"`
	// OneWay blocks are platforms passable from below; the rest are solid.
	OneWay bool `json:"oneWay,omitempty"`
	// Drill blocks break when the player's head hits them from below, and
	// Bullet blocks break when a bullet hits them.
	Drill  bool `json:"drill,omitempty"`
	Bullet bool `json:"bullet,omitempty"`
	// Color is the grey-box draw color, "#rrggbb".
	Color string `json:"color"`
}

// RGB returns Color as bytes. Parse has already checked its format.
func (b Block) RGB() (r, g, bl uint8) {
	v, _ := strconv.ParseUint(b.Color[1:], 16, 32)
	return uint8(v >> 16), uint8(v >> 8), uint8(v)
}

// Gun holds the gunjet: firing downward in the air pushes the player up.
// Times are seconds and speeds px/s, as in Player.
type Gun struct {
	// Magazine is the fuel in shots. It refills on landing.
	Magazine int `json:"magazine"`
	// FireInterval is the time between shots while the button is held.
	FireInterval float64 `json:"fireInterval"`
	// Thrust caps the upward speed a shot leaves: VY = min(VY, -Thrust).
	Thrust float64 `json:"thrust"`

	// Bullets fly straight down at BulletSpeed for BulletLife seconds, or
	// until they hit a solid.
	BulletSpeed  float64 `json:"bulletSpeed"`
	BulletLife   float64 `json:"bulletLife"`
	BulletWidth  int     `json:"bulletWidth"`
	BulletHeight int     `json:"bulletHeight"`
}

// Load reads and parses a tuning file.
func Load(path string) (Tuning, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Tuning{}, fmt.Errorf("tuning: %w", err)
	}
	t, err := Parse(data)
	if err != nil {
		return Tuning{}, fmt.Errorf("tuning: %s: %w", path, err)
	}
	return t, nil
}

// Parse decodes tuning JSON. Unknown fields and missing or non-positive
// values are errors, so a typo cannot silently zero a number.
func Parse(data []byte) (Tuning, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var t Tuning
	if err := dec.Decode(&t); err != nil {
		return Tuning{}, err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return Tuning{}, fmt.Errorf("unexpected data after the tuning object")
	}
	if err := t.validate(); err != nil {
		return Tuning{}, err
	}
	return t, nil
}

func (t Tuning) validate() error {
	p, g := t.Player, t.Gun
	positive := []struct {
		name string
		v    float64
	}{
		{"player.width", float64(p.Width)},
		{"player.height", float64(p.Height)},
		{"player.gravity", p.Gravity},
		{"player.maxFall", p.MaxFall},
		{"player.jumpSpeed", p.JumpSpeed},
		{"player.jumpHoldTime", p.JumpHoldTime},
		{"player.coyoteTime", p.CoyoteTime},
		{"player.jumpBufferTime", p.JumpBufferTime},
		{"player.apexGravThreshold", p.ApexGravThreshold},
		{"player.apexGravMult", p.ApexGravMult},
		{"player.runSpeed", p.RunSpeed},
		{"player.runAccel", p.RunAccel},
		{"player.airAccelMult", p.AirAccelMult},
		{"player.oneWayAssist", p.OneWayAssist},
		{"player.cornerCorrection", float64(p.CornerCorrection)},
		{"player.drillBounce", p.DrillBounce},
		{"player.stompSpeed", p.StompSpeed},
		{"player.stompHoldTime", p.StompHoldTime},
		{"player.maxHP", float64(p.MaxHP)},
		{"player.knockbackX", p.KnockbackX},
		{"player.knockbackY", p.KnockbackY},
		{"player.invulnTime", p.InvulnTime},
		{"water.speed", t.Water.Speed},
		{"water.baseline", t.Water.Baseline},
		{"water.maxLag", t.Water.MaxLag},
		{"water.maxMult", t.Water.MaxMult},
		{"water.slowRange", t.Water.SlowRange},
		{"water.minMult", t.Water.MinMult},
		{"water.startBelow", t.Water.StartBelow},
		{"water.bounce", t.Water.Bounce},
		{"water.retreat", t.Water.Retreat},
		{"water.retreatTime", t.Water.RetreatTime},
		{"water.pauseTime", t.Water.PauseTime},
		{"feel.freezeTime", t.Feel.FreezeTime},
		{"feel.shakeTime", t.Feel.ShakeTime},
		{"feel.shakeInterval", t.Feel.ShakeInterval},
		{"feel.shakeScale", t.Feel.ShakeScale},
		{"feel.jumpX", t.Feel.JumpX},
		{"feel.jumpY", t.Feel.JumpY},
		{"feel.landX", t.Feel.LandX},
		{"feel.landY", t.Feel.LandY},
		{"feel.landSpeed", t.Feel.LandSpeed},
		{"feel.recover", t.Feel.Recover},
		{"feel.flashTime", t.Feel.FlashTime},
		{"gun.magazine", float64(g.Magazine)},
		{"gun.fireInterval", g.FireInterval},
		{"gun.thrust", g.Thrust},
		{"gun.bulletSpeed", g.BulletSpeed},
		{"gun.bulletLife", g.BulletLife},
		{"gun.bulletWidth", float64(g.BulletWidth)},
		{"gun.bulletHeight", float64(g.BulletHeight)},
	}
	for _, f := range positive {
		if !(f.v > 0) {
			return fmt.Errorf("%s must be positive, got %v", f.name, f.v)
		}
		if f.v > MaxValue {
			return fmt.Errorf("%s must be at most %g, got %v", f.name, float64(MaxValue), f.v)
		}
	}
	if t.Water.PauseTime < t.Water.RetreatTime {
		// The pause holds the rise while the water retreats; rising during
		// the retreat would be undone by it every step.
		return fmt.Errorf("water.pauseTime must be at least water.retreatTime (%v), got %v",
			t.Water.RetreatTime, t.Water.PauseTime)
	}
	for _, f := range []struct{ name, v string }{
		{"feel.fullColor", t.Feel.FullColor},
		{"feel.emptyColor", t.Feel.EmptyColor},
	} {
		if !isHexColor(f.v) {
			return fmt.Errorf("%s must be #rrggbb, got %q", f.name, f.v)
		}
	}
	c := t.Camera
	switch {
	case !(c.Anchor > 0 && c.Anchor < 1):
		return fmt.Errorf("camera.anchor must be between 0 and 1, got %v", c.Anchor)
	case !(c.Lookahead >= 0):
		return fmt.Errorf("camera.lookahead must not be negative, got %v", c.Lookahead)
	case c.Lookahead > MaxValue:
		return fmt.Errorf("camera.lookahead must be at most %g, got %v", float64(MaxValue), c.Lookahead)
	case !(c.RemainPerSecond > 0 && c.RemainPerSecond <= MaxCameraRemain):
		return fmt.Errorf("camera.remainPerSecond must be above 0 and at most %v, got %v", MaxCameraRemain, c.RemainPerSecond)
	case t.Tower.Base == "":
		return fmt.Errorf("tower.base is missing")
	case t.Tower.Length < 1 || t.Tower.Length > MaxTowerLength:
		return fmt.Errorf("tower.length must be 1 to %d, got %d", MaxTowerLength, t.Tower.Length)
	case slices.Contains(t.Tower.Pool, ""):
		return fmt.Errorf("tower.pool has an empty chunk name")
	}
	if err := validateBlocks(t.Blocks); err != nil {
		return err
	}
	return validateEnemies(t.Enemies)
}

func validateEnemies(enemies []Enemy) error {
	names := map[string]bool{}
	for i, e := range enemies {
		where := fmt.Sprintf("enemies[%d]", i)
		switch {
		case e.Name == "":
			return fmt.Errorf("%s.name is missing", where)
		case names[e.Name]:
			return fmt.Errorf("%s.name %q is defined twice", where, e.Name)
		case e.Width <= 0 || e.Height <= 0:
			return fmt.Errorf("%s size must be positive, got %dx%d", where, e.Width, e.Height)
		case e.Width > MaxValue || e.Height > MaxValue:
			return fmt.Errorf("%s size must be at most %g, got %dx%d", where, float64(MaxValue), e.Width, e.Height)
		case e.HP <= 0:
			return fmt.Errorf("%s.hp must be positive, got %d", where, e.HP)
		case e.HP > MaxValue:
			return fmt.Errorf("%s.hp must be at most %g, got %d", where, float64(MaxValue), e.HP)
		case !(e.Speed >= 0):
			return fmt.Errorf("%s.speed must not be negative, got %v", where, e.Speed)
		case e.Speed > MaxValue:
			return fmt.Errorf("%s.speed must be at most %g, got %v", where, float64(MaxValue), e.Speed)
		case !isHexColor(e.Color):
			return fmt.Errorf("%s.color must be #rrggbb, got %q", where, e.Color)
		}
		names[e.Name] = true
	}
	return nil
}

func validateBlocks(blocks []Block) error {
	values, names := map[int]bool{}, map[string]bool{}
	for i, b := range blocks {
		where := fmt.Sprintf("blocks[%d]", i)
		switch {
		case b.Value < 1 || b.Value > 255:
			return fmt.Errorf("%s.value must be 1 to 255, got %d", where, b.Value)
		case values[b.Value]:
			return fmt.Errorf("%s.value %d is defined twice", where, b.Value)
		case b.Name == "":
			return fmt.Errorf("%s.name is missing", where)
		case names[b.Name]:
			return fmt.Errorf("%s.name %q is defined twice", where, b.Name)
		case !isHexColor(b.Color):
			return fmt.Errorf("%s.color must be #rrggbb, got %q", where, b.Color)
		}
		values[b.Value], names[b.Name] = true, true
	}
	return nil
}

func isHexColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	_, err := strconv.ParseUint(s[1:], 16, 32)
	return err == nil
}

// Fingerprint returns a short hash of the values, so a replay can tell
// whether it is played back with the tuning it was recorded with.
func (t Tuning) Fingerprint() string {
	data, err := json.Marshal(t)
	if err != nil {
		panic(fmt.Sprintf("tuning: marshal: %v", err))
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}
