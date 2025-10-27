package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/abdoElHodaky/tradSys/pkg/common"
	"go.uber.org/zap"
)

// ServiceMigrationTool provides automated service migration capabilities
type ServiceMigrationTool struct {
	logger   *zap.Logger
	migrator *common.ServiceMigrator
	rootDir  string
}

// NewServiceMigrationTool creates a new service migration tool
func NewServiceMigrationTool(rootDir string, logger *zap.Logger) *ServiceMigrationTool {
	return &ServiceMigrationTool{
		logger:   logger,
		migrator: common.NewServiceMigrator(logger),
		rootDir:  rootDir,
	}
}

// DiscoverServices discovers all services in the codebase
func (smt *ServiceMigrationTool) DiscoverServices() ([]string, error) {
	var services []string

	// Common service directories
	serviceDirs := []string{
		"services",
		"internal",
		"cmd",
	}

	for _, dir := range serviceDirs {
		fullPath := filepath.Join(smt.rootDir, dir)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Look for Go files that might contain services
			if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
				// Check if file contains service patterns
				if smt.containsServicePatterns(path) {
					services = append(services, path)
				}
			}

			return nil
		})

		if err != nil {
			smt.logger.Error("Error walking directory", zap.String("dir", fullPath), zap.Error(err))
		}
	}

	return services, nil
}

// containsServicePatterns checks if a file contains service-like patterns
func (smt *ServiceMigrationTool) containsServicePatterns(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	fileContent := string(content)

	// Look for service patterns
	servicePatterns := []string{
		"type.*Service struct",
		"func.*Start(",
		"func.*Stop(",
		"func.*NewService",
		"func.*New.*Service",
		"interface.*Service",
	}

	for _, pattern := range servicePatterns {
		if strings.Contains(fileContent, pattern) {
			return true
		}
	}

	return false
}

// AnalyzeService analyzes a service file for migration opportunities
func (smt *ServiceMigrationTool) AnalyzeService(filePath string) (*common.ServiceAnalysis, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return smt.migrator.AnalyzeService(string(content))
}

// GenerateMigrationReport generates a comprehensive migration report
func (smt *ServiceMigrationTool) GenerateMigrationReport(services []string) error {
	report := &MigrationReport{
		TotalServices:    len(services),
		AnalyzedServices: 0,
		MigrationNeeded:  0,
		HighPriority:     0,
		MediumPriority:   0,
		LowPriority:      0,
		Services:         make([]*ServiceMigrationInfo, 0),
	}

	for _, servicePath := range services {
		analysis, err := smt.AnalyzeService(servicePath)
		if err != nil {
			smt.logger.Error("Failed to analyze service",
				zap.String("path", servicePath),
				zap.Error(err))
			continue
		}

		report.AnalyzedServices++

		migrationInfo := &ServiceMigrationInfo{
			FilePath:        servicePath,
			ServiceName:     analysis.ServiceName,
			CurrentPatterns: analysis.Patterns,
			MigrationNeeded: analysis.NeedsMigration,
			Priority:        smt.calculatePriority(analysis),
			Recommendations: analysis.Recommendations,
			EstimatedEffort: smt.estimateEffort(analysis),
		}

		if analysis.NeedsMigration {
			report.MigrationNeeded++

			switch migrationInfo.Priority {
			case "high":
				report.HighPriority++
			case "medium":
				report.MediumPriority++
			case "low":
				report.LowPriority++
			}
		}

		report.Services = append(report.Services, migrationInfo)
	}

	// Generate report file
	return smt.writeReport(report)
}

// calculatePriority calculates migration priority based on analysis
func (smt *ServiceMigrationTool) calculatePriority(analysis *common.ServiceAnalysis) string {
	score := 0

	// High priority factors
	if analysis.HasStartStop {
		score += 3
	}
	if analysis.HasHealthCheck {
		score += 2
	}
	if analysis.UsesContext {
		score += 2
	}
	if analysis.HasMetrics {
		score += 1
	}

	// Complexity factors
	if analysis.LineCount > 500 {
		score += 2
	}
	if len(analysis.Dependencies) > 5 {
		score += 1
	}

	if score >= 6 {
		return "high"
	} else if score >= 3 {
		return "medium"
	}
	return "low"
}

// estimateEffort estimates migration effort in hours
func (smt *ServiceMigrationTool) estimateEffort(analysis *common.ServiceAnalysis) int {
	baseEffort := 2 // Base 2 hours for any migration

	// Add effort based on complexity
	if analysis.LineCount > 200 {
		baseEffort += 2
	}
	if analysis.LineCount > 500 {
		baseEffort += 4
	}
	if analysis.LineCount > 1000 {
		baseEffort += 8
	}

	// Add effort for dependencies
	baseEffort += len(analysis.Dependencies) / 2

	// Add effort for missing patterns
	if !analysis.HasStartStop {
		baseEffort += 3
	}
	if !analysis.HasHealthCheck {
		baseEffort += 2
	}
	if !analysis.UsesContext {
		baseEffort += 2
	}

	return baseEffort
}

