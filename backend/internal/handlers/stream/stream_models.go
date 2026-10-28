package stream

import "github.com/Mantie7553/MediaHub/backend/internal/clients/aniskip"

// one stream from ffprobe -show_streams
type probeStream struct {
	Index       int    `json:"index"`
	CodecType   string `json:"codec_type"`
	CodecName   string `json:"codec_name"`
	Disposition struct {
		Default int `json:"default"`
		Forced  int `json:"forced"`
	} `json:"disposition"`
	Tags struct {
		Language string `json:"language"`
		Title    string `json:"title"`
		Filename string `json:"filename"` // attachments
		Mimetype string `json:"mimetype"`
	} `json:"tags"`
}

// one chapter from ffprobe -show_chapters
type probeChapter struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Tags      struct {
		Title string `json:"title"`
	} `json:"tags"`
}

type probeOutput struct {
	Streams  []probeStream  `json:"streams"`
	Chapters []probeChapter `json:"chapters"`
	Format   struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// what ffprobe says about a media file
type probeResult struct {
	Streams  []probeStream
	Chapters []probeChapter
	Duration float64 // seconds
}

type audioTrack struct {
	Index    int
	Language string
	Label    string
}

// subtitle track: a text one extracted to WebVTT for the player, or an image one (PGS,
// VobSub) that can only be burned into the video
type subtitleTrack struct {
	Index       int    `json:"index"`
	File        string `json:"file"`     // empty for image tracks
	AssFile     string `json:"ass_file"` // the ASS/SSA track as it is, for rendering with its styling
	Image       bool   `json:"image"`
	Language    string `json:"language"`
	Label       string `json:"label"`
	Title       string `json:"title"`
	Forced      bool   `json:"forced"`
	FileDefault bool   `json:"file_default"`
}

// what a transcode contains, saved as stream.json in the transcode folder
type streamManifest struct {
	Source        string          `json:"source"`
	Duration      float64         `json:"duration"`
	VideoCodec    string          `json:"video_codec"`
	AudioIndex    int             `json:"audio_index"`
	AudioLanguage string          `json:"audio_language"`
	BurnIndex     int             `json:"burn_index"` // image subtitle drawn into the video, -1 for none
	Subtitles     []subtitleTrack `json:"subtitles"`
	Fonts         []fontFile      `json:"fonts"` // attached fonts the ASS tracks use
}

// a font attached to the file, extracted for rendering ASS subtitles
type fontFile struct {
	Index int    `json:"index"`
	File  string `json:"file"`
}

type languagePreferences struct {
	Audio    string
	Subtitle string
}

type audioResponse struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Label    string `json:"label"`
	Selected bool   `json:"selected"`
}

type subtitleResponse struct {
	Index    int    `json:"index"`
	URL      string `json:"url,omitempty"` // empty for tracks that can only be burned in
	AssURL   string `json:"ass_url,omitempty"`
	Burn     bool   `json:"burn"`
	Language string `json:"language"`
	Label    string `json:"label"`
	Default  bool   `json:"default"`
}

type streamResponse struct {
	Playlist  string             `json:"playlist"`
	Audio     []audioResponse    `json:"audio"`
	Subtitles []subtitleResponse `json:"subtitles"`
	Burned    int                `json:"burned"` // the subtitle burned into this stream, -1 for none
	Fonts     []string           `json:"fonts"`  // attached fonts for the ASS tracks
}

type skipResponse struct {
	Segments []aniskip.Segment `json:"segments"`
	Source   string            `json:"source,omitempty"` // chapters or aniskip
}
