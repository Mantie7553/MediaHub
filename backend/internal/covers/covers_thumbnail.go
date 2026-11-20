package covers

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"

	// decoders for the cover formats image.Decode reads
	_ "image/png"

	_ "golang.org/x/image/webp"

	"golang.org/x/image/draw"

	"github.com/Mantie7553/MediaHub/backend/internal/musicfiles"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/cache"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
	"github.com/Mantie7553/MediaHub/backend/internal/platform/utils"
)

// MaxCoverSide is the longest side a cover is sent at: sharp on the largest cover the app
// shows on a high density screen, a fraction of the size of full resolution album art
const MaxCoverSide = 800

const thumbnailQuality = 85

/*
Function:	Thumbnail
Purpose:	a cover scaled down to fit MaxCoverSide, as a JPEG. The original comes back unchanged
when it is already small enough, is a GIF (which may be animated), or can't be decoded
Params:
  - data: the cover image
  - contentType: its type, from musicfiles.ImageType

Returns:	the image and its content type
*/
func Thumbnail(data []byte, contentType string) ([]byte, string) {
	if contentType == "image/gif" {
		return data, contentType
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (cfg.Width <= MaxCoverSide && cfg.Height <= MaxCoverSide) {
		return data, contentType
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, contentType
	}

	w, h := cfg.Width, cfg.Height
	if w >= h {
		h = max(1, h*MaxCoverSide/w)
		w = MaxCoverSide
	} else {
		w = max(1, w*MaxCoverSide/h)
		h = MaxCoverSide
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)

	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: thumbnailQuality}); err != nil {
		return data, contentType
	}
	return out.Bytes(), "image/jpeg"
}

/*
Function:	Serve
Purpose:	send a cover kept on the NAS, scaled down by Thumbnail. The scaled copy is kept in the
file cache, keyed by the file's size and modified time so a replaced cover is scaled again.
Browsers may reuse it for an hour before checking it's unchanged
Params:
  - w: http response writer
  - r: the request, for conditional GETs
  - path: the cover's file on the NAS
*/
func Serve(w http.ResponseWriter, r *http.Request, path string) {
	f, err := os.Open(cache.Disk().Local(path))
	if os.IsNotExist(err) {
		utils.Error(w, http.StatusNotFound, "cover not found")
		return
	}
	if utils.InternalError(w, err) {
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if utils.InternalError(w, err) {
		return
	}

	key := fmt.Sprintf("cover-thumb|%d|%s|%d|%d", MaxCoverSide, path, info.Size(), info.ModTime().UnixNano())
	data, err := cache.Disk().Remote(key, func() ([]byte, error) {
		original, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}
		contentType := musicfiles.ImageType(original)
		if contentType == "" {
			return nil, errNotImage
		}
		data, _ := Thumbnail(original, contentType)
		return data, nil
	})
	if errors.Is(err, errNotImage) {
		utils.Error(w, http.StatusNotFound, "cover not found")
		return
	}
	if err != nil {
		logger.Error("reading the cover %s: %s", path, err.Error())
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.Header().Set("Content-Type", musicfiles.ImageType(data))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeContent(w, r, "", info.ModTime(), bytes.NewReader(data))
}

var errNotImage = errors.New("not an image a browser shows")
