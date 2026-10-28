package invites

import (
	"database/sql"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/auth"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

const (
	defaultExpiryDays = 7
	maxExpiryDays     = 30
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

/*
Function:	Create
Purpose:	Create a single use sign up code. The code is only ever returned here,
            the database keeps a hash
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createInviteRequest
	if !utils.DecodeJSON(w, r, &req) {
		return
	}

	days, ok := expiryDays(req.ExpiresInDays)
	if !ok {
		utils.Error(w, http.StatusBadRequest, "expires_in_days must be between 1 and 30")
		return
	}

	code, err := auth.GenerateInviteCode()
	if utils.InternalError(w, err) {
		return
	}

	var resp createInviteResponse
	err = h.db.QueryRow(
		`INSERT INTO invite_codes (code_hash, code_hint, note, created_by, expires_at)
		VALUES ($1, $2, $3, $4, NOW() + make_interval(days => $5))
		RETURNING id, expires_at`,
		auth.HashInviteCode(code), auth.InviteCodeHint(code), utils.NullString(req.Note),
		auth.GetUser(r).UserID, days,
	).Scan(&resp.ID, &resp.ExpiresAt)
	if utils.InternalError(w, err) {
		return
	}

	resp.Code = code
	utils.JSON(w, resp, http.StatusCreated)
}

/*
Function:	GetAll
Purpose:	List invites, newest first, with who made and used them
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(
		`SELECT ic.id, ic.code_hint, ic.note, creator.username, ic.created_at,
			ic.expires_at, invitee.username, ic.used_at,
			CASE
				WHEN ic.used_at IS NOT NULL THEN 'used'
				WHEN ic.expires_at <= NOW() THEN 'expired'
				ELSE 'active'
			END
		FROM invite_codes ic
		LEFT JOIN users creator ON creator.id = ic.created_by
		LEFT JOIN users invitee ON invitee.id = ic.used_by
		ORDER BY ic.created_at DESC`,
	)
	if utils.InternalError(w, err) {
		return
	}
	defer rows.Close()

	invites := []inviteResponse{}
	for rows.Next() {
		var i inviteResponse
		if err := rows.Scan(&i.ID, &i.CodeHint, &i.Note, &i.CreatedBy, &i.CreatedAt,
			&i.ExpiresAt, &i.UsedBy, &i.UsedAt, &i.Status); err != nil {
			utils.InternalError(w, err)
			return
		}
		invites = append(invites, i)
	}
	if utils.InternalError(w, rows.Err()) {
		return
	}

	utils.JSON(w, invites)
}

/*
Function:	Revoke
Purpose:	Delete an invite that hasn't been used. Used invites are kept as a record
Params:
  - w: http response writer to respond to the front end
  - r: http request coming from the frontend
*/
func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, ok := utils.UUIDParam(w, r, "id")
	if !ok {
		return
	}

	res, err := h.db.Exec(`DELETE FROM invite_codes WHERE id = $1 AND used_at IS NULL`, id)
	if utils.InternalError(w, err) {
		return
	}
	n, err := res.RowsAffected()
	if utils.InternalError(w, err) {
		return
	}
	if n == 0 {
		utils.Error(w, http.StatusNotFound, "invite not found or already used")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

/*
Function:	expiryDays
Purpose:	Default and bound the requested invite lifetime
Params:
  - requested: days asked for, 0 means use the default
*/
func expiryDays(requested int) (int, bool) {
	if requested == 0 {
		return defaultExpiryDays, true
	}
	if requested < 0 || requested > maxExpiryDays {
		return 0, false
	}
	return requested, true
}
