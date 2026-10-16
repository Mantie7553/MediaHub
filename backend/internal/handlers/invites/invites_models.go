package invites

import "time"

type createInviteRequest struct {
	Note          string `json:"note"`
	ExpiresInDays int    `json:"expires_in_days"`
}

type createInviteResponse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

type inviteResponse struct {
	ID        string     `json:"id"`
	CodeHint  string     `json:"code_hint"`
	Note      *string    `json:"note"`
	CreatedBy *string    `json:"created_by"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedBy    *string    `json:"used_by"`
	UsedAt    *time.Time `json:"used_at"`
	Status    string     `json:"status"`
}
