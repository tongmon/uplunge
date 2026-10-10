// Package rng is the seeded random number generator for game logic. It is
// SplitMix64, written out here so a seed gives the same sequence on every Go
// version and platform, which replays depend on. It must not depend on
// Ebitengine.
package rng

// Rand is a deterministic random number generator. The zero value is a valid
// generator with seed 0.
type Rand struct {
	state uint64
}

// New returns a generator seeded with seed.
func New(seed uint64) *Rand {
	return &Rand{state: seed}
}

// Uint64 returns the next 64 random bits.
func (r *Rand) Uint64() uint64 {
	r.state += 0x9e3779b97f4a7c15
	z := r.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// IntN returns a uniform random number in [0, n). It panics if n <= 0.
func (r *Rand) IntN(n int) int {
	if n <= 0 {
		panic("rng: IntN with n <= 0")
	}
	// Reject the top partial range so every result is equally likely.
	limit := ^uint64(0) - ^uint64(0)%uint64(n)
	for {
		if v := r.Uint64(); v < limit {
			return int(v % uint64(n))
		}
	}
}

// Bool returns a random bool.
func (r *Rand) Bool() bool {
	return r.Uint64()&1 == 1
}
