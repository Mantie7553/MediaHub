package notifications

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/Mantie7553/MediaHub/backend/internal/notify"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

/*
Function:	GetPrefs
Purpose:	which kinds of notification the user gets: every kind they can be sent, true unless they
turned it off
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetPrefs(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	prefs := map[string]bool{}
	for _, kind := range notify.KindsFor(user.Role == "admin") {
		prefs[kind] = true
	}

	rows, err := h.db.Query(`SELECT kind FROM notification_mutes WHERE user_id = $1`, user.UserID)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		if err := rows.Scan(&kind); utils.InternalError(w, err) {
			return
		}
		if _, ok := prefs[kind]; ok {
			prefs[kind] = false
		}
	}
	if utils.InternalError(w, rows.Err()) {
		return
	}
	utils.JSON(w, prefs)
}

/*
Function:	SetPrefs
Purpose:	turn kinds of notification on or off for the user. Kinds left out stay as they are
Params:
  - w: http response writer to respond to the front end
  - r: JSON {kind: on}
*/
func (h *Handler) SetPrefs(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r)
	var req map[string]bool
	if !utils.DecodeJSON(w, r, &req) {
		return
	}
	if err := checkKinds(req, notify.KindsFor(user.Role == "admin")); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := h.db.Begin()
	if utils.InternalError(w, err) {
		return
	}
	defer tx.Rollback()
	for kind, on := range req {
		query := `INSERT INTO notification_mutes (user_id, kind) VALUES ($1, $2) ON CONFLICT DO NOTHING`
		if on {
			query = `DELETE FROM notification_mutes WHERE user_id = $1 AND kind = $2`
		}
		if _, err := tx.Exec(query, user.UserID, kind); utils.InternalError(w, err) {
			return
		}
	}
	if utils.InternalError(w, tx.Commit()) {
		return
	}
	h.GetPrefs(w, r)
}

/*
Function:	checkKinds
Purpose:	an error naming the first kind asked for that the user can't be sent
*/
func checkKinds(req map[string]bool, allowed []string) error {
	for kind := range req {
		if !slices.Contains(allowed, kind) {
			return fmt.Errorf("unknown notification kind %q", kind)
		}
	}
	return nil
}
