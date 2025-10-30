// Package islamic provides Islamic finance services for TradSys v3
package islamic

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// ShariaService provides Sharia compliance and Islamic finance services
type ShariaService struct {
	config          *ShariaConfig
	screeningEngine *ScreeningEngine
	zakatCalculator *ZakatCalculator
	complianceDB    ComplianceDatabase
	shariaBoard     *ShariaBoard
	mu              sync.RWMutex
}

// ShariaConfig holds configuration for Islamic finance services
type ShariaConfig struct {
	EnableScreening     bool
	EnableZakat         bool
	EnableShariaBoard   bool
	ScreeningRules      []ShariaRule
	ZakatRate           float64
	NisabThreshold      float64
	Currency            string
	ShariaStandard      string
	ComplianceLevel     ComplianceLevel
}

// ShariaRule represents an Islamic finance compliance rule
type ShariaRule struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	AssetTypes      []types.AssetType      `json:"asset_types"`
	Validator       func(interface{}) bool `json:"-"`
	ComplianceLevel ComplianceLevel        `json:"compliance_level"`
	IsActive        bool                   `json:"is_active"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// ComplianceLevel represents the level of Sharia compliance
type ComplianceLevel string

const (
	STRICT     ComplianceLevel = "STRICT"
	MODERATE   ComplianceLevel = "MODERATE"
	FLEXIBLE   ComplianceLevel = "FLEXIBLE"
)

// ScreeningEngine performs Sharia compliance screening
type ScreeningEngine struct {
	rules       map[string]ShariaRule
	cache       map[string]*ScreeningResult
	cacheTTL    time.Duration
	mu          sync.RWMutex
}

// ZakatCalculator calculates Zakat for Islamic portfolios
type ZakatCalculator struct {
	config      *ZakatConfig
	rateTable   map[types.AssetType]float64
	exemptions  map[types.AssetType]bool
	mu          sync.RWMutex
}

// ZakatConfig holds Zakat calculation configuration
type ZakatConfig struct {
	StandardRate    float64 // 2.5% standard rate
	NisabThreshold  float64 // Minimum wealth threshold
	Currency        string
	CalculationDate time.Time
	HijriYear       int
}

// ComplianceDatabase interface for storing compliance data
type ComplianceDatabase interface {
	GetScreeningResult(ctx context.Context, symbol string) (*ScreeningResult, error)
	SaveScreeningResult(ctx context.Context, result *ScreeningResult) error
	GetZakatRecord(ctx context.Context, userID string, year int) (*ZakatRecord, error)
	SaveZakatRecord(ctx context.Context, record *ZakatRecord) error
}

// ScreeningResult represents the result of Sharia compliance screening
type ScreeningResult struct {
	Symbol          string    `json:"symbol"`
	IsCompliant     bool      `json:"is_compliant"`
	ComplianceScore float64   `json:"compliance_score"`
	Violations      []string  `json:"violations"`
	Recommendations []string  `json:"recommendations"`
	LastUpdated     time.Time `json:"last_updated"`
	RulesApplied    []string  `json:"rules_applied"`
}

// ZakatRecord represents a Zakat calculation record
type ZakatRecord struct {
	UserID            string    `json:"user_id"`
	Year              int       `json:"year"`
	TotalWealth       float64   `json:"total_wealth"`
	ZakatableAmount   float64   `json:"zakatable_amount"`
	ZakatRate         float64   `json:"zakat_rate"`
	ZakatDue          float64   `json:"zakat_due"`
	Currency          string    `json:"currency"`
	CalculationDate   time.Time `json:"calculation_date"`
	NextDueDate       time.Time `json:"next_due_date"`
	ExemptAssets      []string  `json:"exempt_assets"`
}

// ShariaBoard represents Sharia board information
type ShariaBoard struct {
	Name           string              `json:"name"`
	Members        []ShariaBoardMember `json:"members"`
	Established    time.Time           `json:"established"`
	Certifications []string            `json:"certifications"`
	ContactInfo    ContactInfo         `json:"contact_info"`
}

// ShariaBoardMember represents a Sharia board member
type ShariaBoardMember struct {
	Name           string   `json:"name"`
	Title          string   `json:"title"`
	Qualifications []string `json:"qualifications"`
	Experience     int      `json:"experience_years"`
}

// ContactInfo represents contact information
type ContactInfo struct {
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	Website string `json:"website"`
}

// AssetScreeningInfo represents screening information for an asset
type AssetScreeningInfo struct {
	Symbol          string
	AssetType       types.AssetType
	ComplianceScore float64
	IsCompliant     bool
	Violations      []string
	LastScreened    time.Time
}

// PortfolioZakatInfo represents Zakat information for a portfolio
type PortfolioZakatInfo struct {
	PortfolioID     string
	UserID          string
	TotalValue      float64
	ZakatableValue  float64
	ZakatDue        float64
	LastCalculated  time.Time
	NextDueDate     time.Time
}
