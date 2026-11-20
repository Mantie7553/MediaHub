package tracking

import (
	"math"
	"sort"
)

// SeasonRating is one row of a user's ratings of a series' seasons. AnilistID 0 is the rating
// given to the season itself; others came from saving one AniList/MyAnimeList entry of it
type SeasonRating struct {
	Season    int
	AnilistID int
	Rating    int
}

// SeasonScore is a season's rating as shown: its own, else the average of its entries'
type SeasonScore struct {
	Season int `json:"season_number"`
	Rating int `json:"rating"`
	// rated by the user on the season itself, not worked out from its entries
	IsSet bool `json:"is_set"`
}

// the average of some ratings, rounded half up, nil for none
func average(ratings []int) *int {
	if len(ratings) == 0 {
		return nil
	}
	sum := 0
	for _, r := range ratings {
		sum += r
	}
	avg := int(math.Floor(float64(sum)/float64(len(ratings)) + 0.5))
	return &avg
}

/*
Function:	SeasonScores
Purpose:	each rated season's rating: the one given to the season, else the average of the
ratings its entries were saved with (Bakemonogatari and Nisemonogatari both sit in season 1)
Params:
  - rows: the user's season ratings of one series

Returns: the seasons in order
*/
func SeasonScores(rows []SeasonRating) []SeasonScore {
	own := map[int]int{}
	entries := map[int][]int{}
	for _, r := range rows {
		if r.AnilistID == 0 {
			own[r.Season] = r.Rating
		} else {
			entries[r.Season] = append(entries[r.Season], r.Rating)
		}
	}
	var out []SeasonScore
	for season, rating := range own {
		out = append(out, SeasonScore{Season: season, Rating: rating, IsSet: true})
	}
	for season, ratings := range entries {
		if _, ok := own[season]; !ok {
			out = append(out, SeasonScore{Season: season, Rating: *average(ratings)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Season < out[j].Season })
	return out
}

/*
Function:	SeriesRating
Purpose:	the rating a series gets from its seasons when the user hasn't given it one: the average
of its seasons' ratings. Specials (season 0) don't count
Returns: nil when no regular season is rated
*/
func SeriesRating(scores []SeasonScore) *int {
	var ratings []int
	for _, s := range scores {
		if s.Season != 0 {
			ratings = append(ratings, s.Rating)
		}
	}
	return average(ratings)
}

/*
Function:	LoadSeasonScores
Purpose:	a user's season ratings of a series, as shown
*/
func LoadSeasonScores(q Rows, userID, mediaItemID string) ([]SeasonScore, error) {
	rows, err := q.Query(
		`SELECT season_number, anilist_id, rating FROM user_season_ratings
		WHERE user_id = $1 AND media_item_id = $2`,
		userID, mediaItemID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []SeasonRating
	for rows.Next() {
		var r SeasonRating
		if err := rows.Scan(&r.Season, &r.AnilistID, &r.Rating); err != nil {
			return nil, err
		}
		all = append(all, r)
	}
	return SeasonScores(all), rows.Err()
}

/*
Function:	SetSeasonRating
Purpose:	save or clear a rating of a series' season, given to the season itself (anilistID 0)
or saved with one of its AniList entries, then work the series' rating out again unless the user
picked one
Params:
  - q: the database or a transaction
  - userID, mediaItemID: the user and the series
  - season: the TVDB season
  - anilistID: 0 for the season itself, else the entry the rating came with
  - rating: 1 to 10, nil clears it
*/
func SetSeasonRating(q Rows, userID, mediaItemID string, season, anilistID int, rating *int) error {
	var err error
	if rating == nil {
		_, err = q.Exec(
			`DELETE FROM user_season_ratings
			WHERE user_id = $1 AND media_item_id = $2 AND season_number = $3 AND anilist_id = $4`,
			userID, mediaItemID, season, anilistID,
		)
	} else {
		_, err = q.Exec(
			`INSERT INTO user_season_ratings (user_id, media_item_id, season_number, anilist_id, rating)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (user_id, media_item_id, season_number, anilist_id)
			DO UPDATE SET rating = EXCLUDED.rating, updated_at = NOW()`,
			userID, mediaItemID, season, anilistID, *rating,
		)
	}
	if err != nil {
		return err
	}
	return DeriveSeriesRating(q, userID, mediaItemID)
}

/*
Function:	DeriveSeriesRating
Purpose:	set a series' rating on the user's list to the average of their season ratings, unless
they picked one themselves. A title not on their list is left off it
*/
func DeriveSeriesRating(q Rows, userID, mediaItemID string) error {
	scores, err := LoadSeasonScores(q, userID, mediaItemID)
	if err != nil {
		return err
	}
	_, err = q.Exec(
		`UPDATE user_media_status SET rating = $3, rating_is_set = false
		WHERE user_id = $1 AND media_item_id = $2 AND (rating IS NULL OR NOT rating_is_set)
		AND rating IS DISTINCT FROM $3`,
		userID, mediaItemID, SeriesRating(scores),
	)
	return err
}
