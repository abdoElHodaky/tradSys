# Risk Management API Documentation

## Overview

The Risk Management API provides components for managing risk in a trading system. It includes position limit management, exposure tracking, risk validation, and risk reporting.

## Core Components

### RiskManager

The `RiskManager` is the central component that coordinates all risk management activities. It provides methods for:

- Starting and stopping risk management components
- Setting position limits and risk limits
- Updating positions and market data
- Validating exposures
- Generating risk reports

```go
type RiskManager struct {
    // Logger
    logger *zap.Logger

    // Position limit manager
    positionLimitManager *PositionLimitManager

    // Exposure tracker
    exposureTracker *ExposureTracker

    // Risk validator
    riskValidator *RiskValidator

    // Risk reporter
    riskReporter *RiskReporter

    // ... other fields
}
```

Example usage:

```go
// Create a risk manager
manager := risk.NewRiskManager(
    logger,
    positionLimitManager,
    exposureTracker,
    riskValidator,
    riskReporter,
)

// Start the risk manager
err := manager.Start(ctx)

// Set a position limit
limit := &risk.PositionLimit{
    Symbol:    "BTC-USD",
    AccountID: "account1",
    MaxLong:   10.0,
    MaxShort:  5.0,
    MaxNet:    10.0,
    MaxGross:  15.0,
}
manager.SetPositionLimit(limit)

// Update a position
manager.UpdatePosition(
    "BTC-USD",
    "account1",
    1.0,  // deltaLong
    0.0,  // deltaShort
    50000.0,  // longPrice
    0.0,  // shortPrice
)

// Update market data
manager.UpdateMarketData("BTC-USD", 50000.0)

// Generate a risk report
report := manager.GenerateRiskReport("account1")

// Stop the risk manager
err := manager.Stop(ctx)
```

### PositionLimitManager

The `PositionLimitManager` manages position limits for trading symbols and accounts. It provides methods for:

- Setting position limits
- Setting default position limits
- Checking if a position would exceed limits
- Getting position limits

```go
type PositionLimitManager struct {
    // Logger
    logger *zap.Logger

    // Limits by symbol and account
    limits map[string]map[string]*PositionLimit

    // Default limits
    defaultLimits map[string]*PositionLimit

    // ... other fields
}
```

Example usage:

```go
// Create a position limit manager
manager := risk.NewPositionLimitManager(logger)

// Set a position limit
limit := &risk.PositionLimit{
    Symbol:    "BTC-USD",
    AccountID: "account1",
    MaxLong:   10.0,
    MaxShort:  5.0,
    MaxNet:    10.0,
    MaxGross:  15.0,
}
manager.SetLimit(limit)

// Set a default position limit
defaultLimit := &risk.PositionLimit{
    Symbol:    "BTC-USD",
    AccountID: "",
    MaxLong:   5.0,
    MaxShort:  2.0,
    MaxNet:    5.0,
    MaxGross:  7.0,
}
manager.SetDefaultLimit("BTC-USD", defaultLimit)

// Check if a position would exceed limits
err := manager.CheckLimit(
    "BTC-USD",
    "account1",
    5.0,  // currentLong
    0.0,  // currentShort
    6.0,  // deltaLong
    0.0,  // deltaShort
)
```

### ExposureTracker

The `ExposureTracker` tracks positions and exposures for trading symbols and accounts. It provides methods for:

- Updating positions
- Updating market data
- Setting beta values, sectors, and currencies
- Getting positions and exposures

```go
type ExposureTracker struct {
    // Logger
    logger *zap.Logger

    // Positions by symbol and account
    positions map[string]map[string]*Position

    // Exposures by account
    exposures map[string]*Exposure

    // ... other fields
}
```

Example usage:

