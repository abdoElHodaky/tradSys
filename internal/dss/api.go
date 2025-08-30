package dss

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// API represents the Decision Support System API
type API struct {
	logger      *zap.Logger
	service     Service
	authService AuthService
}

// NewAPI creates a new DSS API
func NewAPI(logger *zap.Logger, service Service, authService AuthService) *API {
	return &API{
		logger:      logger,
		service:     service,
		authService: authService,
	}
}

// RegisterRoutes registers the DSS API routes
func (a *API) RegisterRoutes(router *gin.Engine) {
	dssGroup := router.Group("/api/v1/dss")
	
	// Apply authentication middleware
	dssGroup.Use(a.authMiddleware())
	
	// Analysis endpoints
	dssGroup.POST("/analyze", a.Analyze)
	dssGroup.GET("/analyze/:id", a.GetAnalysis)
	dssGroup.GET("/analyze/indicators", a.ListIndicators)
	dssGroup.POST("/analyze/custom", a.CustomAnalysis)
	
	// Recommendation endpoints
	dssGroup.POST("/recommend", a.Recommend)
	dssGroup.GET("/recommend/:id", a.GetRecommendation)
	dssGroup.GET("/recommend/history", a.GetRecommendationHistory)
	dssGroup.POST("/recommend/execute", a.ExecuteRecommendation)
	
	// Model endpoints
	dssGroup.GET("/models", a.ListModels)
	dssGroup.POST("/models", a.CreateModel)
	dssGroup.GET("/models/:id", a.GetModel)
	dssGroup.PUT("/models/:id", a.UpdateModel)
	dssGroup.DELETE("/models/:id", a.DeleteModel)
	dssGroup.POST("/models/:id/backtest", a.BacktestModel)
	
	// Backtest endpoints
	dssGroup.POST("/backtest", a.Backtest)
	dssGroup.GET("/backtest/:id", a.GetBacktest)
	dssGroup.GET("/backtest/:id/trades", a.GetBacktestTrades)
	dssGroup.GET("/backtest/:id/metrics", a.GetBacktestMetrics)
	
	// Alert endpoints
	dssGroup.POST("/alerts", a.CreateAlert)
	dssGroup.GET("/alerts", a.ListAlerts)
	dssGroup.GET("/alerts/:id", a.GetAlert)
	dssGroup.PUT("/alerts/:id", a.UpdateAlert)
	dssGroup.DELETE("/alerts/:id", a.DeleteAlert)
	dssGroup.GET("/alerts/history", a.GetAlertHistory)
	
	// Webhook endpoints
	dssGroup.POST("/webhooks", a.RegisterWebhook)
	dssGroup.GET("/webhooks", a.ListWebhooks)
	dssGroup.GET("/webhooks/:id", a.GetWebhook)
	dssGroup.PUT("/webhooks/:id", a.UpdateWebhook)
	dssGroup.DELETE("/webhooks/:id", a.DeleteWebhook)
	dssGroup.POST("/webhooks/:id/test", a.TestWebhook)
	
	// Market data endpoints
	dssGroup.GET("/market-data/:symbol", a.GetMarketData)
	dssGroup.GET("/market-data/:symbol/candles", a.GetCandles)
	dssGroup.GET("/market-data/:symbol/depth", a.GetOrderBookDepth)
	dssGroup.GET("/market-data/:symbol/trades", a.GetRecentTrades)
	
	// WebSocket endpoint
	dssGroup.GET("/stream", a.WebSocketHandler)
}

// authMiddleware authenticates API requests
func (a *API) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		apiKey := c.GetHeader("X-API-Key")
		
		// Check for token
		if token != "" {
			// Remove "Bearer " prefix if present
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}
			
			// Validate token
			user, err := a.authService.ValidateToken(c.Request.Context(), token)
			if err != nil {
				a.logger.Error("Invalid token", zap.Error(err))
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": gin.H{
						"code":    "authentication_required",
						"message": "Invalid or expired token",
					},
				})
				c.Abort()
				return
			}
			
			// Set user in context
			c.Set("user", user)
			c.Next()
			return
		}
		
		// Check for API key
		if apiKey != "" {
			// Validate API key
			user, err := a.authService.ValidateAPIKey(c.Request.Context(), apiKey)
			if err != nil {
				a.logger.Error("Invalid API key", zap.Error(err))
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": gin.H{
						"code":    "authentication_required",
						"message": "Invalid API key",
					},
				})
				c.Abort()
				return
			}
			
			// Set user in context
			c.Set("user", user)
			c.Next()
			return
		}
		
		// No authentication provided
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"code":    "authentication_required",
				"message": "Authentication required",
			},
		})
		c.Abort()
	}
}

// Analyze handles market data analysis requests
func (a *API) Analyze(c *gin.Context) {
	var request AnalysisRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		a.logger.Error("Invalid analysis request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "invalid_parameters",
				"message": "Invalid parameters provided",
				"details": err.Error(),
			},
		})
		return
	}
	
	// Validate request
	if request.Symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "invalid_parameters",
				"message": "Symbol is required",
			},
		})
		return
	}
	
	// Get user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "internal_error",
				"message": "User context not found",
			},
		})
		return
	}
	
	// Perform analysis
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	result, err := a.service.Analyze(ctx, user.(User), request)
	if err != nil {
		a.logger.Error("Analysis failed", zap.Error(err), zap.String("symbol", request.Symbol))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "analysis_failed",
				"message": "Failed to perform analysis",
				"details": err.Error(),
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, result)
}

