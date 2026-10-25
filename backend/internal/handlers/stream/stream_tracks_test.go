package stream

import "testing"

// s builds a probeStream: codec type, codec, language, title, default, forced
func s(index int, codecType, codec, lang, title string, def, forced int) probeStream {
	p := probeStream{Index: index, CodecType: codecType, CodecName: codec}
	p.Tags.Language = lang
	p.Tags.Title = title
	p.Disposition.Default = def
	p.Disposition.Forced = forced
	return p
}

// track layouts from the library survey
var (
	chainsawMan = []probeStream{
		s(0, "video", "av1", "", "", 1, 0),
		s(1, "audio", "opus", "jpn", "", 1, 0),
		s(2, "subtitle", "ass", "ita", "Dialoghi", 1, 0),
		s(3, "subtitle", "ass", "eng", "Dialogues & Songs [GJM]", 0, 0),
		s(4, "subtitle", "ass", "eng", "Dialogues & Songs [CR w/ GJM TS]", 0, 0),
	}
	gamers = []probeStream{
		s(0, "video", "hevc", "jpn", "", 1, 0),
		s(1, "audio", "flac", "eng", "English FLAC 5.1", 1, 0),
		s(2, "audio", "flac", "jpn", "Japanese FLAC 2.0", 0, 0),
		s(3, "subtitle", "ass", "eng", "Signs & Songs (LostYears)", 1, 1),
		s(4, "subtitle", "ass", "eng", "Full Subtitles (LostYears)", 0, 0),
		s(5, "subtitle", "ass", "eng", "Signs & Songs (HorribleSubs)", 0, 1),
		s(6, "subtitle", "ass", "eng", "Full Subtitles (HorribleSubs)", 0, 0),
	}
	frieren = []probeStream{
		s(0, "video", "av1", "", "", 1, 0),
		s(1, "audio", "aac", "eng", "", 0, 0),
		s(2, "audio", "opus", "jpn", "", 1, 0),
		s(3, "subtitle", "ass", "eng", "Signs & Songs", 0, 1),
		s(4, "subtitle", "ass", "eng", "Dialogue", 1, 0),
	}
	blueBox = []probeStream{
		s(0, "video", "h264", "jpn", "JPBD", 1, 0),
		s(1, "audio", "flac", "jpn", "Japanese FLAC 2.0", 1, 0),
		s(2, "audio", "eac3", "eng", "English E-AC-3 2.0", 1, 0),
		s(3, "audio", "flac", "jpn", "Japanese Commentary by Voice Actors", 0, 0),
		s(4, "subtitle", "ass", "eng", "English [whomst]", 1, 0),
		s(5, "subtitle", "subrip", "eng", "English [NF]", 1, 0),
		s(6, "subtitle", "ass", "eng", "English Signs/Songs [whomst]", 0, 1),
		s(8, "subtitle", "subrip", "eng", "English [Dubtitle]", 0, 0),
		s(9, "subtitle", "subrip", "eng", "English (SDH)", 0, 0),
		s(10, "subtitle", "subrip", "ara", "Arabic", 1, 0),
		s(25, "subtitle", "subrip", "ita", "Italian", 0, 0),
	}
	durarara = []probeStream{
		s(0, "video", "hevc", "", "", 1, 0),
		s(1, "audio", "aac", "eng", "", 1, 0),
		s(2, "audio", "aac", "jpn", "", 0, 0),
		s(3, "subtitle", "ass", "eng", "Signs & Songs", 1, 0),
		s(4, "subtitle", "hdmv_pgs_subtitle", "eng", "", 0, 0),
	}
	noGameNoLife = []probeStream{
		s(0, "video", "av1", "", "", 1, 0),
		s(1, "audio", "opus", "jpn", "", 1, 0),
		s(2, "audio", "opus", "eng", "", 0, 0),
		s(3, "subtitle", "hdmv_pgs_subtitle", "eng", "Dialogue", 1, 0),
		s(4, "subtitle", "hdmv_pgs_subtitle", "eng", "Signs & Songs", 0, 1),
	}
	englishMovie = []probeStream{
		s(0, "video", "h264", "", "", 1, 0),
		s(1, "audio", "eac3", "eng", "", 1, 0),
		s(2, "subtitle", "subrip", "eng", "English", 0, 0),
		s(3, "subtitle", "subrip", "eng", "English (Forced)", 0, 1),
	}
)

