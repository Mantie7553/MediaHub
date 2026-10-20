package ytdlp

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// ErrNotYouTubeVideo is returned for a source that isn't a single YouTube video link
var ErrNotYouTubeVideo = errors.New("source must be a YouTube or YouTube Music video link")

// YouTube video ids are 11 characters of letters, digits, - and _
var videoIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// the hosts a YouTube or YouTube Music link can use
var youtubeHosts = map[string]bool{
	"youtube.com":       true,
	"www.youtube.com":   true,
	"m.youtube.com":     true,
	"music.youtube.com": true,
	"youtu.be":          true,
}

/*
Function:	CanonicalURL
Purpose:	turn a YouTube or YouTube Music video link into https://www.youtube.com/watch?v=<id>,
or refuse it. Only the 11 character video id is taken from the input, so the result can't carry
anything else to yt-dlp: no option such as --exec (which would run a command), no other site,
no playlist, no local file
Params:
  - raw: the link a user sent, such as a search result's url
*/
func CanonicalURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || !youtubeHosts[strings.ToLower(u.Hostname())] {
		return "", ErrNotYouTubeVideo
	}

	var id string
	switch {
	case strings.EqualFold(u.Hostname(), "youtu.be"):
		id = strings.Trim(u.Path, "/")
	case u.Path == "/watch":
		id = u.Query().Get("v")
	case strings.HasPrefix(u.Path, "/shorts/"):
		id = strings.TrimPrefix(u.Path, "/shorts/")
	}
	if !videoIDRe.MatchString(id) {
		return "", ErrNotYouTubeVideo
	}
	return "https://www.youtube.com/watch?v=" + id, nil
}
