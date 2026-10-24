package utils

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Mantie7553/MediaHub/backend/internal/platform/logger"
)

/*
Function:	JSON
Purpose:	create the JSON response to send to the frontend
Params:
  - w: http response writer to respond to the front end
  - data: whatever we want to send back to the frontend
  - status: variadic so that we do not have to supply it,
    but when provided sets the status code for the response
*/
func JSON(w http.ResponseWriter, data any, status ...int) {
	w.Header().Set("Content-Type", "application/json")
	if len(status) > 0 {
		w.WriteHeader(status[0])
	}
	json.NewEncoder(w).Encode(data)
}

/*
Function:	Error
Purpose:	Generic error handling for error messages
Params:
  - w: http response writer to respond to the front end
  - status: the error status
  - message: the message to display with the error
*/
func Error(w http.ResponseWriter, status int, message string) {
	logger.WarnDepth(3, "http %d: %s", status, message)
	http.Error(w, message, status)
}

/*
Function:	InternalError
Purpose:	Internal Error handling, used for anytime we do not have any other more specific error
Params:
  - w: http response writer to respond to the front end
  - err: the error that occured
*/
func InternalError(w http.ResponseWriter, err error) bool {
	if err != nil {
		logger.ErrorDepth(3, "%s", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return true
	}
	return false
}

/*
Function:	NullString
Purpose:	if a string is empty change it to a nil or return the existing string
Params:
  - s: the string we are checking
*/
func NullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

/*
Function:	NullInt
Purpose:	if the pointer is pointing to a null value return nil else return the value
Params:
  - i: an int pointer for the number we are checking
*/
func NullInt(i *int) *int {
	if i == nil {
		return nil
	}
	return i
}

func SafeMonth(m *int) int {
	if m == nil {
		return 1
	}
	return *m
}

func SafeDay(d *int) int {
	if d == nil {
		return 1
	}
	return *d
}

/*
Function:	SafeFileName
Purpose:	check that a name from the URL is a single plain file name that can't escape its directory
Params:
  - name: the file name we are checking
*/
func SafeFileName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	// reject both separators regardless of OS, plus NUL
	if strings.ContainsAny(name, "/\\\x00") {
		return false
	}
	return name == filepath.Base(name)
}

// ids are UUIDs; anything else can't be one of ours, and would make Postgres fail the query
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

/*
Function:	IsUUID
Purpose:	check that s is a UUID, before using it as an id in a query
*/
func IsUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

/*
Function:	IsWithin
Purpose:	check that a cleaned path stays inside dir (and is not dir itself)
Params:
  - dir: the directory the path must stay in
  - path: the path we are checking
*/
func IsWithin(dir, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

/*
Function:	LikePrefix
Purpose:	a LIKE pattern (used with ESCAPE '\') matching everything that starts with path, with
the wildcards in path escaped
Params:
  - path: the prefix, usually a folder ending in a separator
*/
func LikePrefix(path string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(path) + "%"
}
