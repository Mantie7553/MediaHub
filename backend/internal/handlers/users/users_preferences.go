package users

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

// ISO 639-2 codes as ffprobe reports them, e.g. jpn, eng
var languageCodeRe = regexp.MustCompile(`^[a-z]{3}$`)

type preferences struct {
	// a language code, or "default" for the track the file marks as default
	AudioLanguage string `json:"audio_language"`
	// a language code, or "off"
	SubtitleLanguage string `json:"subtitle_language"`
	// skip openings and endings without asking; left out of an update keeps it as it is
	AutoSkip *bool `json:"auto_skip,omitempty"`
}

/*
Function:	validPreferences
Purpose:	check both languages are a 3 letter code or the allowed special value
Params:
  - p: the preferences to check
*/
func validPreferences(p preferences) bool {
	// "off" also looks like a language code, but audio can't be turned off
	return p.AudioLanguage != "off" &&
		(p.AudioLanguage == "default" || languageCodeRe.MatchString(p.AudioLanguage)) &&
		(p.SubtitleLanguage == "off" || languageCodeRe.MatchString(p.SubtitleLanguage))
}

/*
Function:	GetPreferences
Purpose:	Get the current user's playback preferences: languages and auto skip
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	p := preferences{AutoSkip: new(bool)}
	err := h.db.QueryRow(
		`SELECT preferred_audio_language, preferred_subtitle_language, auto_skip FROM users WHERE id = $1`,
		auth.GetUser(r).UserID,
	).Scan(&p.AudioLanguage, &p.SubtitleLanguage, p.AutoSkip)
	if utils.InternalError(w, err) {
		return
	}
	utils.JSON(w, p)
}

/*
Function:	UpdatePreferences
Purpose:	Save the current user's playback preferences, auto skip only when it's given
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	var p preferences
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validPreferences(p) {
		utils.Error(w, http.StatusBadRequest, `audio_language must be a 3 letter code or "default", subtitle_language a 3 letter code or "off"`)
		return
	}

	// a nil AutoSkip is NULL, keeping the saved one
	saved := new(bool)
	err := h.db.QueryRow(
		`UPDATE users SET preferred_audio_language = $1, preferred_subtitle_language = $2,
		auto_skip = COALESCE($3, auto_skip), updated_at = NOW() WHERE id = $4
		RETURNING auto_skip`,
		p.AudioLanguage, p.SubtitleLanguage, p.AutoSkip, auth.GetUser(r).UserID,
	).Scan(saved)
	if utils.InternalError(w, err) {
		return
	}
	p.AutoSkip = saved
	utils.JSON(w, p)
}
