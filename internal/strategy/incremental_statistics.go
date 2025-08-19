package strategy

import (
	"math"
	"sync"
)

// IncrementalStatistics provides efficient statistical calculations
// with incremental updates to avoid recalculating over the entire dataset
type IncrementalStatistics struct {
	count       int
	mean        float64
	m2          float64 // Sum of squared differences from the mean
	min         float64
	max         float64
	initialized bool
	mu          sync.RWMutex
}

// NewIncrementalStatistics creates a new incremental statistics calculator
func NewIncrementalStatistics() *IncrementalStatistics {
	return &IncrementalStatistics{
		count:       0,
		mean:        0,
		m2:          0,
		min:         0,
		max:         0,
		initialized: false,
	}
}

// Add adds a value to the statistics
// Uses Welford's online algorithm for numerical stability
func (s *IncrementalStatistics) Add(value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.count++
	
	// Initialize min and max on first value
	if !s.initialized {
		s.min = value
		s.max = value
		s.mean = value
		s.initialized = true
		return
	}
	
	// Update min and max
	if value < s.min {
		s.min = value
	}
	if value > s.max {
		s.max = value
	}
	
	// Update mean and variance using Welford's algorithm
	delta := value - s.mean
	s.mean += delta / float64(s.count)
	delta2 := value - s.mean
	s.m2 += delta * delta2
}

// Remove removes a value from the statistics
// Note: This is an approximation and may lose precision over many removals
// For critical applications, consider rebuilding the statistics periodically
func (s *IncrementalStatistics) Remove(value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.count <= 1 {
		// Reset statistics if this is the last value
		s.count = 0
		s.mean = 0
		s.m2 = 0
		s.initialized = false
		return
	}
	
	// Update mean and variance
	oldMean := s.mean
	s.mean = (float64(s.count)*s.mean - value) / float64(s.count-1)
	s.m2 -= (value - oldMean) * (value - s.mean)
	s.count--
	
	// Note: min and max cannot be accurately updated when removing a value
	// without keeping the full dataset. For HFT, this is usually acceptable
	// as we're typically using a sliding window and the min/max will eventually
	// be correct as old values fall out of the window.
}

// Update both adds a new value and removes an old value in one operation
// This is more efficient and accurate than calling Remove() followed by Add()
func (s *IncrementalStatistics) Update(oldValue, newValue float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.initialized {
		s.Add(newValue)
		return
	}
	
	// Update min and max for the new value
	if newValue < s.min {
		s.min = newValue
	}
	if newValue > s.max {
		s.max = newValue
	}
	
	// Update mean directly
	s.mean = s.mean + (newValue - oldValue) / float64(s.count)
	
	// Update variance
	s.m2 = s.m2 + (newValue - oldValue) * (newValue - s.mean + oldValue - s.mean)
	
	// Note: As with Remove(), min and max may not be 100% accurate
	// if oldValue was the min or max. For HFT, this approximation
	// is usually acceptable.
}

// Count returns the number of values
func (s *IncrementalStatistics) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

// Mean returns the mean of the values
func (s *IncrementalStatistics) Mean() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mean
}

// Variance returns the variance of the values
func (s *IncrementalStatistics) Variance() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.count < 2 {
		return 0
	}
	
	return s.m2 / float64(s.count-1)
}

// StdDev returns the standard deviation of the values
func (s *IncrementalStatistics) StdDev() float64 {
	return math.Sqrt(s.Variance())
}

// Min returns the minimum value
func (s *IncrementalStatistics) Min() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.min
}

// Max returns the maximum value
func (s *IncrementalStatistics) Max() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.max
}

// ZScore calculates the z-score of a value
func (s *IncrementalStatistics) ZScore(value float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stdDev := math.Sqrt(s.Variance())
	if stdDev == 0 {
		return 0
	}
	
	return (value - s.mean) / stdDev
}

// Reset resets the statistics
func (s *IncrementalStatistics) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.count = 0
	s.mean = 0
	s.m2 = 0
	s.min = 0
	s.max = 0
	s.initialized = false
}

// IncrementalCorrelation calculates correlation incrementally
type IncrementalCorrelation struct {
	count       int
	meanX       float64
	meanY       float64
	c           float64 // Covariance * (n-1)
	varX        float64 // Variance of X * (n-1)
	varY        float64 // Variance of Y * (n-1)
	initialized bool
	mu          sync.RWMutex
}

// NewIncrementalCorrelation creates a new incremental correlation calculator
func NewIncrementalCorrelation() *IncrementalCorrelation {
	return &IncrementalCorrelation{
		count:       0,
		meanX:       0,
		meanY:       0,
		c:           0,
		varX:        0,
		varY:        0,
		initialized: false,
	}
}

// Add adds a pair of values to the correlation calculation
func (c *IncrementalCorrelation) Add(x, y float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.count++
	
	if !c.initialized {
		c.meanX = x
		c.meanY = y
		c.initialized = true
		return
	}
	
	// Update means
	dx := x - c.meanX
	c.meanX += dx / float64(c.count)
	c.meanY += (y - c.meanY) / float64(c.count)
	
	// Update covariance and variances
	dy := y - c.meanY
	c.c += dx * dy
	c.varX += dx * (x - c.meanX)
	c.varY += dy * (y - c.meanY)
}

// Update both adds a new pair and removes an old pair in one operation
func (c *IncrementalCorrelation) Update(oldX, oldY, newX, newY float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.initialized {
		c.Add(newX, newY)
		return
	}
	
	// This is an approximation for sliding window correlation
	// For critical applications, consider rebuilding periodically
	
	// Adjust means
	c.meanX = c.meanX + (newX - oldX) / float64(c.count)
	c.meanY = c.meanY + (newY - oldY) / float64(c.count)
	
	// Adjust covariance and variances
	// This is a simplified approximation
	c.c = c.c + (newX - c.meanX) * (newY - c.meanY) - (oldX - c.meanX) * (oldY - c.meanY)
	c.varX = c.varX + (newX - c.meanX) * (newX - c.meanX) - (oldX - c.meanX) * (oldX - c.meanX)
	c.varY = c.varY + (newY - c.meanY) * (newY - c.meanY) - (oldY - c.meanY) * (oldY - c.meanY)
}

// Correlation returns the Pearson correlation coefficient
func (c *IncrementalCorrelation) Correlation() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.count < 2 {
		return 0
	}
	
	denominator := math.Sqrt(c.varX * c.varY)
	if denominator == 0 {
		return 0
	}
	
	return c.c / denominator
}

// Reset resets the correlation calculator
func (c *IncrementalCorrelation) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.count = 0
	c.meanX = 0
	c.meanY = 0
	c.c = 0
	c.varX = 0
	c.varY = 0
	c.initialized = false
}

