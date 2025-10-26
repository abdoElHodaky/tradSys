package common

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// APIResponse represents a standardized API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	APIResponse
	Pagination PaginationInfo `json:"pagination,omitempty"`
}

// PaginationInfo contains pagination metadata
type PaginationInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// HandlerUtils provides common utilities for HTTP handlers
type HandlerUtils struct {
	logger *zap.Logger
}

// NewHandlerUtils creates a new handler utilities instance
func NewHandlerUtils(logger *zap.Logger) *HandlerUtils {
	return &HandlerUtils{
		logger: logger,
	}
}

// SuccessResponse sends a successful response
func (h *HandlerUtils) SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

// SuccessResponseWithMessage sends a successful response with a message
func (h *HandlerUtils) SuccessResponseWithMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// CreatedResponse sends a created response
func (h *HandlerUtils) CreatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
		Message: "Resource created successfully",
	})
}

// ErrorResponse sends an error response
func (h *HandlerUtils) ErrorResponse(c *gin.Context, statusCode int, err error) {
	// Use correlation-aware logging
	logger := LogWithCorrelationFromGin(h.logger, c)
	logger.Error("API error",
		zap.Error(err),
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.Int("status", statusCode))

	c.JSON(statusCode, APIResponse{
		Success: false,
		Error:   err.Error(),
	})
}

// BadRequestResponse sends a bad request response
func (h *HandlerUtils) BadRequestResponse(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Error:   message,
	})
}

// NotFoundResponse sends a not found response
func (h *HandlerUtils) NotFoundResponse(c *gin.Context, resource string) {
	c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Error:   resource + " not found",
	})
}

// InternalErrorResponse sends an internal server error response
func (h *HandlerUtils) InternalErrorResponse(c *gin.Context, err error) {
	h.ErrorResponse(c, http.StatusInternalServerError, err)
}

// PaginatedSuccessResponse sends a paginated successful response
func (h *HandlerUtils) PaginatedSuccessResponse(c *gin.Context, data interface{}, pagination PaginationInfo) {
	c.JSON(http.StatusOK, PaginatedResponse{
		APIResponse: APIResponse{
			Success: true,
			Data:    data,
		},
		Pagination: pagination,
	})
}

// GetIDParam extracts and validates an ID parameter from the URL
func (h *HandlerUtils) GetIDParam(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if id == "" {
		h.BadRequestResponse(c, "ID parameter is required")
		return "", false
	}
	return id, true
}

// GetIntParam extracts and validates an integer parameter from the URL
func (h *HandlerUtils) GetIntParam(c *gin.Context, paramName string) (int, bool) {
	paramStr := c.Param(paramName)
	if paramStr == "" {
		h.BadRequestResponse(c, paramName+" parameter is required")
		return 0, false
	}

	param, err := strconv.Atoi(paramStr)
	if err != nil {
		h.BadRequestResponse(c, paramName+" must be a valid integer")
		return 0, false
	}

	return param, true
}

// GetPaginationParams extracts pagination parameters from query string
func (h *HandlerUtils) GetPaginationParams(c *gin.Context) (page, pageSize int) {
	page = 1
	pageSize = 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if sizeStr := c.Query("page_size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}

	return page, pageSize
}

// BindJSON binds JSON request body and handles validation errors
func (h *HandlerUtils) BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		h.BadRequestResponse(c, "Invalid JSON: "+err.Error())
		return false
	}
	return true
}

// ValidateRequired checks if required fields are present
func (h *HandlerUtils) ValidateRequired(c *gin.Context, fields map[string]interface{}) bool {
	for fieldName, fieldValue := range fields {
		if fieldValue == nil || fieldValue == "" {
			h.BadRequestResponse(c, fieldName+" is required")
			return false
		}
	}
	return true
}

// CalculatePagination calculates pagination info
func (h *HandlerUtils) CalculatePagination(page, pageSize int, total int64) PaginationInfo {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	return PaginationInfo{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

// LogRequest logs incoming requests
func (h *HandlerUtils) LogRequest(c *gin.Context, message string) {
	// Use correlation-aware logging
	logger := LogWithCorrelationFromGin(h.logger, c)
	logger.Info(message,
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", c.ClientIP()),
		zap.String("user_agent", c.GetHeader("User-Agent")))
}
