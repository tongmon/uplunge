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
)

// Tuning is the full set of tunables. Every value is a prototype hypothesis
// until the developer confirms it in docs/design.md.
type Tuning struct {
	Player Player `json:"player"`
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

	RunSpeed float64 `json:"runSpeed"`
	RunAccel float64 `json:"runAccel"`
	// AirAccelMult scales RunAccel while airborne.
	AirAccelMult float64 `json:"airAccelMult"`
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
	p := t.Player
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
		{"player.runSpeed", p.RunSpeed},
		{"player.runAccel", p.RunAccel},
		{"player.airAccelMult", p.AirAccelMult},
	}
	for _, f := range positive {
		if !(f.v > 0) {
			return fmt.Errorf("%s must be positive, got %v", f.name, f.v)
		}
	}
	return nil
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
