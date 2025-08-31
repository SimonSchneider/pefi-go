package ui

import (
	"fmt"
	"strconv"
	"unicode"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type UncertainValue struct {
	Distribution uncertain.DistributionType `json:"distribution"`
	Parameters   map[string]float64         `json:"parameters"`
	Samples      []float64                  `json:"samples"`
}

func ParseUncertainValue(val string) (uncertain.Value, error) {
	if unicode.IsDigit(rune(val[0])) || (len(val) > 1 && val[0] == '-' && unicode.IsDigit(rune(val[1]))) {
		// If the value is a simple float, return a fixed uncertain value
		f, err := shttp.ParseFloat(val)
		if err != nil {
			return uncertain.Value{}, fmt.Errorf("parsing float: %w", err)
		}
		return uncertain.NewFixed(f), nil
	}
	// Otherwise, parse it as an uncertain value
	var value uncertain.Value
	if err := value.Decode(val); err != nil {
		return uncertain.Value{}, fmt.Errorf("decoding uncertain value: %w", err)
	}
	return value, nil
}

func ParseHumanNumber[T int | int64 | int32 | float64 | float32](delegate func(string) (T, error)) func(string) (T, error) {
	return func(val string) (T, error) {
		if val == "" {
			return 0, nil
		}
		suffix := val[len(val)-1]
		mult := T(1)
		if suffix == 'k' || suffix == 'K' {
			mult = 1_000
			val = val[:len(val)-1]
		} else if suffix == 'm' || suffix == 'M' {
			mult = 1_000_000
			val = val[:len(val)-1]
		}
		i, err := delegate(val)
		return i * mult, err
	}
}

func ParseInt64(val string) (int64, error) {
	if val == "" {
		return 0, nil
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing int64: %w", err)
	}
	return i, nil
}

func ParseNullableFloat(val string) (*float64, error) {
	if val == "" {
		return nil, nil
	}
	f, err := shttp.ParseFloat(val)
	if err != nil {
		return nil, fmt.Errorf("parsing float: %w", err)
	}
	return &f, nil
}

func WithDefaultNull[T comparable](val T) *T {
	var zero T
	if val == zero {
		return nil
	}
	return &val
}

func OrDefault[T any](val *T) T {
	if val == nil {
		var zero T
		return zero
	}
	return *val
}

func ParseDateCron(val string) (date.Cron, error) {
	return date.Cron(val), nil
}
