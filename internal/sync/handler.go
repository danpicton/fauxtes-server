package sync

import (
	"encoding/json"
	"net/http"

	"github.com/danpicton/fauxtes-server/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	userUUID := auth.UserUUIDFromContext(r.Context())
	if userUUID == "" {
		writeError(w, http.StatusUnauthorized, "Not authenticated.")
		return
	}

	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}

	req.UserUUID = userUUID
	req.SessionUUID = auth.SessionUUIDFromContext(r.Context())
	req.ReadOnly = auth.ReadOnlyFromContext(r.Context())

	resp, err := h.service.Sync(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Sync failed.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"message": message,
		},
	})
}
