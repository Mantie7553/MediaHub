package users

import "testing"

func TestValidPreferences(t *testing.T) {
	tests := []struct {
		p  preferences
		ok bool
	}{
		{preferences{"jpn", "eng"}, true},
		{preferences{"default", "off"}, true},
		{preferences{"eng", "off"}, true},
		{preferences{"", "eng"}, false},
		{preferences{"jpn", ""}, false},
		{preferences{"off", "eng"}, false},
		{preferences{"jpn", "default"}, false},
		{preferences{"JPN", "eng"}, false},
		{preferences{"japanese", "eng"}, false},
	}
	for _, tt := range tests {
		if got := validPreferences(tt.p); got != tt.ok {
			t.Errorf("validPreferences(%+v) = %v, want %v", tt.p, got, tt.ok)
		}
	}
}
