package security

import (
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the security components
var Module = fx.Options(
	// Provide the security components
	fx.Provide(NewPeerAuthenticator),
	fx.Provide(NewAuthMiddleware),
	fx.Provide(NewSecurePeerServer),
	fx.Provide(NewTokenHandler),
	
	// Register the token handler
	fx.Invoke(RegisterTokenHandler),
)

// SecurityParams contains parameters for security components
type SecurityParams struct {
	fx.In
	
	Logger *zap.Logger
	Config PeerAuthConfig `optional:"true"`
}

// NewPeerAuthenticator creates a new peer authenticator
func NewPeerAuthenticator(params SecurityParams) *PeerAuthenticator {
	// Use default config if not provided
	config := params.Config
	if config.JWTSecret == "" {
		config = DefaultPeerAuthConfig()
	}
	
	return NewPeerAuthenticator(config, params.Logger)
}

// RegisterTokenHandler registers the token handler with an HTTP server
func RegisterTokenHandler(
	handler *TokenHandler,
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) {
	// Create the HTTP server
	mux := http.NewServeMux()
	
	// Register the token handler
	handler.RegisterHandlers(mux)
	
	// Create the HTTP server
	server := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}
	
	// Register lifecycle hooks
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			logger.Info("Starting token server", zap.String("addr", server.Addr))
			
			// Start the server in a goroutine
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("Token server error", zap.Error(err))
				}
			}()
			
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Stopping token server")
			return server.Shutdown(ctx)
		},
	})
}

