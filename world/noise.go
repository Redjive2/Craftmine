package world

import "math"

// hashCoord mixes (seed, x, y) into a deterministic uint64 with good avalanche.
// SplitMix64-style multiplies after xor-folding the inputs. Used as the source
// of pseudo-randomness for value noise and tree placement.
func hashCoord(seed int64, x, y int) uint64 {
	h := uint64(seed)
	h ^= uint64(int64(x)) * 0x9E3779B97F4A7C15
	h ^= uint64(int64(y)) * 0xBF58476D1CE4E5B9
	h ^= h >> 30
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 27
	h *= 0x94D049BB133111EB
	h ^= h >> 31
	return h
}

// random01 maps (seed, x, y) to a deterministic float in [0, 1).
func random01(seed int64, x, y int) float64 {
	return float64(hashCoord(seed, x, y)>>11) / float64(1<<53)
}

// smoothstep is the standard 6t^5 - 15t^4 + 10t^3 polynomial used to blend
// value-noise grid cells without visible grid lines.
func smoothstep(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// valueNoise2D samples 2D value noise at (x, z): four grid corners get
// random01 values, smoothstep-interpolated. Output is in [0, 1).
func valueNoise2D(seed int64, x, z float64) float64 {
	x0, z0 := int(math.Floor(x)), int(math.Floor(z))
	x1, z1 := x0+1, z0+1
	fx, fz := x-float64(x0), z-float64(z0)
	sx, sz := smoothstep(fx), smoothstep(fz)
	n00 := random01(seed, x0, z0)
	n10 := random01(seed, x1, z0)
	n01 := random01(seed, x0, z1)
	n11 := random01(seed, x1, z1)
	return lerp(lerp(n00, n10, sx), lerp(n01, n11, sx), sz)
}

// fbm2D is fractal Brownian motion: several octaves of valueNoise2D summed
// with decaying amplitude. Each octave uses a different seed offset so the
// layers don't constructively reinforce into a tiled look. Output is in [0, 1).
func fbm2D(seed int64, x, z float64, octaves int, baseFrequency, lacunarity, gain float64) float64 {
	if octaves <= 0 {
		return 0
	}
	freq := baseFrequency
	amp := 1.0
	total := 0.0
	ampSum := 0.0
	for octave := 0; octave < octaves; octave++ {
		total += amp * valueNoise2D(seed+int64(octave)*0x1000193, x*freq, z*freq)
		ampSum += amp
		amp *= gain
		freq *= lacunarity
	}
	return total / ampSum
}
