package uncertain

import (
	"math"
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		encoded string
		want    Value
		wantErr bool
	}{
		{"fixed(1)", NewFixed(1), false},
		{"fixed(1.5)", NewFixed(1.5), false},
		{"fixed(-1.5)", NewFixed(-1.5), false},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.encoded, func(t *testing.T) {
			got, err := Decode(tt.encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPowNegativeBaseNonIntegerExponentPanics(t *testing.T) {
	cfg := NewConfig(42, 1)
	base := NewFixed(-2.0)
	exp := NewFixed(0.5) // non-integer exponent

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for negative base with non-integer exponent, but did not panic")
		}
	}()

	// Force sampling to trigger the panic
	_ = base.Pow(cfg, exp).Sample(cfg)
}

func TestMappedMean(t *testing.T) {
	base := NewFixed(100)
	mapped := NewMapped(func(cfg *Config) float64 {
		return base.Sample(cfg) * 2
	})
	got := mapped.Mean()
	if math.Abs(got-200) > 0.01 {
		t.Errorf("Mean() = %v, want 200", got)
	}
}

func TestMappedMeanWithUncertain(t *testing.T) {
	base := NewUniform(90, 110)
	mapped := NewMapped(func(cfg *Config) float64 {
		return base.Sample(cfg) * 2
	})
	got := mapped.Mean()
	if math.Abs(got-200) > 5 {
		t.Errorf("Mean() = %v, want ~200", got)
	}
}
