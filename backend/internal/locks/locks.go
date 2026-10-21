/*
Package locks keeps an admin's edits through library syncs. A field an admin sets by hand is
named in media_items.locked_fields, and every sync writes it through these helpers, which keep
the current value while the field is locked
*/
package locks

import "fmt"

// the fields an admin can lock, as stored in media_items.locked_fields
const (
	Title         = "title"
	Description   = "description"
	Cover         = "cover"
	ReleaseDate   = "release_date"
	Genres        = "genres"
	Studio        = "studio"
	Status        = "status"
	Director      = "director"
	RuntimeMins   = "runtime_mins"
	Author        = "author"
	TotalChapters = "total_chapters"
	TotalVolumes  = "total_volumes"
	Artist        = "artist"
	TrackNumber   = "track_number"
)

var known = map[string]bool{
	Title: true, Description: true, Cover: true, ReleaseDate: true, Genres: true,
	Studio: true, Status: true, Director: true, RuntimeMins: true, Author: true,
	TotalChapters: true, TotalVolumes: true, Artist: true, TrackNumber: true,
}

// Known reports whether name is a field that can be locked
func Known(name string) bool {
	return known[name]
}

/*
Function:	Keep
Purpose:	SQL for one assignment in an UPDATE of media_items: column = value, unless an admin
locked field, then it stays as it is
Params:
  - field: the locked_fields name
  - column: the media_items column
  - value: the SQL for the new value, such as $1 or COALESCE(description, $3)
*/
func Keep(field, column, value string) string {
	mustKnow(field)
	return fmt.Sprintf("%s = CASE WHEN '%s' = ANY(media_items.locked_fields) THEN media_items.%s ELSE %s END",
		column, field, column, value)
}

/*
Function:	KeepMeta
Purpose:	SQL for one assignment in an ON CONFLICT update of a type's metadata table
(anime_metadata, movie_metadata, ...): column = value unless the media item has field locked
Params:
  - field: the locked_fields name
  - table: the metadata table, keyed by media_item_id
  - column: its column
  - value: the SQL for the new value, usually EXCLUDED.<column>
*/
func KeepMeta(field, table, column, value string) string {
	mustKnow(field)
	// the lock is checked inside the subquery: ANY((SELECT ...)) would compare against each
	// row of the subquery rather than the array's elements
	return fmt.Sprintf("%s = CASE WHEN (SELECT '%s' = ANY(locked_fields) FROM media_items WHERE id = %s.media_item_id) THEN %s.%s ELSE %s END",
		column, field, table, table, column, value)
}

// the field names are written into SQL, so only the constants above are allowed
func mustKnow(field string) {
	if !known[field] {
		panic("locks: unknown field " + field)
	}
}
