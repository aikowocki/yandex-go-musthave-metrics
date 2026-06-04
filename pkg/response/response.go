package response

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

func WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		zap.S().Errorw("failed to write error response", "error", err)
	}
}