// writeReport writes the migration report to a file
func (smt *ServiceMigrationTool) writeReport(report *MigrationReport) error {
	reportPath := filepath.Join(smt.rootDir, "SERVICE_MIGRATION_REPORT.md")

	content := smt.generateReportContent(report)

	err := os.WriteFile(reportPath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	smt.logger.Info("Migration report generated", zap.String("path", reportPath))
	return nil
}

// generateReportContent generates the markdown content for the report
func (smt *ServiceMigrationTool) generateReportContent(report *MigrationReport) string {
	var sb strings.Builder

	sb.WriteString("# 🔄 Service Migration Report\n\n")
	sb.WriteString("## 📊 Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Services Discovered**: %d\n", report.TotalServices))
	sb.WriteString(fmt.Sprintf("- **Services Analyzed**: %d\n", report.AnalyzedServices))
	sb.WriteString(fmt.Sprintf("- **Services Needing Migration**: %d\n", report.MigrationNeeded))
	sb.WriteString(fmt.Sprintf("- **High Priority**: %d\n", report.HighPriority))
	sb.WriteString(fmt.Sprintf("- **Medium Priority**: %d\n", report.MediumPriority))
	sb.WriteString(fmt.Sprintf("- **Low Priority**: %d\n", report.LowPriority))

	totalEffort := 0
	for _, service := range report.Services {
		if service.MigrationNeeded {
			totalEffort += service.EstimatedEffort
		}
	}
	sb.WriteString(fmt.Sprintf("- **Total Estimated Effort**: %d hours\n\n", totalEffort))

	// High priority services
	sb.WriteString("## 🔴 High Priority Services\n\n")
	for _, service := range report.Services {
		if service.Priority == "high" && service.MigrationNeeded {
			sb.WriteString(fmt.Sprintf("### %s\n", service.ServiceName))
			sb.WriteString(fmt.Sprintf("- **File**: `%s`\n", service.FilePath))
			sb.WriteString(fmt.Sprintf("- **Estimated Effort**: %d hours\n", service.EstimatedEffort))
			sb.WriteString("- **Current Patterns**:\n")
			for _, pattern := range service.CurrentPatterns {
				sb.WriteString(fmt.Sprintf("  - %s\n", pattern))
			}
			sb.WriteString("- **Recommendations**:\n")
			for _, rec := range service.Recommendations {
				sb.WriteString(fmt.Sprintf("  - %s\n", rec))
			}
			sb.WriteString("\n")
		}
	}

	// Medium priority services
	sb.WriteString("## 🟡 Medium Priority Services\n\n")
	for _, service := range report.Services {
		if service.Priority == "medium" && service.MigrationNeeded {
			sb.WriteString(fmt.Sprintf("### %s\n", service.ServiceName))
			sb.WriteString(fmt.Sprintf("- **File**: `%s`\n", service.FilePath))
			sb.WriteString(fmt.Sprintf("- **Estimated Effort**: %d hours\n", service.EstimatedEffort))
			sb.WriteString("- **Recommendations**: %s\n\n", strings.Join(service.Recommendations, ", "))
		}
	}

	// Low priority services
	sb.WriteString("## 🟢 Low Priority Services\n\n")
	for _, service := range report.Services {
		if service.Priority == "low" && service.MigrationNeeded {
			sb.WriteString(fmt.Sprintf("- **%s** (`%s`) - %d hours\n",
				service.ServiceName, service.FilePath, service.EstimatedEffort))
		}
	}

	// Migration guide
	sb.WriteString("\n## 🚀 Migration Guide\n\n")
	sb.WriteString("### Step 1: High Priority Services\n")
	sb.WriteString("Focus on services with existing lifecycle management that can benefit most from the BaseService pattern.\n\n")
	sb.WriteString("### Step 2: Use Migration Framework\n")
	sb.WriteString("```go\n")
	sb.WriteString("migrator := common.NewServiceMigrator(logger)\n")
	sb.WriteString("analysis := migrator.AnalyzeService(serviceCode)\n")
	sb.WriteString("plan := migrator.GenerateMigrationPlan(analysis)\n")
	sb.WriteString("```\n\n")
	sb.WriteString("### Step 3: Apply BaseService Pattern\n")
	sb.WriteString("```go\n")
	sb.WriteString("type MyService struct {\n")
	sb.WriteString("    *common.BaseService\n")
	sb.WriteString("    // existing fields\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")
	sb.WriteString("### Step 4: Test and Validate\n")
	sb.WriteString("Ensure all migrated services maintain existing functionality while gaining new capabilities.\n\n")

	return sb.String()
}

// MigrationReport represents the overall migration report
type MigrationReport struct {
	TotalServices    int                     `json:"total_services"`
	AnalyzedServices int                     `json:"analyzed_services"`
	MigrationNeeded  int                     `json:"migration_needed"`
	HighPriority     int                     `json:"high_priority"`
	MediumPriority   int                     `json:"medium_priority"`
	LowPriority      int                     `json:"low_priority"`
	Services         []*ServiceMigrationInfo `json:"services"`
}

// ServiceMigrationInfo represents migration info for a single service
type ServiceMigrationInfo struct {
	FilePath        string   `json:"file_path"`
	ServiceName     string   `json:"service_name"`
	CurrentPatterns []string `json:"current_patterns"`
	MigrationNeeded bool     `json:"migration_needed"`
	Priority        string   `json:"priority"`
	Recommendations []string `json:"recommendations"`
	EstimatedEffort int      `json:"estimated_effort"`
}

func main() {
	var (
		rootDir = flag.String("root", ".", "Root directory to scan for services")
		verbose = flag.Bool("verbose", false, "Enable verbose logging")
	)
	flag.Parse()

	// Setup logger
	var logger *zap.Logger
	var err error

	if *verbose {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Create migration tool
	tool := NewServiceMigrationTool(*rootDir, logger)

	logger.Info("Starting service discovery and migration analysis")

	// Discover services
	services, err := tool.DiscoverServices()
	if err != nil {
		logger.Fatal("Failed to discover services", zap.Error(err))
	}

	logger.Info("Services discovered", zap.Int("count", len(services)))

	// Generate migration report
	err = tool.GenerateMigrationReport(services)
	if err != nil {
		logger.Fatal("Failed to generate migration report", zap.Error(err))
	}

	logger.Info("Service migration analysis complete")
}