// pick runs the same selection StreamMedia does for a set of preferences
func pick(streams []probeStream, audioPref, subPref string) (audio, subtitle int) {
	audio = -1
	if a := selectAudio(streams, audioPref); a != nil {
		audio = a.Index
	}
	m := buildManifest("in.mkv", probeResult{Streams: streams}, audio)
	return audio, defaultSubtitle(m.Subtitles, m.AudioLanguage, subPref)
}

func TestTrackSelection(t *testing.T) {
	tests := []struct {
		name      string
		streams   []probeStream
		audioPref string
		subPref   string
		audio     int
		subtitle  int
	}{
		// default preferences: Japanese audio, English subtitles
		{"chainsaw man: italian first, english dialogue chosen", chainsawMan, "jpn", "eng", 1, 3},
		{"gamers: japanese audio second, full subs not signs", gamers, "jpn", "eng", 2, 4},
		{"frieren: japanese default but second", frieren, "jpn", "eng", 2, 4},
		{"blue box: skips commentary, SDH, dubtitle", blueBox, "jpn", "eng", 1, 4},
		{"durarara: only text track is signs", durarara, "jpn", "eng", 2, 3},
		{"no game no life: only image subtitles", noGameNoLife, "jpn", "eng", 1, -1},
		{"english movie: no japanese, only forced signs on", englishMovie, "jpn", "eng", 1, 3},

		// English dub preference
		{"gamers dub: english audio, signs only", gamers, "eng", "eng", 1, 3},
		{"blue box dub: english audio, signs only", blueBox, "eng", "eng", 2, 6},
		{"chainsaw man dub: no english audio, falls back, dialogue on", chainsawMan, "eng", "eng", 1, 3},

		// other preferences
		{"subtitles off", gamers, "jpn", "off", 2, -1},
		{"file default audio", frieren, "default", "eng", 2, 4},
		{"file default audio is english", gamers, "default", "eng", 1, 3},
		{"italian subtitles", chainsawMan, "jpn", "ita", 1, 2},
		{"arabic subtitles", blueBox, "jpn", "ara", 1, 10},
		{"language with no track", chainsawMan, "jpn", "fre", 1, -1},
		{"no audio tracks", []probeStream{s(0, "video", "h264", "", "", 1, 0)}, "jpn", "eng", -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			audio, subtitle := pick(tt.streams, tt.audioPref, tt.subPref)
			if audio != tt.audio {
				t.Errorf("audio = %d, want %d", audio, tt.audio)
			}
			if subtitle != tt.subtitle {
				t.Errorf("subtitle = %d, want %d", subtitle, tt.subtitle)
			}
		})
	}
}

func TestSubtitleTracksSkipsImageFormats(t *testing.T) {
	tracks := subtitleTracks(durarara)
	if len(tracks) != 1 || tracks[0].Index != 3 {
		t.Fatalf("tracks = %+v, want only the ASS track", tracks)
	}
	if len(subtitleTracks(noGameNoLife)) != 0 {
		t.Fatal("PGS tracks should not be extracted")
	}
}

func TestAudioTracksListsEverything(t *testing.T) {
	tracks := audioTracks(blueBox)
	if len(tracks) != 3 {
		t.Fatalf("got %d audio tracks, want 3 (commentary can still be picked by hand)", len(tracks))
	}
	if tracks[2].Label != "Japanese Commentary by Voice Actors" {
		t.Errorf("label = %q", tracks[2].Label)
	}
}

func TestTrackLabel(t *testing.T) {
	tests := []struct {
		stream probeStream
		want   string
	}{
		{s(3, "subtitle", "ass", "eng", "Full Subtitles [MTBB]", 0, 0), "English - Full Subtitles [MTBB]"},
		{s(3, "subtitle", "ass", "eng", "English [whomst]", 0, 0), "English [whomst]"},
		{s(3, "subtitle", "ass", "ita", "", 0, 0), "Italian"},
		{s(3, "subtitle", "ass", "", "", 0, 0), "Track 3"},
		{s(3, "subtitle", "ass", "zxx", "Signs/Songs [GSK_kun]", 0, 0), "Signs/Songs [GSK_kun]"},
		{s(1, "audio", "opus", "jpn", "", 1, 0), "Japanese"},
	}
	for _, tt := range tests {
		if got := trackLabel(tt.stream); got != tt.want {
			t.Errorf("trackLabel(%q, %q) = %q, want %q", tt.stream.Tags.Language, tt.stream.Tags.Title, got, tt.want)
		}
	}
}
