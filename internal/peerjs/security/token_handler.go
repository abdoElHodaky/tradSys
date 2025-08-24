package security

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// TokenRequest represents a request for a token
type TokenRequest struct {
	PeerID string `json:"peer_id"`
	Role   string `json:"role,omitempty"`
}

// TokenResponse represents a response with a token
type TokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TokenHandler handles token requests
type TokenHandler struct {
	authenticator *PeerAuthenticator
	logger        *zap.Logger
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(authenticator *PeerAuthenticator, logger *zap.Logger) *TokenHandler {
	return &TokenHandler{
		authenticator: authenticator,
		logger:        logger,
	}
}

// HandleTokenRequest handles a token request
func (h *TokenHandler) HandleTokenRequest(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Check rate limit
	if err := h.authenticator.CheckRateLimit(r); err != nil {
		h.logger.Warn("Rate limit exceeded",
			zap.Error(err),
			zap.String("remote_addr", r.RemoteAddr))
		
		RespondWithError(w, http.StatusTooManyRequests, err)
		return
	}
	
	// Parse the request
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse token request", zap.Error(err))
		RespondWithError(w, http.StatusBadRequest, err)
		return
	}
	
	// Validate the request
	if req.PeerID == "" {
		h.logger.Error("Missing peer ID in token request")
		RespondWithError(w, http.StatusBadRequest, errors.New("missing peer ID"))
		return
	}
	
	// Set default role if not provided
	if req.Role == "" {
		req.Role = "user"
	}
	
	// Generate the token
	token, err := h.authenticator.GenerateToken(req.PeerID, req.Role)
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		RespondWithError(w, http.StatusInternalServerError, err)
		return
	}
	
	// Create the response
	resp := TokenResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(h.authenticator.config.TokenExpiration),
	}
	
	// Set the content type
	w.Header().Set("Content-Type", "application/json")
	
	// Write the response
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("Failed to encode token response", zap.Error(err))
		RespondWithError(w, http.StatusInternalServerError, err)
		return
	}
}

// RegisterHandlers registers the token handler with an HTTP server
func (h *TokenHandler) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/api/token", h.HandleTokenRequest)
}

