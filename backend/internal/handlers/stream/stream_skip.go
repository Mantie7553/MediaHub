package stream

import (
	"errors"
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/aniskip"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/go-chi/chi/v5"
)

// an episode AniSkip had nothing for is asked again after this, people keep adding times
const skipRecheck = 7 * 24 * time.Hour

// chapter titles of the parts worth skipping, as release groups name them
var chapterTypes = []struct {
	re   *regexp.Regexp
	kind string
}{
	{regexp.MustCompile(`(?i)^\s*(op|opening|intro)\s*\d*\b`), "op"},
	{regexp.MustCompile(`(?i)^\s*(ed|ending|outro|credits)\s*\d*\b`), "ed"},
	{regexp.MustCompile(`(?i)^\s*(recap|previously)\b`), "recap"},
	{regexp.MustCompile(`(?i)^\s*(preview|next episode)\b`), "preview"},
}

/*
Function:	SkipTimes
Purpose:	the openings, endings, recaps and previews of an episode or movie, for skip buttons.
            The file's own chapters when it names them, otherwise AniSkip's times for episodes.
            Apart from playback so a slow AniSkip never holds up the video
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) SkipTimes(w http.ResponseWriter, r *http.Request) {
	mediaType := chi.URLParam(r, "type")
	mediaID, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	filePath, ok := h.videoFile(w, mediaType, mediaID)
	if !ok {
		return
	}

	probe, err := h.probes.get(filePath)
	if utils.InternalError(w, err) {
		return
	}
	if segments := chapterSegments(probe.Chapters); len(segments) > 0 {
		utils.JSON(w, skipResponse{Segments: segments, Source: "chapters"})
		return
	}

	resp := skipResponse{Segments: []aniskip.Segment{}}
	if mediaType == "episode" {
		// no skip buttons is fine, so AniSkip being down isn't an error for the player
		segments, err := h.aniskipSegments(mediaID, filePath, probe.Duration)
		if err != nil {
			logger.Warn("skip times for episode %s: %v", mediaID, err)
		} else if len(segments) > 0 {
			resp = skipResponse{Segments: segments, Source: "aniskip"}
		}
	}
	utils.JSON(w, resp)
}

/*
Function:	chapterSegments
Purpose:	the chapters whose titles say they're an opening, ending, recap or preview
Params:
  - chapters: the file's chapters from ffprobe
*/
func chapterSegments(chapters []probeChapter) []aniskip.Segment {
	segments := []aniskip.Segment{}
	for _, c := range chapters {
		start, err1 := strconv.ParseFloat(c.StartTime, 64)
		end, err2 := strconv.ParseFloat(c.EndTime, 64)
		if err1 != nil || err2 != nil || end <= start {
			continue
		}
		for _, t := range chapterTypes {
			if t.re.MatchString(c.Tags.Title) {
				segments = append(segments, aniskip.Segment{Type: t.kind, Start: start, End: end})
				break
			}
		}
	}
	return segments
}

/*
Function:	aniskipSegments
Purpose:	AniSkip's times for an episode, saved so it's only asked again for a new file or,
            when it had none, after a while. None when the season has no MyAnimeList id
Params:
  - episodeID: the episode
  - filePath: the episode's file, times are for one cut
  - length: the file's length in seconds
*/
func (h *Handler) aniskipSegments(episodeID, filePath string, length float64) ([]aniskip.Segment, error) {
	var malID, episode int
	err := h.db.QueryRow(
		`SELECT m.mal_id, e.episode_number FROM episodes e
		JOIN anime_season_mal m ON m.media_item_id = e.media_item_id AND m.season_number = e.season_number
		WHERE e.id = $1`,
		episodeID,
	).Scan(&malID, &episode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var cachedPath string
	var cachedJSON []byte
	var fetchedAt time.Time
	err = h.db.QueryRow(
		`SELECT file_path, segments, fetched_at FROM skip_times WHERE episode_id = $1`,
		episodeID,
	).Scan(&cachedPath, &cachedJSON, &fetchedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil && cachedPath == filePath {
		var cached []aniskip.Segment
		if json.Unmarshal(cachedJSON, &cached) == nil && (len(cached) > 0 || time.Since(fetchedAt) < skipRecheck) {
			return cached, nil
		}
	}

	segments, err := h.aniskip.SkipTimes(malID, episode, length)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(segments)
	if err != nil {
		return nil, err
	}
	if _, err := h.db.Exec(
		`INSERT INTO skip_times (episode_id, file_path, segments, fetched_at) VALUES ($1, $2, $3, NOW())
		ON CONFLICT (episode_id) DO UPDATE SET
		file_path = EXCLUDED.file_path, segments = EXCLUDED.segments, fetched_at = EXCLUDED.fetched_at`,
		episodeID, filePath, encoded,
	); err != nil {
		logger.Warn("could not save skip times for episode %s: %v", episodeID, err)
	}
	return segments, nil
}
