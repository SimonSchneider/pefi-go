package uncertain

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
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
	DistFixed     DistributionType = "fixed"
	DistEmpirical DistributionType = "empirical"
	DistUniform   DistributionType = "uniform"
	DistNormal    DistributionType = "normal"
	DistMapped    DistributionType = "mapped" // For custom sampling functions
)

type Value struct {
	Distribution DistributionType

	Fixed   ParamsFixedValue
	Uniform ParamsUniform
	Normal  ParamsNormal

	Samples   []float64                 // Only for DistEmpirical
	SampleFun func(cfg *Config) float64 // Optional mapping function for custom sampling
}

type ParamsFixedValue struct {
	Value float64
}

type ParamsUniform struct {
	Min float64
	Max float64
}

type ParamsNormal struct {
	Mean   float64
	Stddev float64
}

func NewFixed(value float64) Value {
	return Value{
		Distribution: DistFixed,
		Fixed:        ParamsFixedValue{Value: value},
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
		Uniform:      ParamsUniform{Min: min, Max: max},
	}
}

func NewNormal(mean, stddev float64) Value {
	return Value{
		Distribution: DistNormal,
		Normal:       ParamsNormal{Mean: mean, Stddev: stddev},
	}
}

func (u Value) Valid() bool {
	switch u.Distribution {
	case DistFixed:
		return true
	case DistUniform:
		return u.Uniform.Min < u.Uniform.Max
	case DistNormal:
		return u.Normal.Stddev > 0 // Mean can be any value
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
		return "Fixed(" + fmt.Sprintf("%f", u.Fixed.Value) + ")"
	case DistUniform:
		return "Uniform(" + fmt.Sprintf("%f, %f", u.Uniform.Min, u.Uniform.Max) + ")"
	case DistNormal:
		return "Normal(" + fmt.Sprintf("%f, %f", u.Normal.Mean, u.Normal.Stddev) + ")"
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
		return u.Fixed.Value
	case DistUniform:
		return (u.Uniform.Min + u.Uniform.Max) / 2
	case DistNormal:
		return u.Normal.Mean
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
	switch u.Distribution {
	case DistFixed:
		return func(q float64) float64 {
			return u.Fixed.Value
		}
	case DistEmpirical:
		if len(u.Samples) == 0 {
			panic("not yet implemented")
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
	case DistUniform:
		return func(p float64) float64 {
			if p < 0 || p > 1 {
				panic("Quantiles: p must be in [0, 1]")
			}
			return u.Uniform.Min + p*(u.Uniform.Max-u.Uniform.Min)
		}
	default:
		panic("Unknown Distribution")
	}
}

func (u Value) Sample(ucfg *Config) float64 {
	switch u.Distribution {
	case DistFixed:
		return u.Fixed.Value
	case DistUniform:
		return u.Uniform.Min + ucfg.RNG.Float64()*(u.Uniform.Max-u.Uniform.Min)
	case DistNormal:
		return ucfg.RNG.NormFloat64()*u.Normal.Stddev + u.Normal.Mean
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
		result := op(u.Fixed.Value, v.Fixed.Value)
		return NewFixed(result)
	}

	// One fixed: sample the other and apply op
	if u.isFixed() {
		return v.sampleWithFixed(cfg, u.Fixed.Value, func(b, a float64) float64 { return op(a, b) })
	}
	if v.isFixed() {
		return u.sampleWithFixed(cfg, v.Fixed.Value, op)
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
		return NewFixed(op(u.Fixed.Value, fixed))
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
	return u.Distribution == DistFixed && u.Fixed.Value == 0
}

func (u Value) SimpleEncode() string {
	if u.Distribution == DistFixed {
		return strconv.FormatFloat(u.Fixed.Value, 'f', -1, 64)
	}
	return u.SafeEncode()
}

func (u Value) SafeEncode() string {
	// SafeEncode is a helper to encode the value without error handling
	encoded, err := u.Encode()
	if err != nil {
		return ""
	}
	return encoded
}

func (u Value) Encode() (string, error) {
	switch u.Distribution {
	case DistFixed:
		return fmt.Sprintf("fixed(%s)", strconv.FormatFloat(u.Fixed.Value, 'f', -1, 64)), nil
	case DistUniform:
		return fmt.Sprintf("uniform(%s,%s)", strconv.FormatFloat(u.Uniform.Min, 'f', -1, 64), strconv.FormatFloat(u.Uniform.Max, 'f', -1, 64)), nil
	case DistNormal:
		return fmt.Sprintf("normal(%s,%s)", strconv.FormatFloat(u.Normal.Mean, 'f', -1, 64), strconv.FormatFloat(u.Normal.Stddev, 'f', -1, 64)), nil
	case DistEmpirical:
		return "", fmt.Errorf("empirical distribution cannot be encoded")
	case DistMapped:
		return "", fmt.Errorf("empirical distribution cannot be encoded")
	default:
		return "", fmt.Errorf("unknown distribution type: %s", u.Distribution)
	}
}

func Decode(encoded string) (Value, error) {
	var v Value
	if err := v.Decode(encoded); err != nil {
		return Value{}, fmt.Errorf("decoding value: %w", err)
	}
	return v, nil
}

func (u *Value) Decode(encoded string) error {
	var dist DistributionType
	currIdx := 0
	for currIdx < len(encoded) && encoded[currIdx] != '(' {
		currIdx++
	}
	if currIdx == len(encoded) {
		return fmt.Errorf("invalid encoded string: %s", encoded)
	}
	dist = DistributionType(encoded[:currIdx])
	currIdx++ // Skip the '('
	if currIdx >= len(encoded) {
		return fmt.Errorf("invalid encoded string: %s", encoded)
	}
	switch dist {
	case DistFixed:
		_, value, err := parseFloatUntil(encoded, currIdx, ')')
		if err != nil {
			return fmt.Errorf("parsing fixed value: %w", err)
		}
		*u = NewFixed(value)
		return nil
	case DistUniform:
		currIdx, minVal, err := parseFloatUntil(encoded, currIdx, ',')
		if err != nil {
			return fmt.Errorf("parsing uniform min: %w", err)
		}
		currIdx++ // Skip the ','
		currIdx, maxVal, err := parseFloatUntil(encoded, currIdx, ')')
		if err != nil {
			return fmt.Errorf("parsing uniform max: %w", err)
		}
		v := NewUniform(minVal, maxVal)
		if !v.Valid() {
			return fmt.Errorf("invalid uniform distribution (%f, %f)", minVal, maxVal)
		}
		*u = v
		return nil
	case DistNormal:
		currIdx, mean, err := parseFloatUntil(encoded, currIdx, ',')
		if err != nil {
			return fmt.Errorf("parsing normal mean: %w", err)
		}
		currIdx++ // Skip the ','
		currIdx, stddev, err := parseFloatUntil(encoded, currIdx, ')')
		if err != nil {
			return fmt.Errorf("parsing normal stddev: %w", err)
		}
		v := NewNormal(mean, stddev)
		if !v.Valid() {
			return fmt.Errorf("invalid normal distribution: mean %f, stddev %f", mean, stddev)
		}
		*u = v
		return nil
	}
	return fmt.Errorf("unknown distribution type: %s", dist)
}

func parseFloatUntil(encoded string, start int, match byte) (int, float64, error) {
	currIdx := start
	for currIdx < len(encoded) && encoded[currIdx] != match {
		currIdx++
	}
	if currIdx >= len(encoded) {
		return 0, 0, fmt.Errorf("no number found before %c in %s", match, encoded)
	}
	if currIdx == start {
		return 0, 0, fmt.Errorf("no number found before %c in %s", match, encoded)
	}
	num, err := strconv.ParseFloat(strings.Trim(encoded[start:currIdx], " \n"), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parsing float: %w", err)
	}
	return currIdx, num, nil
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
