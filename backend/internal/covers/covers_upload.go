package covers

import (
	"io"
	"net/http"

	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

// largest cover image accepted, well above a high resolution poster scan
const maxBytes = 10 << 20

/*
Function:	ReadUpload
Purpose:	read the "file" of a multipart upload and check it is an image a browser shows.
Responds with the problem and returns false when it isn't
*/
func ReadUpload(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+(1<<20))
	if err := r.ParseMultipartForm(maxBytes); err != nil {
		utils.Error(w, http.StatusRequestEntityTooLarge, "the image is too large or the upload was cut off")
		return nil, false
	}
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("file")
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "no image in the upload")
		return nil, false
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if utils.InternalError(w, err) {
		return nil, false
	}
	if len(data) > maxBytes {
		utils.Error(w, http.StatusRequestEntityTooLarge, "the image is too large")
		return nil, false
	}
	if musicfiles.ImageType(data) == "" {
		utils.Error(w, http.StatusBadRequest, "covers must be JPEG, PNG, WebP or GIF images")
		return nil, false
	}
	return data, true
}
