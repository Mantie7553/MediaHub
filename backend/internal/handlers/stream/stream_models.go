package stream

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
	} `json:"tags"`
}

type probeOutput struct {
	Streams []probeStream `json:"streams"`
	Format  struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

// what ffprobe says about a media file
type probeResult struct {
	Streams  []probeStream
	Duration float64 // seconds
}

type audioTrack struct {
	Index    int
	Language string
	Label    string
}

// subtitle track extracted to WebVTT for the player
type subtitleTrack struct {
	Index       int    `json:"index"`
	File        string `json:"file"`
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
	Subtitles     []subtitleTrack `json:"subtitles"`
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
	URL      string `json:"url"`
	Language string `json:"language"`
	Label    string `json:"label"`
	Default  bool   `json:"default"`
}

type streamResponse struct {
	Playlist  string             `json:"playlist"`
	Audio     []audioResponse    `json:"audio"`
	Subtitles []subtitleResponse `json:"subtitles"`
}
