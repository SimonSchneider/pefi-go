package uncertain

import (
	"fmt"
	"math"
	"math/rand"
)

type Config struct {
	rng     *rand.Rand
	samples int
}

func NewConfig(seed int64, samples int) *Config {
	return &Config{
		rng:     rand.New(rand.NewSource(seed)),
		samples: samples,
	}
}

type DistributionType string

const (
	DistFixed      DistributionType = "fixed"
	DistTriangular DistributionType = "triangular"
	DistEmpirical  DistributionType = "empirical"
	DistUniform    DistributionType = "uniform"
	DistNormal     DistributionType = "normal"
)

type Value struct {
	Distribution DistributionType
	Parameters   map[string]float64
	Samples      []float64 // Only for DistEmpirical
}

func NewFixed(value float64) Value {
	return Value{
		Distribution: DistFixed,
		Parameters: map[string]float64{
			"value": value,
		},
	}
}

func NewUniform(min, max float64) Value {
	return Value{
		Distribution: DistUniform,
		Parameters: map[string]float64{
			"min": min,
			"max": max,
		},
	}
}

func (u Value) String() string {
	switch u.Distribution {
	case DistFixed:
		return "Fixed(" + fmt.Sprintf("%f", u.Parameters["value"]) + ")"
	case DistTriangular:
		return "Triangular(" + fmt.Sprintf("%f, %f, %f", u.Parameters["min"], u.Parameters["mode"], u.Parameters["max"]) + ")"
	case DistUniform:
		return "Uniform(" + fmt.Sprintf("%f, %f", u.Parameters["min"], u.Parameters["max"]) + ")"
	case DistNormal:
		return "Normal(" + fmt.Sprintf("%f, %f", u.Parameters["mean"], u.Parameters["stddev"]) + ")"
	case DistEmpirical:
		return fmt.Sprintf("Empirical(%f, %d samples)", u.Mean(), len(u.Samples))
	default:
		return "Unknown Distribution"
	}
}

func (u Value) Mean() float64 {
	switch u.Distribution {
	case DistFixed:
		return u.Parameters["value"]
	case DistTriangular:
		return u.Parameters["mode"]
	case DistUniform:
		mi := u.Parameters["min"]
		ma := u.Parameters["max"]
		return (mi + ma) / 2
	case DistNormal:
		return u.Parameters["mean"]
	case DistEmpirical:
		if len(u.Samples) == 0 {
			return 0
		}
		sum := 0.0
		for _, sample := range u.Samples {
			sum += sample
		}
		return sum / float64(len(u.Samples))
	default:
		return 0
	}
}

func (u Value) Sample(ucfg *Config) float64 {
	switch u.Distribution {
	case DistFixed:
		return u.Parameters["value"]
	case DistTriangular:
		mi := u.Parameters["min"]
		mode := u.Parameters["mode"]
		ma := u.Parameters["max"]
		uRand := ucfg.rng.Float64()
		c := (mode - mi) / (ma - mi)
		if uRand < c {
			return mi + (ma-mi)*math.Sqrt(uRand*c)
		} else {
			return ma - (ma-mi)*math.Sqrt((1-uRand)*(1-c))
		}
	case DistUniform:
		mi := u.Parameters["min"]
		ma := u.Parameters["max"]
		return mi + ucfg.rng.Float64()*(ma-mi)
	case DistNormal:
		mean := u.Parameters["mean"]
		stddev := u.Parameters["stddev"]
		return ucfg.rng.NormFloat64()*stddev + mean
	case DistEmpirical:
		if len(u.Samples) == 0 {
			return 0
		}
		return u.Samples[ucfg.rng.Intn(len(u.Samples))]
	default:
		return 0
	}
}

func (u Value) isFixed() bool {
	return u.Distribution == DistFixed
}

func (u Value) operate(cfg *Config, v Value, op func(a, b float64) float64) Value {
	// Both fixed: operate directly
	if u.isFixed() && v.isFixed() {
		result := op(u.Parameters["value"], v.Parameters["value"])
		return NewFixed(result)
	}

	// One fixed: sample the other and apply op
	if u.isFixed() {
		return v.sampleWithFixed(cfg, u.Parameters["value"], func(b, a float64) float64 { return op(a, b) })
	}
	if v.isFixed() {
		return u.sampleWithFixed(cfg, v.Parameters["value"], op)
	}

	// Both variable: sample both
	samples := min(len(u.Samples), len(v.Samples), cfg.samples)
	res := make([]float64, samples)
	for i := 0; i < samples; i++ {
		res[i] = op(u.Sample(cfg), v.Sample(cfg))
	}
	return Value{
		Distribution: DistEmpirical,
		Samples:      res,
	}
}

func (u Value) sampleWithFixed(cfg *Config, fixed float64, op func(a, b float64) float64) Value {
	res := make([]float64, cfg.samples)
	for i := 0; i < cfg.samples; i++ {
		res[i] = op(u.Sample(cfg), fixed)
	}
	return Value{
		Distribution: DistEmpirical,
		Samples:      res,
	}
}

func (u Value) ApplyFixed(cfg *Config, fixed float64, op func(a, b float64) float64) Value {
	if u.isFixed() {
		return NewFixed(op(u.Parameters["value"], fixed))
	}
	return u.sampleWithFixed(cfg, fixed, op)
}

func (u Value) Add(cfg *Config, v Value) Value {
	return u.operate(cfg, v, func(a, b float64) float64 { return a + b })
}

func (u Value) Sub(cfg *Config, v Value) Value {
	return u.operate(cfg, v, func(a, b float64) float64 { return a - b })
}

func (u Value) Mul(cfg *Config, v Value) Value {
	return u.operate(cfg, v, func(a, b float64) float64 { return a * b })
}

func (u Value) Pow(cfg *Config, v Value) Value {
	return u.operate(cfg, v, func(a, b float64) float64 {
		// Simple error handling: avoid complex numbers or invalid inputs
		if a < 0 && b != math.Trunc(b) {
			return 0 // Or NaN or panic
		}
		return math.Pow(a, b)
	})
}
