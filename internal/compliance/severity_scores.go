// Package compliance provides severity scoring configuration
package compliance

// SeverityScores defines the score penalties for different violation severities
// This replaces the hardcoded switch statement with a configurable lookup table
var SeverityScores = map[ViolationSeverity]float64{
	SeverityCritical: 50.0, // Critical violations have the highest penalty
	SeverityHigh:     25.0, // High severity violations
	SeverityMedium:   10.0, // Medium severity violations
	SeverityLow:      5.0,  // Low severity violations
}

// GetSeverityScore returns the score penalty for a given severity level
// Returns 0 for unknown severity levels to maintain backward compatibility
func GetSeverityScore(severity ViolationSeverity) float64 {
	if score, exists := SeverityScores[severity]; exists {
		return score
	}
	return 0.0 // Default for unknown severities
}

// SetSeverityScore allows runtime configuration of severity scores
// This enables dynamic adjustment of compliance scoring without code changes
func SetSeverityScore(severity ViolationSeverity, score float64) {
	SeverityScores[severity] = score
}

// GetAllSeverityScores returns a copy of the current severity score mapping
// Useful for configuration management and debugging
func GetAllSeverityScores() map[ViolationSeverity]float64 {
	scores := make(map[ViolationSeverity]float64, len(SeverityScores))
	for severity, score := range SeverityScores {
		scores[severity] = score
	}
	return scores
}

// ResetSeverityScores resets severity scores to default values
func ResetSeverityScores() {
	SeverityScores[SeverityCritical] = 50.0
	SeverityScores[SeverityHigh] = 25.0
	SeverityScores[SeverityMedium] = 10.0
	SeverityScores[SeverityLow] = 5.0
}
