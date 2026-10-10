package tuning

import (
	"strings"
	"testing"
)

func TestShippedFileLoads(t *testing.T) {
	if _, err := Load("../../data/tuning.json"); err != nil {
		t.Fatal(err)
	}
}

const valid = `{"player": {
	"width": 12, "height": 20,
	"gravity": 1800, "maxFall": 320,
	"jumpSpeed": 210, "jumpHoldTime": 0.2,
	"coyoteTime": 0.1, "jumpBufferTime": 0.08,
	"apexGravThreshold": 80, "apexGravMult": 0.5,
	"runSpeed": 180, "runAccel": 2000, "airAccelMult": 0.65,
	"oneWayAssist": 80, "cornerCorrection": 4, "drillBounce": 240,
	"stompSpeed": 280, "stompHoldTime": 0.2,
	"maxHP": 4, "knockbackX": 180, "knockbackY": 190, "invulnTime": 1.5
}, "gun": {
	"magazine": 8, "fireInterval": 0.1, "thrust": 240,
	"bulletSpeed": 480, "bulletLife": 0.2, "bulletWidth": 4, "bulletHeight": 8
}, "camera": {
	"anchor": 0.667, "lookahead": 0.2, "remainPerSecond": 0.01
}, "water": {
	"speed": 30, "baseline": 24, "maxLag": 48, "maxMult": 3, "slowRange": 64, "minMult": 0.5,
	"startBelow": 32, "bounce": 400, "retreat": 96, "retreatTime": 0.4, "pauseTime": 0.5
}, "feel": {
	"freezeTime": 0.05, "shakeTime": 0.2, "shakeInterval": 0.04, "shakeScale": 20,
	"stompShakeTime": 0.16, "stompShakeScale": 12,
	"jumpX": 0.6, "jumpY": 1.4, "landX": 1.6, "landY": 0.4, "landSpeed": 480, "recover": 1.75,
	"fullColor": "#ff9a3c", "emptyColor": "#3c78ff", "flashTime": 0.12
}, "tower": {
	"base": "Start", "pool": ["Shaft", "Blocks"], "length": 6
}, "lab": {
	"rows": 300, "enemy": "Floater", "spacing": 96, "jitter": 16
}, "blocks": [
	{"value": 1, "name": "solid", "color": "#707070"},
	{"value": 3, "name": "drill", "drill": true, "color": "#C06040"}
], "enemies": [
	{"name": "Floater", "width": 14, "height": 12, "hp": 2, "stompable": true, "speed": 30, "color": "#60b060"},
	{"name": "Spiker", "width": 14, "height": 14, "hp": 3, "speed": 0, "color": "#c04060"}
]}`

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr string
	}{
		{"valid", valid, ""},
		{"trailing whitespace", valid + "\n\t \n", ""},
		{"trailing garbage", valid + " garbage", "after the tuning object"},
		{"second object", valid + valid, "after the tuning object"},
		{"unknown field", strings.Replace(valid, `"gravity"`, `"gravty"`, 1), "unknown field"},
		{"missing field", strings.Replace(valid, `"maxFall": 320,`, ``, 1), "player.maxFall must be positive"},
		{"negative value", strings.Replace(valid, `"runSpeed": 180`, `"runSpeed": -180`, 1), "player.runSpeed must be positive"},
		{"zero size", strings.Replace(valid, `"width": 12`, `"width": 0`, 1), "player.width must be positive"},
		{"missing gun field", strings.Replace(valid, `"magazine": 8, `, ``, 1), "gun.magazine must be positive"},
		{"anchor out of range", strings.Replace(valid, `"anchor": 0.667`, `"anchor": 1`, 1), "camera.anchor must be between 0 and 1"},
		{"negative lookahead", strings.Replace(valid, `"lookahead": 0.2`, `"lookahead": -0.2`, 1), "camera.lookahead must not be negative"},
		{"zero lookahead", strings.Replace(valid, `"lookahead": 0.2`, `"lookahead": 0`, 1), ""},
		{"remain of 1", strings.Replace(valid, `"remainPerSecond": 0.01`, `"remainPerSecond": 1`, 1), "camera.remainPerSecond must be above 0 and at most 0.99"},
		{"remain of 0.99", strings.Replace(valid, `"remainPerSecond": 0.01`, `"remainPerSecond": 0.99`, 1), ""},
		{"no tower base", strings.Replace(valid, `"base": "Start", `, ``, 1), "tower.base is missing"},
		{"zero tower length", strings.Replace(valid, `"length": 6`, `"length": 0`, 1), "tower.length must be 1 to 1000"},
		{"huge tower length", strings.Replace(valid, `"length": 6`, `"length": 2147483647`, 1), "tower.length must be 1 to 1000"},
		{"empty pool name", strings.Replace(valid, `["Shaft", "Blocks"]`, `["Shaft", ""]`, 1), "tower.pool has an empty chunk name"},
		{"retreat longer than the pause", strings.Replace(valid, `"pauseTime": 0.5`, `"pauseTime": 0.3`, 1), "water.pauseTime must be at least water.retreatTime"},
		{"huge value", strings.Replace(valid, `"speed": 30, `, `"speed": 1e308, `, 1), "water.speed must be at most 1e+06"},
		{"huge lookahead", strings.Replace(valid, `"lookahead": 0.2`, `"lookahead": 1e308`, 1), "camera.lookahead must be at most 1e+06"},
		{"huge enemy speed", strings.Replace(valid, `"stompable": true, "speed": 30`, `"stompable": true, "speed": 1e308`, 1), "enemies[0].speed must be at most 1e+06"},
		{"huge enemy size", strings.Replace(valid, `"width": 14, "height": 14`, `"width": 2000000, "height": 14`, 1), "enemies[1] size must be at most 1e+06"},
		{"huge enemy hp", strings.Replace(valid, `"hp": 3`, `"hp": 2000000`, 1), "enemies[1].hp must be at most 1e+06"},
		{"missing freeze time", strings.Replace(valid, `"freezeTime": 0.05, `, ``, 1), "feel.freezeTime must be positive"},
		{"bad lamp color", strings.Replace(valid, `"#3c78ff"`, `"blue"`, 1), "feel.emptyColor must be #rrggbb"},
		{"lab without enemy", strings.Replace(valid, `"enemy": "Floater", `, ``, 1), "lab.enemy is missing"},
		{"lab jitter too big", strings.Replace(valid, `"jitter": 16`, `"jitter": 48`, 1), "lab.jitter must be"},
		{"lab too short", strings.Replace(valid, `"rows": 300`, `"rows": 3`, 1), "lab.rows must be"},
		{"missing water speed", strings.Replace(valid, `"speed": 30, `, ``, 1), "water.speed must be positive"},
		{"missing max HP", strings.Replace(valid, `"maxHP": 4, `, ``, 1), "player.maxHP must be positive"},
		{"enemy name twice", strings.Replace(valid, `"name": "Spiker"`, `"name": "Floater"`, 1), `enemies[1].name "Floater" is defined twice`},
		{"enemy without name", strings.Replace(valid, `"name": "Spiker", `, ``, 1), "enemies[1].name is missing"},
		{"enemy zero size", strings.Replace(valid, `"width": 14, "height": 14`, `"width": 0, "height": 14`, 1), "enemies[1] size must be positive"},
		{"enemy zero hp", strings.Replace(valid, `"hp": 3`, `"hp": 0`, 1), "enemies[1].hp must be positive"},
		{"enemy negative speed", strings.Replace(valid, `"stompable": true, "speed": 30`, `"stompable": true, "speed": -30`, 1), "enemies[0].speed must not be negative"},
		{"enemy bad color", strings.Replace(valid, `"#c04060"`, `"red"`, 1), "enemies[1].color must be #rrggbb"},
		{"block value out of range", strings.Replace(valid, `"value": 3`, `"value": 256`, 1), "blocks[1].value must be 1 to 255"},
		{"block value twice", strings.Replace(valid, `"value": 3`, `"value": 1`, 1), "blocks[1].value 1 is defined twice"},
		{"block name twice", strings.Replace(valid, `"name": "drill"`, `"name": "solid"`, 1), `blocks[1].name "solid" is defined twice`},
		{"block name missing", strings.Replace(valid, `"name": "drill", `, ``, 1), "blocks[1].name is missing"},
		{"bad block color", strings.Replace(valid, `"#C06040"`, `"#C0604"`, 1), "blocks[1].color must be #rrggbb"},
		{"unknown block field", strings.Replace(valid, `"drill": true`, `"dril": true`, 1), "unknown field"},
		{"not json", `{`, "unexpected EOF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse([]byte(tt.json))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.Player.Gravity != 1800 || got.Player.AirAccelMult != 0.65 {
					t.Fatalf("parsed values wrong: %+v", got.Player)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestFingerprint(t *testing.T) {
	a, err := Parse([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	// Same values written differently.
	b, err := Parse([]byte(strings.ReplaceAll(valid, ", ", ",\n  ")))
	if err != nil {
		t.Fatal(err)
	}
	if a.Fingerprint() != b.Fingerprint() {
		t.Error("same values give different fingerprints")
	}
	c := a
	c.Player.JumpSpeed++
	if a.Fingerprint() == c.Fingerprint() {
		t.Error("changed value gives the same fingerprint")
	}
	if len(a.Fingerprint()) != 16 {
		t.Errorf("fingerprint %q, want 16 hex digits", a.Fingerprint())
	}
}

func TestBlockRGB(t *testing.T) {
	r, g, b := Block{Color: "#C06040"}.RGB()
	if r != 0xc0 || g != 0x60 || b != 0x40 {
		t.Fatalf("RGB() = %x %x %x, want c0 60 40", r, g, b)
	}
}

func TestCloneSharesNoSlices(t *testing.T) {
	a, err := Parse([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	b := a.Clone()
	b.Blocks[0].Name = "x"
	b.Tower.Pool[0] = "x"
	if a.Blocks[0].Name == "x" || a.Tower.Pool[0] == "x" {
		t.Fatal("changing the clone changed the original")
	}
}
