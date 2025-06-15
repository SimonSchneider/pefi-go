package uncertain

import (
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
