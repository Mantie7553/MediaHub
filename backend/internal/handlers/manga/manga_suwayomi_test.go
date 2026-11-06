package manga

import "testing"

func TestSameTitle(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Frieren: Beyond Journey's End", "frieren beyond journeys end", true},
		{"Chainsaw Man", "Chainsaw Man 2", false},
		{"", "", false},
		{"!!!", "???", false},
		{"Kaguya-sama", "KAGUYA SAMA", true},
	}
	for _, c := range cases {
		if got := sameTitle(c.a, c.b); got != c.want {
			t.Errorf("sameTitle(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
