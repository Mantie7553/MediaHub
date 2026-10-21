package auth

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess = "access"
	TokenTypeMedia  = "media"

	// MediaTokenTTL is how long a signed media URL stays valid, counted from the start of
	// the hour it was minted in. Players remint on 401.
	MediaTokenTTL = 2 * time.Hour

	MediaTokenParam = "media_token"
)

var errInvalidMediaToken = errors.New("invalid media token")

/*
MediaScope: the exact resource a media token grants access to
  - MediaType: episode, movie, music, or light_novel
  - MediaID: id of the episode / movie / track / light novel
  - VolumeID: light novel volume id, empty otherwise
*/
type MediaScope struct {
	MediaType string `json:"media_type"`
	MediaID   string `json:"media_id"`
	VolumeID  string `json:"volume_id,omitempty"`
}

type MediaClaims struct {
	Type string `json:"typ"`
	MediaScope
	jwt.RegisteredClaims
}

func mediaSecret() []byte {
	return []byte(os.Getenv("MEDIA_SIGNING_SECRET"))
}

/*
Function:	GenerateMediaToken
Purpose:	Create a short lived token that only authorizes GETs for one media resource
Params:
  - userID: the id of the user the token is minted for
  - scope: the resource the token is valid for
*/
func GenerateMediaToken(userID string, scope MediaScope) (string, error) {
	secret := mediaSecret()
	if len(secret) == 0 {
		return "", errors.New("MEDIA_SIGNING_SECRET not set")
	}

	// issue on the hour so repeat requests get the same URL and the browser can cache
	// covers/images; tokens are valid for between TTL-1h and TTL
	issued := time.Now().Truncate(time.Hour)
	claims := MediaClaims{
		Type:       TokenTypeMedia,
		MediaScope: scope,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(issued.Add(MediaTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(issued),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

/*
Function:	ValidateMediaToken
Purpose:	Check that a media token is valid and scoped to exactly the requested resource
Params:
  - tokenStr: the token we are checking
  - want: the resource being requested
*/
func ValidateMediaToken(tokenStr string, want MediaScope) (*MediaClaims, error) {
	secret := mediaSecret()
	if len(secret) == 0 {
		return nil, errors.New("MEDIA_SIGNING_SECRET not set")
	}

	claims := &MediaClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.Type != TokenTypeMedia {
		return nil, errInvalidMediaToken
	}

	// exact match on every field, so episode/1 never authorizes episode/12
	if claims.MediaScope != want {
		return nil, errInvalidMediaToken
	}

	return claims, nil
}

/*
Function:	SignMediaPath
Purpose:	Mint a media token for a scope and append it to a path as a query param
Params:
  - path: the path to sign
  - userID: the id of the user the token is minted for
  - scope: the resource the token is valid for
*/
func SignMediaPath(path, userID string, scope MediaScope) (string, error) {
	token, err := GenerateMediaToken(userID, scope)
	if err != nil {
		return "", err
	}
	return path + "?" + MediaTokenParam + "=" + url.QueryEscape(token), nil
}

var (
	musicCoverPath    = regexp.MustCompile(`^/api/stream/music/([^/?]+)/cover$`)
	uploadedCoverPath = regexp.MustCompile(`^/api/media/([^/?]+)/cover$`)
	// a light novel's cover is an image inside its first volume's EPUB
	volumeImagePath = regexp.MustCompile(`^/api/light-novels/([^/?]+)/volumes/([^/?]+)/images/[^/?]+$`)
)

/*
Function:	SignCoverURL
Purpose:	Sign a stored music cover, admin uploaded cover or light novel volume image URL in place
            so <img> tags can load it. Any other cover URL (external hosts, the MangaDex proxy) is
            left unchanged
Params:
  - cover: the cover_image_url read from the database
  - userID: the id of the user the token is minted for
*/
func SignCoverURL(cover *string, userID string) error {
	if cover == nil {
		return nil
	}
	var scope MediaScope
	if m := musicCoverPath.FindStringSubmatch(*cover); m != nil {
		scope = MediaScope{MediaType: "music", MediaID: m[1]}
	} else if m := uploadedCoverPath.FindStringSubmatch(*cover); m != nil {
		scope = MediaScope{MediaType: "cover", MediaID: m[1]}
	} else if m := volumeImagePath.FindStringSubmatch(*cover); m != nil {
		scope = MediaScope{MediaType: "light_novel", MediaID: m[1], VolumeID: m[2]}
	} else {
		return nil
	}
	signed, err := SignMediaPath(*cover, userID, scope)
	if err != nil {
		return err
	}
	*cover = signed
	return nil
}

/*
Function:	RequireMediaToken
Purpose:	Only allow requests carrying a media token scoped to the requested resource
Params:
  - scopeFor: builds the scope being requested from the routed request
*/
func RequireMediaToken(scopeFor func(r *http.Request) MediaScope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := r.URL.Query().Get(MediaTokenParam)
			if tokenStr == "" {
				http.Error(w, "missing media token", http.StatusUnauthorized)
				return
			}

			if _, err := ValidateMediaToken(tokenStr, scopeFor(r)); err != nil {
				http.Error(w, "invalid or expired media token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
