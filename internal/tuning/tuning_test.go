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
	"runSpeed": 180, "runAccel": 2000, "airAccelMult": 0.65
}}`

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
