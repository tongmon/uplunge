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
	Tower  Tower   `json:"tower"`
	Blocks []Block `json:"blocks"`
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
	}
	c := t.Camera
	switch {
	case !(c.Anchor > 0 && c.Anchor < 1):
		return fmt.Errorf("camera.anchor must be between 0 and 1, got %v", c.Anchor)
	case !(c.Lookahead >= 0):
		return fmt.Errorf("camera.lookahead must not be negative, got %v", c.Lookahead)
	case !(c.RemainPerSecond > 0 && c.RemainPerSecond < 1):
		return fmt.Errorf("camera.remainPerSecond must be between 0 and 1, got %v", c.RemainPerSecond)
	case t.Tower.Base == "":
		return fmt.Errorf("tower.base is missing")
	case t.Tower.Length < 1:
		return fmt.Errorf("tower.length must be at least 1, got %d", t.Tower.Length)
	case slices.Contains(t.Tower.Pool, ""):
		return fmt.Errorf("tower.pool has an empty chunk name")
	}
	return validateBlocks(t.Blocks)
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
