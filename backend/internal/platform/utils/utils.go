package utils

import (
	"encoding/json"
	"net/http"

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
		logger.ErrorDepth(3, err.Error())
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
func NullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

/*
Function:	NullInt
Purpose:	if the pointer is pointing to a null value return nil else return the value
Params:
  - i: an int pointer for the number we are checking
*/
func NullInt(i *int) interface{} {
	if i == nil {
		return nil
	}
	return *i
}