// GetAnalysis retrieves a previous analysis
func (a *API) GetAnalysis(c *gin.Context) {
	analysisID := c.Param("id")
	
	// Get user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "internal_error",
				"message": "User context not found",
			},
		})
		return
	}
	
	// Get analysis
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	
	analysis, err := a.service.GetAnalysis(ctx, user.(User), analysisID)
	if err != nil {
		a.logger.Error("Failed to get analysis", zap.Error(err), zap.String("analysis_id", analysisID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "retrieval_failed",
				"message": "Failed to retrieve analysis",
				"details": err.Error(),
			},
		})
		return
	}
	
	if analysis == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "resource_not_found",
				"message": "Analysis not found",
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, analysis)
}

// ListIndicators lists available technical indicators
func (a *API) ListIndicators(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	
	indicators, err := a.service.ListIndicators(ctx)
	if err != nil {
		a.logger.Error("Failed to list indicators", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "retrieval_failed",
				"message": "Failed to retrieve indicators",
				"details": err.Error(),
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"indicators": indicators,
	})
}

// CustomAnalysis performs custom analysis with provided algorithm
func (a *API) CustomAnalysis(c *gin.Context) {
	var request CustomAnalysisRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		a.logger.Error("Invalid custom analysis request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "invalid_parameters",
				"message": "Invalid parameters provided",
				"details": err.Error(),
			},
		})
		return
	}
	
	// Get user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "internal_error",
				"message": "User context not found",
			},
		})
		return
	}
	
	// Perform custom analysis
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	
	result, err := a.service.CustomAnalysis(ctx, user.(User), request)
	if err != nil {
		a.logger.Error("Custom analysis failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "analysis_failed",
				"message": "Failed to perform custom analysis",
				"details": err.Error(),
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, result)
}

// Recommend generates trading recommendations
func (a *API) Recommend(c *gin.Context) {
	var request RecommendationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		a.logger.Error("Invalid recommendation request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "invalid_parameters",
				"message": "Invalid parameters provided",
				"details": err.Error(),
			},
		})
		return
	}
	
	// Get user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "internal_error",
				"message": "User context not found",
			},
		})
		return
	}
	
	// Generate recommendation
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	recommendation, err := a.service.Recommend(ctx, user.(User), request)
	if err != nil {
		a.logger.Error("Recommendation failed", zap.Error(err), zap.String("symbol", request.Symbol))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "recommendation_failed",
				"message": "Failed to generate recommendation",
				"details": err.Error(),
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, recommendation)
}

// GetRecommendation retrieves a specific recommendation
func (a *API) GetRecommendation(c *gin.Context) {
	recommendationID := c.Param("id")
	
	// Get user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "internal_error",
				"message": "User context not found",
			},
		})
		return
	}
	
	// Get recommendation
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	
	recommendation, err := a.service.GetRecommendation(ctx, user.(User), recommendationID)
	if err != nil {
		a.logger.Error("Failed to get recommendation", zap.Error(err), zap.String("recommendation_id", recommendationID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "retrieval_failed",
				"message": "Failed to retrieve recommendation",
				"details": err.Error(),
			},
		})
		return
	}
	
	if recommendation == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "resource_not_found",
				"message": "Recommendation not found",
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, recommendation)
}

// GetRecommendationHistory retrieves historical recommendations
func (a *API) GetRecommendationHistory(c *gin.Context) {
	// Parse query parameters
	symbol := c.Query("symbol")
	limit := 10 // Default limit
	
	// Get user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "internal_error",
				"message": "User context not found",
			},
		})
		return
	}
	
	// Get recommendation history
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	
	history, err := a.service.GetRecommendationHistory(ctx, user.(User), symbol, limit)
	if err != nil {
		a.logger.Error("Failed to get recommendation history", zap.Error(err), zap.String("symbol", symbol))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "retrieval_failed",
				"message": "Failed to retrieve recommendation history",
				"details": err.Error(),
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"recommendations": history,
	})
}

// ExecuteRecommendation executes a recommendation as a trade
func (a *API) ExecuteRecommendation(c *gin.Context) {
	var request ExecuteRecommendationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		a.logger.Error("Invalid execute recommendation request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "invalid_parameters",
				"message": "Invalid parameters provided",
				"details": err.Error(),
			},
		})
		return
	}
	
	// Get user from context
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "internal_error",
				"message": "User context not found",
			},
		})
		return
	}
	
	// Execute recommendation
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	result, err := a.service.ExecuteRecommendation(ctx, user.(User), request)
	if err != nil {
		a.logger.Error("Failed to execute recommendation", zap.Error(err), zap.String("recommendation_id", request.RecommendationID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "execution_failed",
				"message": "Failed to execute recommendation",
				"details": err.Error(),
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, result)
}

// WebSocketHandler handles WebSocket connections
func (a *API) WebSocketHandler(c *gin.Context) {
	// Implementation for WebSocket handling
	// This would typically upgrade the HTTP connection to WebSocket
	// and handle real-time data streaming
}

// Additional handler methods for other endpoints would be implemented here
// ListModels, CreateModel, GetModel, UpdateModel, DeleteModel, BacktestModel
// Backtest, GetBacktest, GetBacktestTrades, GetBacktestMetrics
// CreateAlert, ListAlerts, GetAlert, UpdateAlert, DeleteAlert, GetAlertHistory
// RegisterWebhook, ListWebhooks, GetWebhook, UpdateWebhook, DeleteWebhook, TestWebhook
// GetMarketData, GetCandles, GetOrderBookDepth, GetRecentTrades

