/*
Package archive hides download requests and jobs from the admin's Downloads lists without
deleting them. An archived row keeps everything it had and can be brought back
*/
package archive

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
	"github.com/lib/pq"
)

// most rows one call changes, far more than the Downloads page ever shows at once
const maxIDs = 5000

// the rows to change: the ones the admin was looking at, so a row that arrived since isn't
// swept up with them
type request struct {
	IDs      []string `json:"ids"`
	Archived bool     `json:"archived"`
}

/*
Function:	Filter
Purpose:	the WHERE condition for a list: archived rows with ?archived=true, the rest otherwise
Params:
  - r: the list request
  - alias: the table's alias in the query
*/
func Filter(r *http.Request, alias string) string {
	if r.URL.Query().Get("archived") == "true" {
		return alias + ".archived_at IS NOT NULL"
	}
	return alias + ".archived_at IS NULL"
}

/*
Function:	Handler
Purpose:	an endpoint that archives ({"ids": [...], "archived": true}) or restores
({"archived": false}) rows of a table, answering with how many changed
Params:
  - db: the database
  - table: download_requests or download_jobs, never user input
*/
func Handler(db *sql.DB, table string) http.HandlerFunc {
	query := fmt.Sprintf(
		`UPDATE %s SET archived_at = CASE WHEN $2 THEN NOW() ELSE NULL END
		WHERE id = ANY($1::uuid[]) AND (archived_at IS NULL) = $2`,
		table,
	)

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if !utils.DecodeJSON(w, r, &req) {
			return
		}
		if len(req.IDs) == 0 || len(req.IDs) > maxIDs {
			utils.Error(w, http.StatusBadRequest, fmt.Sprintf("send between 1 and %d ids", maxIDs))
			return
		}
		for _, id := range req.IDs {
			if !utils.IsUUID(id) {
				utils.Error(w, http.StatusBadRequest, "invalid id")
				return
			}
		}

		res, err := db.Exec(query, pq.Array(req.IDs), req.Archived)
		if utils.InternalError(w, err) {
			return
		}
		n, err := res.RowsAffected()
		if utils.InternalError(w, err) {
			return
		}
		utils.JSON(w, map[string]int64{"updated": n})
	}
}
