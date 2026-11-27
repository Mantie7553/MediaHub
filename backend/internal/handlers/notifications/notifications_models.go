package notifications

import "time"

type notificationResponse struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Title     string     `json:"title"`
	Body      *string    `json:"body"`
	Link      *string    `json:"link"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at"`
}

type markReadRequest struct {
	Read bool `json:"read"`
}