```go
// Create an exposure tracker
tracker := risk.NewExposureTracker(logger)

// Update a position
tracker.UpdatePosition(
    "BTC-USD",
    "account1",
    1.0,  // deltaLong
    0.0,  // deltaShort
    50000.0,  // longPrice
    0.0,  // shortPrice
)

// Update market data
tracker.UpdateMarketData("BTC-USD", 50000.0)

// Set beta value
tracker.SetBeta("BTC-USD", 1.5)

// Set sector
tracker.SetSector("BTC-USD", "Cryptocurrency")

// Set currency
tracker.SetCurrency("BTC-USD", "USD")

// Get a position
position := tracker.GetPosition("BTC-USD", "account1")

// Get an exposure
exposure := tracker.GetExposure("account1")
```

### RiskValidator

The `RiskValidator` validates orders and exposures against risk limits. It provides methods for:

- Setting risk limits
- Validating orders
- Validating exposures

```go
type RiskValidator struct {
    // Logger
    logger *zap.Logger

    // Position limit manager
    positionLimitManager *PositionLimitManager

    // Exposure tracker
    exposureTracker *ExposureTracker

    // Circuit breaker factory
    circuitBreakerFactory *resilience.CircuitBreakerFactory

    // ... other fields
}
```

Example usage:

```go
// Create a risk validator
validator := risk.NewRiskValidator(
    logger,
    positionLimitManager,
    exposureTracker,
    circuitBreakerFactory,
)

// Set a risk limit
limit := &risk.RiskLimit{
    AccountID:           "account1",
    MaxNotionalExposure: 1000000.0,
    MaxBetaExposure:     500000.0,
    MaxSectorExposure: map[string]float64{
        "Cryptocurrency": 500000.0,
    },
    MaxCurrencyExposure: map[string]float64{
        "USD": 1000000.0,
    },
    MaxDrawdown:   0.1,
    MaxDailyLoss:  10000.0,
    MaxOrderSize: map[string]float64{
        "BTC-USD": 10.0,
    },
    MaxOrderValue: 100000.0,
}
validator.SetRiskLimit(limit)

// Validate an order
result, err := validator.ValidateOrder(ctx, order)
if err != nil {
    // Handle error
}
if !result.Passed {
    // Handle validation failure
    reason := result.Reason
    details := result.Details
}

// Validate an exposure
result, err := validator.ValidateExposure(ctx, "account1")
```

### RiskReporter

The `RiskReporter` generates risk reports for accounts. It provides methods for:

- Starting and stopping report generation
- Setting the report generation interval
- Generating reports on demand
- Getting reports
- Exporting reports to JSON and CSV

```go
type RiskReporter struct {
    // Logger
    logger *zap.Logger

    // Exposure tracker
    exposureTracker *ExposureTracker

    // ... other fields
}
```

Example usage:

```go
// Create a risk reporter
reporter := risk.NewRiskReporter(
    logger,
    exposureTracker,
)

// Start the risk reporter
err := reporter.Start(ctx)

// Set the report generation interval
reporter.SetReportInterval(15 * time.Minute)

// Generate a report on demand
report := reporter.GenerateReport("account1")

// Get reports
reports := reporter.GetReports("account1", 10)

// Get the latest report
latestReport := reporter.GetLatestReport("account1")

// Export a report to JSON
err := reporter.ExportReportToJSON(report, "report.json")

// Export a report to CSV
err := reporter.ExportReportToCSV(report, "report.csv")

// Stop the risk reporter
err := reporter.Stop(ctx)
```

## Risk Middleware

The risk middleware components provide integration with the order flow. They include:

- `OrderValidationMiddleware`: Validates orders against risk limits
- `ExposureValidationMiddleware`: Validates exposures after order execution
- `CircuitBreakerMiddleware`: Implements circuit breaker pattern for order handling

```go
type OrderValidationMiddleware struct {
    // Logger
    logger *zap.Logger

    // Risk validator
    riskValidator *risk.RiskValidator

    // Next handler
    next OrderHandler
}
```

Example usage:

```go
// Create an order validation middleware
middleware := middleware.NewOrderValidationMiddleware(
    logger,
    riskValidator,
    nextHandler,
)

// Handle an order
response, err := middleware.HandleOrder(ctx, order)
```

## Data Structures

### Position

The `Position` struct represents a position in a trading symbol:

