package rng

import "testing"

func TestUint64MatchesSplitMix64(t *testing.T) {
	// Reference outputs of SplitMix64 for seed 0. If these change, every
	// recorded tower replay changes with them.
	want := []uint64{0xe220a8397b1dcdaf, 0x6e789e6aa1b965f4, 0x06c45d188009454f}
	r := New(0)
	for i, w := range want {
		if got := r.Uint64(); got != w {
			t.Fatalf("output %d = %#x, want %#x", i, got, w)
		}
	}
}

func TestSameSeedSameSequence(t *testing.T) {
	a, b := New(42), New(42)
	for i := 0; i < 100; i++ {
		if a.IntN(7) != b.IntN(7) {
			t.Fatalf("sequences differ at %d", i)
		}
	}
}

func TestIntNCoversRange(t *testing.T) {
	r := New(1)
	var seen [5]int
	for i := 0; i < 5000; i++ {
		v := r.IntN(5)
		if v < 0 || v >= 5 {
			t.Fatalf("IntN(5) = %d", v)
		}
		seen[v]++
	}
	for v, n := range seen {
		if n < 800 || n > 1200 {
			t.Fatalf("value %d came up %d times in 5000, want about 1000", v, n)
		}
	}
}

func TestIntNPanicsOnZero(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("IntN(0) did not panic")
		}
	}()
	New(0).IntN(0)
}
