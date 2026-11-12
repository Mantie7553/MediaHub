package users

import "testing"

func TestValidPreferences(t *testing.T) {
	tests := []struct {
		p  preferences
		ok bool
	}{
		{preferences{"jpn", "eng", nil, nil}, true},
		{preferences{"default", "off", nil, nil}, true},
		{preferences{"eng", "off", nil, nil}, true},
		{preferences{"", "eng", nil, nil}, false},
		{preferences{"jpn", "", nil, nil}, false},
		{preferences{"off", "eng", nil, nil}, false},
		{preferences{"jpn", "default", nil, nil}, false},
		{preferences{"JPN", "eng", nil, nil}, false},
		{preferences{"japanese", "eng", nil, nil}, false},
		{preferences{"jpn", "eng", nil, ptr("overview")}, true},
		{preferences{"jpn", "eng", nil, ptr("music")}, true},
		{preferences{"jpn", "eng", nil, ptr("light_novel")}, true},
		{preferences{"jpn", "eng", nil, ptr("")}, false},
		{preferences{"jpn", "eng", nil, ptr("music_album")}, false},
		{preferences{"jpn", "eng", nil, ptr("Music")}, false},
	}
	for _, tt := range tests {
		if got := validPreferences(tt.p); got != tt.ok {
			t.Errorf("validPreferences(%+v) = %v, want %v", tt.p, got, tt.ok)
		}
	}
}

func ptr(s string) *string { return &s }
