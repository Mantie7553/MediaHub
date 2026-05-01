package requests

import "time"

type createRequest struct {
	MediaItemId		string	`json:"media_item_id"`
	TitleOverride	string	`json:"title_override"`
	SourceUrl		string	`json:"source_url"`
}

type rejectRequest struct {
    AdminNotes string `json:"admin_notes"`
}

type downloadRequestResponse struct {
    ID            string     `json:"id"`
    Status        string     `json:"status"`
    AutoApproved  bool       `json:"auto_approved"`
    RequestedAt   time.Time  `json:"requested_at"`
    TitleOverride *string    `json:"title_override"`
    MediaTitle    *string    `json:"media_title"`
	RequestedBy	  *string	 `json:"requested_by"`
}