package services

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ETFService handles ETF-specific operations and calculations
type ETFService struct {
	db           *gorm.DB
	assetService *AssetService
	logger       *zap.Logger
}

// ETFMetrics represents comprehensive ETF performance and operational metrics
type ETFMetrics struct {
	Symbol             string               `json:"symbol"`
	NAV                float64              `json:"nav"`
	MarketPrice        float64              `json:"market_price"`
	Premium            float64              `json:"premium"`
	TrackingError      float64              `json:"tracking_error"`
	ExpenseRatio       float64              `json:"expense_ratio"`
	AUM                float64              `json:"aum"`
	DividendYield      float64              `json:"dividend_yield"`
	BenchmarkIndex     string               `json:"benchmark_index"`
	CreationUnitSize   int                  `json:"creation_unit_size"`
	LastCreationDate   time.Time            `json:"last_creation_date"`
	LastRedemptionDate time.Time            `json:"last_redemption_date"`
	Liquidity          LiquidityMetrics     `json:"liquidity"`
	Holdings           []ETFHolding         `json:"holdings"`
	PerformanceMetrics ETFPerformance       `json:"performance_metrics"`
	RiskMetrics        ETFRiskMetrics       `json:"risk_metrics"`
	TaxEfficiency      TaxEfficiencyMetrics `json:"tax_efficiency"`
}

// LiquidityMetrics represents ETF liquidity characteristics
type LiquidityMetrics struct {
	BidAskSpread        float64 `json:"bid_ask_spread"`
	AverageVolume       int64   `json:"average_volume"`
	MedianVolume        int64   `json:"median_volume"`
	VolumeWeightedPrice float64 `json:"volume_weighted_price"`
	LiquidityScore      float64 `json:"liquidity_score"`
	MarketImpact        float64 `json:"market_impact"`
}

// ETFHolding represents individual holdings within an ETF
type ETFHolding struct {
	Symbol      string  `json:"symbol"`
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	Shares      int64   `json:"shares"`
	MarketValue float64 `json:"market_value"`
	Sector      string  `json:"sector"`
	Country     string  `json:"country"`
}

// ETFPerformance represents ETF performance metrics
type ETFPerformance struct {
	OneDay         float64 `json:"one_day"`
	OneWeek        float64 `json:"one_week"`
	OneMonth       float64 `json:"one_month"`
	ThreeMonth     float64 `json:"three_month"`
	SixMonth       float64 `json:"six_month"`
	YTD            float64 `json:"ytd"`
	OneYear        float64 `json:"one_year"`
	ThreeYear      float64 `json:"three_year"`
	FiveYear       float64 `json:"five_year"`
	TenYear        float64 `json:"ten_year"`
	SinceInception float64 `json:"since_inception"`
}

// ETFRiskMetrics represents ETF risk characteristics
type ETFRiskMetrics struct {
	Beta               float64 `json:"beta"`
	Alpha              float64 `json:"alpha"`
	Volatility         float64 `json:"volatility"`
	SharpeRatio        float64 `json:"sharpe_ratio"`
	MaxDrawdown        float64 `json:"max_drawdown"`
	VaR95              float64 `json:"var_95"`
	VaR99              float64 `json:"var_99"`
	CorrelationToIndex float64 `json:"correlation_to_index"`
}

// TaxEfficiencyMetrics represents ETF tax characteristics
type TaxEfficiencyMetrics struct {
	TaxEfficiencyRatio       float64   `json:"tax_efficiency_ratio"`
	CapitalGainsDistribution float64   `json:"capital_gains_distribution"`
	DividendDistribution     float64   `json:"dividend_distribution"`
	LastDistributionDate     time.Time `json:"last_distribution_date"`
	TurnoverRatio            float64   `json:"turnover_ratio"`
}

// CreationRedemptionOperation represents ETF creation/redemption activity
type CreationRedemptionOperation struct {
	ID                    string    `json:"id"`
	Symbol                string    `json:"symbol"`
	OperationType         string    `json:"operation_type"` // "creation" or "redemption"
	Units                 int       `json:"units"`
	SharesPerUnit         int       `json:"shares_per_unit"`
	TotalShares           int       `json:"total_shares"`
	NAVPerShare           float64   `json:"nav_per_share"`
	TotalValue            float64   `json:"total_value"`
	AuthorizedParticipant string    `json:"authorized_participant"`
	Timestamp             time.Time `json:"timestamp"`
	Status                string    `json:"status"`
}

// ETFPricePoint represents a price point for tracking error calculation
type ETFPricePoint struct {
	Date  time.Time
	Price float64
}

// BenchmarkPricePoint represents a benchmark price point
type BenchmarkPricePoint struct {
	Date  time.Time
	Price float64
}

// ETFAnalysisRequest represents a request for ETF analysis
type ETFAnalysisRequest struct {
	Symbol    string
	StartDate time.Time
	EndDate   time.Time
	Benchmark string
	Metrics   []string
}

// ETFAnalysisResult represents the result of ETF analysis
type ETFAnalysisResult struct {
	Symbol          string
	AnalysisDate    time.Time
	Metrics         ETFMetrics
	Recommendations []string
	Warnings        []string
}

// ETFScreeningCriteria represents criteria for ETF screening
type ETFScreeningCriteria struct {
	MinAUM           float64
	MaxExpenseRatio  float64
	MinLiquidity     float64
	MaxTrackingError float64
	Sectors          []string
	Regions          []string
	AssetClasses     []string
}

// ETFComparisonResult represents comparison between multiple ETFs
type ETFComparisonResult struct {
	ETFs           []string
	ComparisonDate time.Time
	Metrics        map[string]ETFMetrics
	Rankings       map[string]int
	BestPerformer  string
	WorstPerformer string
}
