package manga

import (
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/suwayomi"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

// languages searched, matching the chapters the downloader takes
const suwayomiLanguage = "en"

// longest search text accepted
const maxQueryLength = 200

// Suwayomi problems answer 424 Failed Dependency, not 502/503: the frontend reads those as
// MediaHub itself being down, and the real message would be lost

/*
SuwayomiLinkInfo: the Suwayomi title a manga gets its missing chapters from
*/
type SuwayomiLinkInfo struct {
	SourceID   string `json:"source_id"`
	SourceName string `json:"source_name"`
	MangaID    int    `json:"manga_id"`
	Title      string `json:"title"`
}

/*
SuwayomiCandidate: a search result an admin can link a manga to. Sources often list a title
several times (re-uploads, coloured editions), and usually only one is kept up to date, so
the top results carry their latest chapter
  - Exact: its title matches the manga's, ignoring case and punctuation
  - Checked: its chapters were looked up with the search
  - Chapters: its latest chapter and count, nil when not checked or it has no numbered chapters
  - MostRecent: the exact match with the newest chapter, the one to pick
*/
type SuwayomiCandidate struct {
	suwayomi.Manga
	Exact      bool          `json:"exact"`
	Checked    bool          `json:"checked"`
	Chapters   *ChapterStats `json:"chapters"`
	MostRecent bool          `json:"most_recent"`
}

/*
ChapterStats: how far a source's copy of a title goes
*/
type ChapterStats struct {
	Latest float64 `json:"latest"`
	Count  int     `json:"count"`
}

// results whose chapters are looked up with the search: the exact matches, or the first few
// results when none is exact. Each one is a request to the source site
const (
	maxStatsLookups = 8
	fallbackLookups = 5
	parallelLookups = 4
)

/*
Function:	chapterStats
Purpose:	The latest chapter and chapter count of a Suwayomi title, nil when it has no numbered
chapters
*/
func chapterStats(client *suwayomi.Client, mangaID int) (*ChapterStats, error) {
	chapters, err := client.Chapters(mangaID)
	if err != nil {
		return nil, err
	}
	if len(chapters) == 0 {
		return nil, nil
	}
	return &ChapterStats{Latest: chapters[len(chapters)-1].Number, Count: len(chapters)}, nil
}

/*
Function:	addChapterStats
Purpose:	Look up the chapters of the results most likely to be picked, a few at a time so a
source site isn't flooded. A lookup that fails just leaves that result without stats
*/
func addChapterStats(client *suwayomi.Client, results []SuwayomiCandidate) {
	var picks []int
	for i := range results {
		if results[i].Exact && len(picks) < maxStatsLookups {
			picks = append(picks, i)
		}
	}
	if len(picks) == 0 {
		for i := 0; i < len(results) && i < fallbackLookups; i++ {
			picks = append(picks, i)
		}
	}

	sem := make(chan struct{}, parallelLookups)
	var wg sync.WaitGroup
	for _, i := range picks {
		wg.Add(1)
		go func(c *SuwayomiCandidate) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			stats, err := chapterStats(client, c.ID)
			if err != nil {
				logger.Warn("suwayomi chapters of %q on %s: %s", c.Title, c.SourceName, err.Error())
				return
			}
			c.Checked, c.Chapters = true, stats
		}(&results[i])
	}
	wg.Wait()
}

/*
Function:	rankCandidates
Purpose:	Exact title matches first, newest latest chapter first among them, and mark the
newest exact match as the most recent. Results without stats come after those with
*/
func rankCandidates(results []SuwayomiCandidate) {
	latest := func(c SuwayomiCandidate) float64 {
		if c.Chapters == nil {
			return -1
		}
		return c.Chapters.Latest
	}
	sort.SliceStable(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.Exact != b.Exact {
			return a.Exact
		}
		if latest(a) != latest(b) {
			return latest(a) > latest(b)
		}
		return a.SourceName < b.SourceName
	})
	if len(results) > 0 && results[0].Exact && results[0].Chapters != nil {
		results[0].MostRecent = true
	}
}

/*
Function:	GetSuwayomiLink
Purpose:	Admin: whether Suwayomi is set up, and the title this manga is linked to (null if none)
*/
func (h *Handler) GetSuwayomiLink(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	link, err := h.suwayomiLink(id)
	if utils.InternalError(w, err) {
		return
	}
	utils.JSON(w, map[string]any{"configured": suwayomi.NewClient().Configured(), "link": link})
}

/*
Function:	SearchSuwayomi
Purpose:	Admin: search every installed English source for a title to link this manga to. The
"q" parameter defaults to the manga's title. Sources are searched at the same time, and one
that fails is reported without hiding the others' results. Exact title matches come first,
newest chapter first, with their chapter stats
*/
func (h *Handler) SearchSuwayomi(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	title, ok := h.mangaTitle(w, id)
	if !ok {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		query = title
	}
	if len(query) > maxQueryLength {
		utils.Error(w, http.StatusBadRequest, "search text is too long")
		return
	}

	client := suwayomi.NewClient()
	if !client.Configured() {
		utils.Error(w, http.StatusFailedDependency, "Suwayomi isn't set up on this server")
		return
	}
	sources, err := client.Sources(suwayomiLanguage)
	if err != nil {
		logger.Error("suwayomi sources: %s", err.Error())
		utils.Error(w, http.StatusFailedDependency, "could not reach Suwayomi")
		return
	}

	type failure struct {
		Source string `json:"source"`
		Error  string `json:"error"`
	}
	results := []SuwayomiCandidate{}
	failed := []failure{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, src := range sources {
		wg.Add(1)
		go func(src suwayomi.Source) {
			defer wg.Done()
			found, err := client.Search(src, query)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				logger.Warn("suwayomi search on %s: %s", src.Name, err.Error())
				failed = append(failed, failure{Source: src.Name, Error: "the source didn't answer"})
				return
			}
			for _, m := range found {
				results = append(results, SuwayomiCandidate{Manga: m, Exact: sameTitle(m.Title, title) || sameTitle(m.Title, query)})
			}
		}(src)
	}
	wg.Wait()

	// exact matches first, so they are the ones looked up
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Exact != results[j].Exact {
			return results[i].Exact
		}
		return results[i].SourceName < results[j].SourceName
	})
	addChapterStats(client, results)
	rankCandidates(results)
	utils.JSON(w, map[string]any{"query": query, "sources": len(sources), "results": results, "failed": failed})
}

