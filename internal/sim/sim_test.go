package sim

import "testing"

func TestStepAdvancesOneTick(t *testing.T) {
	w := NewWorld()
	for i := 0; i < 3*Hz; i++ {
		w.Step(Input{})
	}
	if got, want := w.Tick, uint64(3*Hz); got != want {
		t.Fatalf("Tick = %d, want %d", got, want)
	}
}

func TestDtMatchesHz(t *testing.T) {
	if got := Dt * Hz; got != 1 {
		t.Fatalf("Dt * Hz = %v, want 1", got)
	}
}
