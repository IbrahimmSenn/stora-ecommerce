// response.go — JSON response helpers.
package response

import (
	"encoding/json"
	"log"
	"net/http"
)

type errorBody struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, errorBody{Error: message})
}

// ErrorWithCode is like Error but adds a stable machine-readable code so
// frontends can branch on it without string-matching the human message.
func ErrorWithCode(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorBody{Error: message, Code: code})
}
