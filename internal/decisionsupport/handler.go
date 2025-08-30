package decisionsupport

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for the decision support service
type Handler struct {
	service *Service
	logger  *zap.Logger
}

// NewHandler creates a new decision support handler
func NewHandler(service *Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers the handler routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/decision-support/analyze", h.handleAnalyze).Methods("POST")
	router.HandleFunc("/api/decision-support/recommendations", h.handleGetRecommendations).Methods("GET")
	router.HandleFunc("/api/decision-support/scenarios", h.handleAnalyzeScenarios).Methods("POST")
	router.HandleFunc("/api/decision-support/backtest", h.handleBacktest).Methods("POST")
	router.HandleFunc("/api/decision-support/insights/{symbol}", h.handleGetInsights).Methods("GET")
	router.HandleFunc("/api/decision-support/portfolio/optimize", h.handleOptimizePortfolio).Methods("GET")
	router.HandleFunc("/api/decision-support/alerts/configure", h.handleConfigureAlert).Methods("POST")
	router.HandleFunc("/api/decision-support/alerts", h.handleGetAlerts).Methods("GET")
	router.HandleFunc("/api/decision-support/alerts/{id}/acknowledge", h.handleAcknowledgeAlert).Methods("POST")
}

// handleAnalyze handles the analyze endpoint
func (h *Handler) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	var request AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Set default values if not provided
	if request.EndTime.IsZero() {
		request.EndTime = time.Now()
	}
	if request.StartTime.IsZero() {
		request.StartTime = request.EndTime.Add(-30 * 24 * time.Hour)
	}
	
	recommendations, err := h.service.Analyze(r.Context(), request)
	if err != nil {
		h.logger.Error("Failed to analyze", zap.Error(err))
		http.Error(w, "Failed to analyze: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recommendations)
}

// handleGetRecommendations handles the get recommendations endpoint
func (h *Handler) handleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	limitStr := r.URL.Query().Get("limit")
	
	limit := 10 // Default limit
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
			return
		}
	}
	
	recommendations, err := h.service.GetRecommendations(r.Context(), symbol, limit)
	if err != nil {
		h.logger.Error("Failed to get recommendations", zap.Error(err))
		http.Error(w, "Failed to get recommendations: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recommendations)
}

// handleAnalyzeScenarios handles the analyze scenarios endpoint
func (h *Handler) handleAnalyzeScenarios(w http.ResponseWriter, r *http.Request) {
	var request ScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	results, err := h.service.AnalyzeScenarios(r.Context(), request)
	if err != nil {
		h.logger.Error("Failed to analyze scenarios", zap.Error(err))
		http.Error(w, "Failed to analyze scenarios: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleBacktest handles the backtest endpoint
func (h *Handler) handleBacktest(w http.ResponseWriter, r *http.Request) {
	var request BacktestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	result, err := h.service.Backtest(r.Context(), request)
	if err != nil {
		h.logger.Error("Failed to run backtest", zap.Error(err))
		http.Error(w, "Failed to run backtest: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetInsights handles the get insights endpoint
func (h *Handler) handleGetInsights(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	symbol := vars["symbol"]
	
	insights, err := h.service.GetInsights(r.Context(), symbol)
	if err != nil {
		h.logger.Error("Failed to get insights", zap.Error(err))
		http.Error(w, "Failed to get insights: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(insights)
}

// handleOptimizePortfolio handles the optimize portfolio endpoint
func (h *Handler) handleOptimizePortfolio(w http.ResponseWriter, r *http.Request) {
	var portfolio Portfolio
	if err := json.NewDecoder(r.Body).Decode(&portfolio); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	objective := r.URL.Query().Get("objective")
	if objective == "" {
		objective = "sharpe" // Default objective
	}
	
	optimizedPortfolio, err := h.service.OptimizePortfolio(r.Context(), portfolio, objective)
	if err != nil {
		h.logger.Error("Failed to optimize portfolio", zap.Error(err))
		http.Error(w, "Failed to optimize portfolio: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(optimizedPortfolio)
}

// handleConfigureAlert handles the configure alert endpoint
func (h *Handler) handleConfigureAlert(w http.ResponseWriter, r *http.Request) {
	var config AlertConfiguration
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	id, err := h.service.ConfigureAlert(r.Context(), config)
	if err != nil {
		h.logger.Error("Failed to configure alert", zap.Error(err))
		http.Error(w, "Failed to configure alert: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]string{"id": id}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetAlerts handles the get alerts endpoint
func (h *Handler) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	acknowledgedStr := r.URL.Query().Get("acknowledged")
	acknowledged := false
	if acknowledgedStr != "" {
		var err error
		acknowledged, err = strconv.ParseBool(acknowledgedStr)
		if err != nil {
			http.Error(w, "Invalid acknowledged parameter", http.StatusBadRequest)
			return
		}
	}
	
	alerts, err := h.service.GetAlerts(r.Context(), acknowledged)
	if err != nil {
		h.logger.Error("Failed to get alerts", zap.Error(err))
		http.Error(w, "Failed to get alerts: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// handleAcknowledgeAlert handles the acknowledge alert endpoint
func (h *Handler) handleAcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["id"]
	
	err := h.service.AcknowledgeAlert(r.Context(), alertID)
	if err != nil {
		h.logger.Error("Failed to acknowledge alert", zap.Error(err))
		http.Error(w, "Failed to acknowledge alert: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
}

