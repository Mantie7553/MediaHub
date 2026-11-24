package animemerge

import (
	"database/sql"

	"github.com/Mantie7553/MediaHub/backend/internal/clients/arr"
)

/*
Function:	LoadLinks
Purpose:	read the series links an admin made and lay them over Fribb's list
Params:
  - db: the database
*/
func LoadLinks(db *sql.DB) ([]arr.SeriesLink, error) {
	rows, err := db.Query(`SELECT anilist_id, tvdb_id, season_number FROM anime_series_links ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	links := []arr.SeriesLink{}
	for rows.Next() {
		var l arr.SeriesLink
		if err := rows.Scan(&l.AnilistID, &l.TvdbID, &l.Season); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	arr.SetSeriesLinks(links)
	return links, nil
}
