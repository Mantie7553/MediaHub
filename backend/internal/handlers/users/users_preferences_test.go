package users

import "testing"

func TestValidPreferences(t *testing.T) {
	tests := []struct {
		p  preferences
		ok bool
	}{
		{preferences{"jpn", "eng", nil}, true},
		{preferences{"default", "off", nil}, true},
		{preferences{"eng", "off", nil}, true},
		{preferences{"", "eng", nil}, false},
		{preferences{"jpn", "", nil}, false},
		{preferences{"off", "eng", nil}, false},
		{preferences{"jpn", "default", nil}, false},
		{preferences{"JPN", "eng", nil}, false},
		{preferences{"japanese", "eng", nil}, false},
	}
	for _, tt := range tests {
		if got := validPreferences(tt.p); got != tt.ok {
			t.Errorf("validPreferences(%+v) = %v, want %v", tt.p, got, tt.ok)
		}
	}
}