/*
Function:	SuwayomiChapterStats
Purpose:	Admin: the latest chapter and chapter count of one Suwayomi title, from ?manga_id=. For
search results that weren't looked up with the search, and for the linked title
*/
func (h *Handler) SuwayomiChapterStats(w http.ResponseWriter, r *http.Request) {
	if _, ok := utils.UUIDParam(w, r, "id"); !ok {
		return
	}
	mangaID, err := strconv.Atoi(r.URL.Query().Get("manga_id"))
	if err != nil || mangaID <= 0 {
		utils.Error(w, http.StatusBadRequest, "manga_id is required")
		return
	}
	client := suwayomi.NewClient()
	if !client.Configured() {
		utils.Error(w, http.StatusFailedDependency, "Suwayomi isn't set up on this server")
		return
	}
	stats, err := chapterStats(client, mangaID)
	if err != nil {
		logger.Warn("suwayomi chapters of %d: %s", mangaID, err.Error())
		utils.Error(w, http.StatusFailedDependency, "the source didn't answer")
		return
	}
	utils.JSON(w, map[string]any{"chapters": stats})
}

/*
Function:	LinkSuwayomi
Purpose:	Admin: link this manga to a Suwayomi title, from {"manga_id"}. Its title and source are
read from Suwayomi, and the source must be an installed English one. Linking again replaces
the old link
*/
func (h *Handler) LinkSuwayomi(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	if _, ok := h.mangaTitle(w, id); !ok {
		return
	}
	var body struct {
		MangaID int `json:"manga_id"`
	}
	if !utils.DecodeJSON(w, r, &body) {
		return
	}
	if body.MangaID <= 0 {
		utils.Error(w, http.StatusBadRequest, "manga_id is required")
		return
	}

	client := suwayomi.NewClient()
	if !client.Configured() {
		utils.Error(w, http.StatusFailedDependency, "Suwayomi isn't set up on this server")
		return
	}
	m, err := client.MangaByID(body.MangaID)
	if err != nil {
		logger.Error("suwayomi manga %d: %s", body.MangaID, err.Error())
		utils.Error(w, http.StatusFailedDependency, "could not look that title up in Suwayomi")
		return
	}
	sources, err := client.Sources(suwayomiLanguage)
	if err != nil {
		utils.Error(w, http.StatusFailedDependency, "could not reach Suwayomi")
		return
	}
	var source *suwayomi.Source
	for i := range sources {
		if sources[i].ID == m.SourceID {
			source = &sources[i]
		}
	}
	if source == nil {
		utils.Error(w, http.StatusBadRequest, "that title's source isn't an installed English source")
		return
	}

	_, err = h.db.Exec(
		`INSERT INTO suwayomi_items (media_item_id, source_id, source_name, suwayomi_manga_id, title)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (media_item_id) DO UPDATE SET source_id = EXCLUDED.source_id,
			source_name = EXCLUDED.source_name, suwayomi_manga_id = EXCLUDED.suwayomi_manga_id,
			title = EXCLUDED.title, linked_at = NOW()`,
		id, source.ID, source.Name, m.ID, m.Title,
	)
	if utils.InternalError(w, err) {
		return
	}
	utils.JSON(w, SuwayomiLinkInfo{SourceID: source.ID, SourceName: source.Name, MangaID: m.ID, Title: m.Title})
}

/*
Function:	UnlinkSuwayomi
Purpose:	Admin: stop getting this manga's chapters from Suwayomi. Chapters already downloaded stay
*/
func (h *Handler) UnlinkSuwayomi(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	if _, err := h.db.Exec(`DELETE FROM suwayomi_items WHERE media_item_id = $1`, id); utils.InternalError(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// suwayomiLink is the manga's link, nil when it has none
func (h *Handler) suwayomiLink(id string) (*SuwayomiLinkInfo, error) {
	var link SuwayomiLinkInfo
	err := h.db.QueryRow(
		`SELECT source_id, source_name, suwayomi_manga_id, title FROM suwayomi_items WHERE media_item_id = $1`, id,
	).Scan(&link.SourceID, &link.SourceName, &link.MangaID, &link.Title)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// mangaTitle is the title of the manga with this id, answering 404/400 when it isn't one
func (h *Handler) mangaTitle(w http.ResponseWriter, id string) (string, bool) {
	var title, mediaType string
	err := h.db.QueryRow(`SELECT title, type FROM media_items WHERE id = $1`, id).Scan(&title, &mediaType)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Error(w, http.StatusNotFound, "manga not found")
		return "", false
	}
	if utils.InternalError(w, err) {
		return "", false
	}
	if mediaType != "manga" {
		utils.Error(w, http.StatusBadRequest, "only manga can be linked to Suwayomi")
		return "", false
	}
	return title, true
}

// sameTitle compares titles by their letters and digits only, ignoring case
func sameTitle(a, b string) bool {
	norm := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, s)
	}
	na := norm(a)
	return na != "" && na == norm(b)
}
