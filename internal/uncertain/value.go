package uncertain

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
)

type Config struct {
	RNG     *rand.Rand
	Samples int
}

func NewConfig(seed int64, samples int) *Config {
	return &Config{
		RNG:     rand.New(rand.NewSource(seed)),
		Samples: samples,
	}
}

type DistributionType string

const (
	DistFixed      DistributionType = "fixed"
	DistTriangular DistributionType = "triangular"
	DistEmpirical  DistributionType = "empirical"
	DistUniform    DistributionType = "uniform"
	DistNormal     DistributionType = "normal"
	DistMapped     DistributionType = "mapped" // For custom sampling functions
)

type Value struct {
	Distribution DistributionType
	Parameters   map[string]float64
	Samples      []float64                 // Only for DistEmpirical
	SampleFun    func(cfg *Config) float64 // Optional mapping function for custom sampling
}

func NewFixed(value float64) Value {
	return Value{
		Distribution: DistFixed,
		Parameters: map[string]float64{
			"value": value,
		},
	}
}

func NewMapped(sampleFun func(cfg *Config) float64) Value {
	return Value{
		Distribution: DistMapped,
		SampleFun:    sampleFun,
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

func NewNormal(mean, stddev float64) Value {
	return Value{
		Distribution: DistNormal,
		Parameters: map[string]float64{
			"mean":   mean,
			"stddev": stddev,
		},
	}
}

func (u Value) Valid() bool {
	switch u.Distribution {
	case DistFixed:
		if _, ok := u.Parameters["value"]; !ok || math.IsNaN(u.Parameters["value"]) {
			return false
		}
		return true
	case DistTriangular:
		return u.Parameters["min"] < u.Parameters["mode"] && u.Parameters["mode"] < u.Parameters["max"]
	case DistUniform:
		return u.Parameters["min"] < u.Parameters["max"]
	case DistNormal:
		return u.Parameters["stddev"] > 0 // Mean can be any value
	case DistEmpirical:
		return len(u.Samples) > 0 // Must have samples
	case DistMapped:
		return u.SampleFun != nil
	default:
		return false
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
		q := u.Quantiles()
		q1, q9 := q(0.05), q(0.95)
		return fmt.Sprintf("Empirical(%f [%f], %d samples)", u.Mean(), q9-q1, len(u.Samples))
	case DistMapped:
		return "Mapped(defined by custom sampling function)"
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
	case DistMapped:
		// TODO: mean will have to take *config as an argument for sampling
		panic("not yet implemented: Mapped distribution does not have a defined mean")
	default:
		return 0
	}
}

// Quantiles returns the func to calculate the p-th quantile (0 <= p <= 1) for empirical distributions using linear interpolation.
func (u Value) Quantiles() func(q float64) float64 {
	if u.Distribution == DistFixed {
		return func(q float64) float64 {
			return 0
		}
	}
	if u.Distribution != DistEmpirical || len(u.Samples) == 0 {
		panic("not yet implemented '" + u.Distribution + "' does not have a defined quantile function")
		return func(q float64) float64 {
			return 0 // Not applicable for non-empirical distributions or empty samples
		}
	}
	sorted := make([]float64, len(u.Samples))
	copy(sorted, u.Samples)
	sort.Float64s(sorted)
	n := float64(len(sorted))
	return func(p float64) float64 {
		pos := p * (n - 1)
		lower := int(math.Floor(pos))
		upper := int(math.Ceil(pos))
		if lower == upper {
			return sorted[lower]
		}
		weight := pos - float64(lower)
		return sorted[lower]*(1-weight) + sorted[upper]*weight
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
		uRand := ucfg.RNG.Float64()
		c := (mode - mi) / (ma - mi)
		if uRand < c {
			return mi + (ma-mi)*math.Sqrt(uRand*c)
		} else {
			return ma - (ma-mi)*math.Sqrt((1-uRand)*(1-c))
		}
	case DistUniform:
		mi := u.Parameters["min"]
		ma := u.Parameters["max"]
		return mi + ucfg.RNG.Float64()*(ma-mi)
	case DistNormal:
		mean := u.Parameters["mean"]
		stddev := u.Parameters["stddev"]
		return ucfg.RNG.NormFloat64()*stddev + mean
	case DistEmpirical:
		if len(u.Samples) == 0 {
			return 0
		}
		return u.Samples[ucfg.RNG.Intn(len(u.Samples))]
	case DistMapped:
		return u.SampleFun(ucfg)
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
	res := make([]float64, cfg.Samples)
	for i := 0; i < cfg.Samples; i++ {
		res[i] = op(u.Sample(cfg), v.Sample(cfg))
	}
	return Value{
		Distribution: DistEmpirical,
		Samples:      res,
	}
}

func (u Value) sampleWithFixed(cfg *Config, fixed float64, op func(a, b float64) float64) Value {
	res := make([]float64, cfg.Samples)
	for i := 0; i < cfg.Samples; i++ {
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

func (u Value) Exp() Value {
	return NewMapped(func(cfg *Config) float64 {
		return math.Exp(u.Sample(cfg))
	})
}

func (u Value) Pow(cfg *Config, v Value) Value {
	return u.operate(cfg, v, func(a, b float64) float64 {
		// Simple error handling: avoid complex numbers or invalid inputs
		if a < 0 && b != math.Trunc(b) {
			fmt.Printf("Warning: Attempting to raise negative base %f to non-integer exponent %f\n", a, b)
			//panic("bla")
			return 0 // Or NaN or panic
		}
		return math.Pow(a, b)
	})
}

func (u Value) Zero() bool {
	if u.Distribution == "" {
		return true
	}
	return u.Distribution == DistFixed && u.Parameters["value"] == 0
}

type ContextKey struct{}

var ctxKey = ContextKey{}

func WithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, ctxKey, cfg)
}

func GetConfig(ctx context.Context) *Config {
	cfg, ok := ctx.Value(ctxKey).(*Config)
	if !ok {
		panic("Uncertain config not found in context")
	}
	return cfg
}
