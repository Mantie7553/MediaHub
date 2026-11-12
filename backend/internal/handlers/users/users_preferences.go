package users

import (
	"net/http"
	"regexp"
	"slices"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

// ISO 639-2 codes as ffprobe reports them, e.g. jpn, eng
var languageCodeRe = regexp.MustCompile(`^[a-z]{3}$`)

// the pages "/" can open, matching the users.home_page check
var homePages = []string{"overview", "anime", "movie", "manga", "music", "light_novel"}

type preferences struct {
	// a language code, or "default" for the track the file marks as default
	AudioLanguage string `json:"audio_language"`
	// a language code, or "off"
	SubtitleLanguage string `json:"subtitle_language"`
	// skip openings and endings without asking; left out of an update keeps it as it is
	AutoSkip *bool `json:"auto_skip,omitempty"`
	// the page "/" opens, one of homePages; left out of an update keeps it as it is
	HomePage *string `json:"home_page,omitempty"`
}

/*
Function:	validPreferences
Purpose:	check both languages are a 3 letter code or the allowed special value, and the home
page, when given, is one of homePages
Params:
  - p: the preferences to check
*/
func validPreferences(p preferences) bool {
	// "off" also looks like a language code, but audio can't be turned off
	return p.AudioLanguage != "off" &&
		(p.AudioLanguage == "default" || languageCodeRe.MatchString(p.AudioLanguage)) &&
		(p.SubtitleLanguage == "off" || languageCodeRe.MatchString(p.SubtitleLanguage)) &&
		(p.HomePage == nil || slices.Contains(homePages, *p.HomePage))
}

/*
Function:	GetPreferences
Purpose:	Get the current user's preferences: languages, auto skip and home page
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	p := preferences{AutoSkip: new(bool), HomePage: new(string)}
	err := h.db.QueryRow(
		`SELECT preferred_audio_language, preferred_subtitle_language, auto_skip, home_page FROM users WHERE id = $1`,
		auth.GetUser(r).UserID,
	).Scan(&p.AudioLanguage, &p.SubtitleLanguage, p.AutoSkip, p.HomePage)
	if utils.InternalError(w, err) {
		return
	}
	utils.JSON(w, p)
}

/*
Function:	UpdatePreferences
Purpose:	Save the current user's preferences, auto skip and the home page only when they're given
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	var p preferences
	if !utils.DecodeJSON(w, r, &p) {
		return
	}
	if !validPreferences(p) {
		utils.Error(w, http.StatusBadRequest, `audio_language must be a 3 letter code or "default", subtitle_language a 3 letter code or "off", `+
			`home_page one of overview, anime, movie, manga, music, light_novel`)
		return
	}

	// a nil AutoSkip or HomePage is NULL, keeping the saved one
	savedSkip, savedHome := new(bool), new(string)
	err := h.db.QueryRow(
		`UPDATE users SET preferred_audio_language = $1, preferred_subtitle_language = $2,
		auto_skip = COALESCE($3, auto_skip), home_page = COALESCE($4, home_page), updated_at = NOW()
		WHERE id = $5
		RETURNING auto_skip, home_page`,
		p.AudioLanguage, p.SubtitleLanguage, p.AutoSkip, p.HomePage, auth.GetUser(r).UserID,
	).Scan(savedSkip, savedHome)
	if utils.InternalError(w, err) {
		return
	}
	p.AutoSkip, p.HomePage = savedSkip, savedHome
	utils.JSON(w, p)
}
