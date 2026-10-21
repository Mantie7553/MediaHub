package deletion

import (
	"errors"
	"net/http"
	"testing"
)

type fakeEpisodes struct {
	calls  []string
	fileID int
	failOn string
}

func (f *fakeEpisodes) step(name string) error {
	f.calls = append(f.calls, name)
	if name == f.failOn {
		return errors.New("Sonarr is down")
	}
	return nil
}

func (f *fakeEpisodes) UnmonitorEpisode(int) error { return f.step("unmonitor") }
func (f *fakeEpisodes) EpisodeFileID(int) (int, error) {
	return f.fileID, f.step("lookup")
}
func (f *fakeEpisodes) DeleteEpisodeFile(int) error { return f.step("delete file") }

func TestRemoveFromSonarrUnmonitorsFirst(t *testing.T) {
	f := &fakeEpisodes{fileID: 40}
	h := &Handler{episodes: f}
	if err := h.removeFromSonarr(12); err != nil {
		t.Fatal(err)
	}
	want := []string{"unmonitor", "lookup", "delete file"}
	if len(f.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", f.calls, want)
	}
	for i := range want {
		if f.calls[i] != want[i] {
			t.Fatalf("calls = %v, want %v", f.calls, want)
		}
	}
}

func TestRemoveFromSonarrWithoutAFile(t *testing.T) {
	f := &fakeEpisodes{}
	h := &Handler{episodes: f}
	if err := h.removeFromSonarr(12); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 2 {
		t.Errorf("an episode without a file needs no file deletion, calls = %v", f.calls)
	}
}

func TestRemoveFromSonarrFailureStops(t *testing.T) {
	f := &fakeEpisodes{fileID: 40, failOn: "unmonitor"}
	h := &Handler{episodes: f}
	err := h.removeFromSonarr(12)
	var de *deleteError
	if !errors.As(err, &de) || de.status != http.StatusBadGateway {
		t.Fatalf("err = %v, want a 502 deleteError", err)
	}
	if len(f.calls) != 1 {
		t.Errorf("nothing should run after a failure, calls = %v", f.calls)
	}
}
