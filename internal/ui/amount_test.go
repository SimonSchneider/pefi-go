package ui

import (
	"fmt"
	"math"
	"testing"
)

func TestParseAmount_PlainNumbers(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"0", 0},
		{"123", 123},
		{"1.5", 1.5},
		{"-42", -42},
		{"-0.5", -0.5},
		{"1e-10", 1e-10},
		{"1e2", 100},
		{"  99  ", 99},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("expression %q = %g", tt.in, tt.want), func(t *testing.T) {
			got, err := ParseAmount(tt.in)
			if err != nil {
				t.Fatalf("ParseAmount(%q) err = %v", tt.in, err)
			}
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("ParseAmount(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseAmount_Expressions(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"1+2", 3},
		{"10-3", 7},
		{"2*3", 6},
		{"10/2", 5},
		{"1+2*3", 7},
		{"2*3+4", 10},
		{"10/2+3", 8},
		{"500+23+43", 566},
		{"500+23+43-294*1.23", 500 + 23 + 43 - 294*1.23},
		{"(1+2)*3", 9},
		{"2*(3+4)", 14},
		{"(100+50)*1.23", 184.5},
		{"((1+2)*3)+4", 13},
		{"-5", -5},
		{"-5+3", -2},
		{"5*-2", -10},
		{" 500 + 23 ", 523},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("expression %q = %g", tt.in, tt.want), func(t *testing.T) {
			got, err := ParseAmount(tt.in)
			if err != nil {
				t.Fatalf("ParseAmount(%q) err = %v", tt.in, err)
			}
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("ParseAmount(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseAmount_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"   ",
		"abc",
		"1+",
		"1++2",
		"(1",
		"1)",
		")(",
		"1/0",
	}
	for _, in := range invalid {
		t.Run(fmt.Sprintf("invalid %q", in), func(t *testing.T) {
			_, err := ParseAmount(in)
			if err == nil {
				t.Error("ParseAmount expected error, got nil")
			}
		})
	}
}

func TestParseAmount_ScientificNotationNotExpression(t *testing.T) {
	// 1e-10 must parse as single number, not 1e minus 10
	got, err := ParseAmount("1e-10")
	if err != nil {
		t.Fatalf("ParseAmount(1e-10) err = %v", err)
	}
	want := 1e-10
	if math.Abs(got-want) > 1e-20 {
		t.Errorf("ParseAmount(1e-10) = %v, want %v", got, want)
	}
}

func TestParseAmount_DivisionByZero(t *testing.T) {
	_, err := ParseAmount("1/0")
	if err == nil {
		t.Error("ParseAmount(1/0) expected error (division by zero)")
	}
}
