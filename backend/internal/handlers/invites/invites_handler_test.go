package invites

import "testing"

func TestExpiryDays(t *testing.T) {
	tests := []struct {
		requested int
		want      int
		ok        bool
	}{
		{0, defaultExpiryDays, true},
		{1, 1, true},
		{30, 30, true},
		{31, 0, false},
		{-1, 0, false},
	}
	for _, tt := range tests {
		got, ok := expiryDays(tt.requested)
		if got != tt.want || ok != tt.ok {
			t.Errorf("expiryDays(%d) = %d, %v; want %d, %v", tt.requested, got, ok, tt.want, tt.ok)
		}
	}
}
