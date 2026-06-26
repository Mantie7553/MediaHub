package users

import "time"

type createUserRequest struct {
	Username           string `json:"username"`
	Email              string `json:"email"`
	Password           string `json:"password"`
	Role               string `json:"role"`
	DownloadPermission string `json:"download_permission"`
}

type updateUserRequest struct {
	Role               string `json:"role"`
	DownloadPermission string `json:"download_permission"`
	Password           string `json:"password"`
}

type userResponse struct {
	ID                 string    `json:"id"`
	Username           string    `json:"username"`
	Email              string    `json:"email"`
	Role               string    `json:"role"`
	DownloadPermission string    `json:"download_permission"`
	CreatedAt          time.Time `json:"created_at"`
}