```go
type Position struct {
    // Symbol is the trading symbol
    Symbol string

    // AccountID is the account ID
    AccountID string

    // Long is the long position
    Long float64

    // Short is the short position (as a positive number)
    Short float64

    // AvgLongPrice is the average price of the long position
    AvgLongPrice float64

    // AvgShortPrice is the average price of the short position
    AvgShortPrice float64

    // UnrealizedPnL is the unrealized profit and loss
    UnrealizedPnL float64

    // LastUpdateTime is the last update time
    LastUpdateTime time.Time
}
```

### Exposure

The `Exposure` struct represents an exposure for an account:

```go
type Exposure struct {
    // AccountID is the account ID
    AccountID string

    // Notional is the notional exposure
    Notional float64

    // Beta is the beta-adjusted exposure
    Beta float64

    // Sector is the sector exposure
    Sector map[string]float64

    // Currency is the currency exposure
    Currency map[string]float64

    // LastUpdateTime is the last update time
    LastUpdateTime time.Time
}
```

### PositionLimit

The `PositionLimit` struct represents a position limit:

```go
type PositionLimit struct {
    // Symbol is the trading symbol
    Symbol string

    // AccountID is the account ID
    AccountID string

    // MaxLong is the maximum long position
    MaxLong float64

    // MaxShort is the maximum short position (as a positive number)
    MaxShort float64

    // MaxNet is the maximum net position (long - short)
    MaxNet float64

    // MaxGross is the maximum gross position (long + short)
    MaxGross float64
}
```

### RiskLimit

The `RiskLimit` struct represents a risk limit:

```go
type RiskLimit struct {
    // AccountID is the account ID
    AccountID string

    // MaxNotionalExposure is the maximum notional exposure
    MaxNotionalExposure float64

    // MaxBetaExposure is the maximum beta-adjusted exposure
    MaxBetaExposure float64

    // MaxSectorExposure is the maximum sector exposure
    MaxSectorExposure map[string]float64

    // MaxCurrencyExposure is the maximum currency exposure
    MaxCurrencyExposure map[string]float64

    // MaxDrawdown is the maximum drawdown
    MaxDrawdown float64

    // MaxDailyLoss is the maximum daily loss
    MaxDailyLoss float64

    // MaxOrderSize is the maximum order size
    MaxOrderSize map[string]float64

    // MaxOrderValue is the maximum order value
    MaxOrderValue float64
}
```

### RiskReport

The `RiskReport` struct represents a risk report:

```go
type RiskReport struct {
    // Timestamp is the timestamp of the report
    Timestamp time.Time `json:"timestamp"`

    // AccountID is the account ID
    AccountID string `json:"account_id"`

    // Positions are the positions
    Positions map[string]*Position `json:"positions"`

    // Exposure is the exposure
    Exposure *Exposure `json:"exposure"`

    // RiskMetrics are the risk metrics
    RiskMetrics map[string]float64 `json:"risk_metrics"`
}
```

## fx Integration

The risk management components are integrated with the fx dependency injection framework through the following modules:

- `risk/fx/module.go`: Provides the risk management components
- `risk/middleware/fx/module.go`: Provides the risk middleware components

Example usage:

```go
// Create an fx application with risk management components
app := fx.New(
    risk.fx.Module,
    risk.middleware.fx.Module,
    // Other modules...
)

// Start the application
app.Start(ctx)
```

## Error Handling

The risk management components use the following error handling patterns:

- Returning errors from methods
- Logging errors with zap
- Using circuit breakers for resilience
- Providing detailed error messages

Common errors:

- `ErrPositionLimitExceeded`: Position limit exceeded
- `ErrLimitNotFound`: Limit not found
- `ErrRiskCheckFailed`: Risk check failed
- `ErrRiskLimitExceeded`: Risk limit exceeded
- `ErrInvalidOrder`: Invalid order

## Concurrency

The risk management components use the following concurrency patterns:

- Using mutexes for thread safety
- Using channels for communication
- Using context for cancellation

## Performance Considerations

- Use circuit breakers for resilience
- Use timeouts for long-running operations
- Use context for cancellation
- Use mutexes for thread safety

