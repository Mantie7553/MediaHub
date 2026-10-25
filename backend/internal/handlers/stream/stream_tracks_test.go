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
	m := buildManifest("in.mkv", probeResult{Streams: streams}, audio, -1)
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
		{"durarara: dialogue only as PGS, burned in over the signs track", durarara, "jpn", "eng", 2, 4},
		{"no game no life: only image subtitles", noGameNoLife, "jpn", "eng", 1, 3},
		{"durarara dub: text signs, no image dialogue", durarara, "eng", "eng", 1, 3},
		{"text dialogue beats image dialogue", append([]probeStream{s(5, "subtitle", "subrip", "eng", "", 0, 0)}, noGameNoLife...), "jpn", "eng", 1, 5},
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

func TestSubtitleTracksImageFormatsAreBurnOnly(t *testing.T) {
	tracks := subtitleTracks(durarara)
	if len(tracks) != 2 || tracks[0].Index != 3 || tracks[1].Index != 4 {
		t.Fatalf("tracks = %+v, want the ASS and PGS tracks", tracks)
	}
	if tracks[0].Image || tracks[0].File != "subs_3.vtt" || tracks[0].AssFile != "subs_3.ass" {
		t.Errorf("ASS track = %+v, want WebVTT and ASS files", tracks[0])
	}
	if !tracks[1].Image || tracks[1].File != "" || tracks[1].AssFile != "" {
		t.Errorf("PGS track = %+v, want burn only with no file", tracks[1])
	}
	if srt := subtitleTracks(englishMovie); srt[0].AssFile != "" {
		t.Errorf("SRT track kept as ASS: %+v", srt[0])
	}
	if !imageSubtitle(tracks, 4) || imageSubtitle(tracks, 3) || imageSubtitle(tracks, 1) {
		t.Error("imageSubtitle wrong")
	}

	other := subtitleTracks([]probeStream{s(2, "subtitle", "eia_608", "eng", "", 0, 0)})
	if len(other) != 0 {
		t.Errorf("unsupported codec listed: %+v", other)
	}
}

// attachment builds an attached file stream
func attachment(index int, codec, filename, mimetype string) probeStream {
	p := s(index, "attachment", codec, "", "", 0, 0)
	p.Tags.Filename = filename
	p.Tags.Mimetype = mimetype
	return p
}

func TestFontFiles(t *testing.T) {
	streams := append([]probeStream{}, chainsawMan...)
	streams = append(streams,
		attachment(5, "ttf", "Gandhi Sans.TTF", "application/x-truetype-font"),
		attachment(6, "otf", "font", "application/vnd.ms-opentype"),
		attachment(7, "", "cover.jpg", "image/jpeg"),
		attachment(8, "ttf", "../../evil.woff2", "font/woff2"),
	)
	fonts := fontFiles(streams, subtitleTracks(streams))
	want := []fontFile{{5, "font_5.ttf"}, {6, "font_6.ttf"}, {8, "font_8.woff2"}}
	if len(fonts) != len(want) {
		t.Fatalf("fonts = %+v, want %+v", fonts, want)
	}
	for i := range want {
		if fonts[i] != want[i] {
			t.Errorf("font %d = %+v, want %+v", i, fonts[i], want[i])
		}
	}

	// no ASS track to use them
	withSRT := append([]probeStream{}, englishMovie...)
	withSRT = append(withSRT, attachment(4, "ttf", "a.ttf", "font/ttf"))
	if got := fontFiles(withSRT, subtitleTracks(withSRT)); len(got) != 0 {
		t.Errorf("fonts extracted without an ASS track: %+v", got)
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
