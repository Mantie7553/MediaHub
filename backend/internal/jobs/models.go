package jobs

import "time"

type JobResponse struct {
	ID              string     `json:"id"`
	RequestID       string     `json:"request_id"`
	Status          string     `json:"status"`
	Handler         string     `json:"handler"`
	ProgressPct     int        `json:"progress_pct"`
	DestinationPath string     `json:"destination_path"`
	SourceURL       string     `json:"source_url"`
	MediaTitle      string     `json:"media_title"`
	ErrorMessage    *string    `json:"error_message"`
	StartedAt       *time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type JobRequest struct {
	RequestID string `json:"request_id"`
	SourceURL string `json:"source_url"`
}
