package plex

type linkRequest struct {
	RatingKey string `json:"rating_key"`
	LibraryID string `json:"library_id"`
}

type streamResponse struct {
	URL string `json:"url"`
}
