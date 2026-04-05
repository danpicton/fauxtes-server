package auth

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	req.UserAgent = r.UserAgent()

	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidEmail):
			writeError(w, http.StatusBadRequest, "Invalid email.")
		case errors.Is(err, ErrDuplicateEmail):
			writeError(w, http.StatusConflict, "This email is already registered.")
		case errors.Is(err, ErrInvalidAPIVersion):
			writeError(w, http.StatusBadRequest, "Invalid API version.")
		case errors.Is(err, ErrLegacyAPIVersion):
			writeError(w, http.StatusBadRequest, "Legacy API version not supported for registration.")
		case errors.Is(err, ErrRegistrationDisabled):
			writeError(w, http.StatusForbidden, "Registration is disabled.")
		default:
			writeError(w, http.StatusInternalServerError, "An error occurred.")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetParams(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		writeError(w, http.StatusBadRequest, "Please provide an email address.")
		return
	}

	h.getParamsForEmail(w, r, email)
}

func (h *Handler) GetParamsPost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "Please provide an email address.")
		return
	}

	h.getParamsForEmail(w, r, req.Email)
}

func (h *Handler) getParamsForEmail(w http.ResponseWriter, r *http.Request, email string) {
	kp, err := h.service.GetParams(r.Context(), GetParamsRequest{
		Email:         email,
		Authenticated: false,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidEmail) {
			writeError(w, http.StatusBadRequest, "Invalid email.")
			return
		}
		writeError(w, http.StatusInternalServerError, "An error occurred.")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kp)
}

func (h *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	req.UserAgent = r.UserAgent()

	resp, err := h.service.SignIn(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidEmail):
			writeError(w, http.StatusBadRequest, "Username cannot be empty.")
		case errors.Is(err, ErrInvalidAPIVersion):
			writeError(w, http.StatusBadRequest, "Invalid API version.")
		case errors.Is(err, ErrCodeVerifierRequired):
			writeJSON(w, http.StatusGone, map[string]any{
				"error": map[string]any{
					"message": "Please update your client application.",
				},
			})
		case errors.Is(err, ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "Invalid email or password.")
		case errors.Is(err, ErrAccountLocked):
			writeError(w, http.StatusLocked, "Account is locked. Please try again later.")
		default:
			writeError(w, http.StatusInternalServerError, "An error occurred.")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) RefreshSession(w http.ResponseWriter, r *http.Request) {
	var req RefreshSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}

	resp, err := h.service.RefreshSession(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionNotFound):
			writeError(w, http.StatusUnauthorized, "Invalid session.")
		case errors.Is(err, ErrSessionExpired):
			writeError(w, http.StatusUnauthorized, "Session expired.")
		default:
			writeError(w, http.StatusInternalServerError, "An error occurred.")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) SignOut(w http.ResponseWriter, r *http.Request) {
	sessionUUID := SessionUUIDFromContext(r.Context())
	if sessionUUID == "" {
		writeError(w, http.StatusUnauthorized, "Not authenticated.")
		return
	}

	if err := h.service.SignOut(r.Context(), sessionUUID); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			writeError(w, http.StatusBadRequest, "Invalid session.")
			return
		}
		writeError(w, http.StatusInternalServerError, "An error occurred.")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"message": message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
