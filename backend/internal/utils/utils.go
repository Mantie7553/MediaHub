package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func JSON(w http.ResponseWriter, data any, status ...int) {
	w.Header().Set("Content-Type", "application/json")
	if len(status) > 0 {
		w.WriteHeader(status[0])
	}
	json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, status int, message string) {
	fmt.Println("other error:", message)
	http.Error(w, message, status)
}

func InternalError(w http.ResponseWriter, err error) bool {
	if err != nil {
		fmt.Println("internal error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return true
	}
	return false
}

func NullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func NullInt(i *int) interface{} {
	if i == nil {
		return nil
	}
	return *i
}
