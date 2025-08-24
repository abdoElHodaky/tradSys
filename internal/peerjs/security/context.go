package security

import (
	"context"
)

// contextKey is a type for context keys
type contextKey string

// claimsKey is the key for claims in the context
const claimsKey contextKey = "claims"

// WithClaims adds claims to the context
func WithClaims(ctx context.Context, claims *PeerClaims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext gets claims from the context
func ClaimsFromContext(ctx context.Context) (*PeerClaims, bool) {
	claims, ok := ctx.Value(claimsKey).(*PeerClaims)
	return claims, ok
}

// PeerIDFromContext gets the peer ID from the context
func PeerIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	
	return claims.PeerID, true
}

// RoleFromContext gets the role from the context
func RoleFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return "", false
	}
	
	return claims.Role, true
}

