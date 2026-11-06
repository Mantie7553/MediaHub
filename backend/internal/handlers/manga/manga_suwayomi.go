package manga

import (
	"database/sql"
	"errors"
	"net/http"
	"sort"
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
SuwayomiCandidate: a search result an admin can link a manga to
  - Exact: its title matches the manga's, ignoring case and punctuation
*/
type SuwayomiCandidate struct {
	suwayomi.Manga
	Exact bool `json:"exact"`
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
that fails is reported without hiding the others' results. Exact title matches come first
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

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Exact != results[j].Exact {
			return results[i].Exact
		}
		return results[i].SourceName < results[j].SourceName
	})
	utils.JSON(w, map[string]any{"query": query, "sources": len(sources), "results": results, "failed": failed})
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
