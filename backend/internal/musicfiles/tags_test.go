package musicfiles

import (
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSplitGenres(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"Rock", []string{"Rock"}},
		{"J-Pop; Anime", []string{"J-Pop", "Anime"}},
		{"Rock/Pop, Indie", []string{"Rock", "Pop", "Indie"}},
		{"Rock\x00Metal\x00", []string{"Rock", "Metal"}},
		{"(17)", nil},
		{"Pop; pop ;POP", []string{"Pop"}},
	}
	for _, tt := range tests {
		if got := SplitGenres(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("SplitGenres(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseDuration(t *testing.T) {
	for in, want := range map[string]int{"215.4\n": 215, "215.5": 216, "N/A": 0, "": 0, "-1": 0} {
		if got := parseDuration(in); got != want {
			t.Errorf("parseDuration(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestCleanAndYear(t *testing.T) {
	if got := clean("  Title\x00\x00"); got != "Title" {
		t.Errorf("clean = %q", got)
	}
	if validYear(2019) != 2019 || validYear(0) != 0 || validYear(20190) != 0 {
		t.Error("validYear")
	}
}

func TestReadTagsAndDuration(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}
	path := filepath.Join(t.TempDir(), "track.mp3")
	out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=3",
		"-metadata", "title=Sparkle", "-metadata", "artist=RADWIMPS", "-metadata", "album=Your Name",
		"-metadata", "track=4/27", "-metadata", "date=2016", "-metadata", "genre=J-Rock; Soundtrack",
		"-id3v2_version", "3", path).CombinedOutput()
	if err != nil {
		t.Fatalf("ffmpeg: %v %s", err, out)
	}

	tags, ok := ReadTags(path)
	if !ok {
		t.Fatal("no tags read")
	}
	want := Tags{Title: "Sparkle", Artist: "RADWIMPS", Album: "Your Name", Track: 4, Year: 2016, Genres: []string{"J-Rock", "Soundtrack"}}
	if !reflect.DeepEqual(tags, want) {
		t.Errorf("ReadTags = %+v, want %+v", tags, want)
	}
	if d := Duration(path); d != 3 {
		t.Errorf("Duration = %d, want 3", d)
	}
	if _, ok := ReadTags(filepath.Join(t.TempDir(), "missing.mp3")); ok {
		t.Error("a missing file read as tagged")
	}
}
