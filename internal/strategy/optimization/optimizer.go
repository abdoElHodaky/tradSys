package optimization

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/workerpool"
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrInvalidParameters = errors.New("invalid parameters")
	ErrOptimizationFailed = errors.New("optimization failed")
)

// OptimizationMethod represents the optimization method
type OptimizationMethod string

// Optimization methods
const (
	OptimizationMethodGrid     OptimizationMethod = "grid"
	OptimizationMethodRandom   OptimizationMethod = "random"
	OptimizationMethodGenetic  OptimizationMethod = "genetic"
	OptimizationMethodBayesian OptimizationMethod = "bayesian"
)

// ParameterRange represents a range for a parameter
type ParameterRange struct {
	// Min is the minimum value
	Min float64

	// Max is the maximum value
	Max float64

	// Step is the step size (for grid search)
	Step float64

	// IsInteger indicates if the parameter is an integer
	IsInteger bool
}

// OptimizationConfig represents the configuration for optimization
type OptimizationConfig struct {
	// StrategyType is the type of strategy to optimize
	StrategyType string

	// StrategyName is the name of the strategy
	StrategyName string

	// Symbols are the trading symbols
	Symbols []string

	// Parameters are the parameters to optimize
	Parameters map[string]ParameterRange

	// FixedParameters are parameters that are not optimized
	FixedParameters map[string]interface{}

	// Method is the optimization method
	Method OptimizationMethod

	// Iterations is the number of iterations
	Iterations int

	// PopulationSize is the population size (for genetic algorithm)
	PopulationSize int

	// MutationRate is the mutation rate (for genetic algorithm)
	MutationRate float64

	// CrossoverRate is the crossover rate (for genetic algorithm)
	CrossoverRate float64

	// Metric is the metric to optimize
	Metric string

	// Maximize indicates if the metric should be maximized
	Maximize bool
}

// ParameterSet represents a set of parameters
type ParameterSet struct {
	// Parameters are the parameter values
	Parameters map[string]interface{}

	// Metrics are the evaluation metrics
	Metrics map[string]float64

	// Rank is the rank of the parameter set
	Rank int
}

// OptimizationResult represents the result of optimization
type OptimizationResult struct {
	// BestParameters is the best parameter set
	BestParameters map[string]interface{}

	// BestMetrics are the metrics for the best parameter set
	BestMetrics map[string]float64

	// AllResults are all parameter sets evaluated
	AllResults []*ParameterSet

	// Method is the optimization method used
	Method OptimizationMethod

	// Iterations is the number of iterations performed
	Iterations int

	// Duration is the duration of the optimization
	Duration time.Duration
}

// StrategyOptimizer optimizes strategy parameters
type StrategyOptimizer struct {
	// Factory is the strategy factory
	factory strategy.StrategyFactory

	// WorkerPool is the worker pool
	workerPool *workerpool.WorkerPoolFactory

	// Logger
	logger *zap.Logger

	// Evaluator is the strategy evaluator
	evaluator *StrategyEvaluator
}

// NewStrategyOptimizer creates a new StrategyOptimizer
func NewStrategyOptimizer(
	factory strategy.StrategyFactory,
	evaluator *StrategyEvaluator,
	workerPool *workerpool.WorkerPoolFactory,
	logger *zap.Logger,
) *StrategyOptimizer {
	return &StrategyOptimizer{
		factory:    factory,
		evaluator:  evaluator,
		workerPool: workerPool,
		logger:     logger,
	}
}

// Optimize optimizes strategy parameters
func (o *StrategyOptimizer) Optimize(
	ctx context.Context,
	config OptimizationConfig,
) (*OptimizationResult, error) {
	// Validate configuration
	if err := o.validateConfig(config); err != nil {
		return nil, err
	}

	startTime := time.Now()

	var result *OptimizationResult
	var err error

	// Choose optimization method
	switch config.Method {
	case OptimizationMethodGrid:
		result, err = o.gridSearch(ctx, config)
	case OptimizationMethodRandom:
		result, err = o.randomSearch(ctx, config)
	case OptimizationMethodGenetic:
		result, err = o.geneticAlgorithm(ctx, config)
	default:
		return nil, errors.New("unsupported optimization method")
	}

	if err != nil {
		return nil, err
	}

	// Set duration
	result.Duration = time.Since(startTime)

	return result, nil
}

// validateConfig validates the optimization configuration
func (o *StrategyOptimizer) validateConfig(config OptimizationConfig) error {
	// Check if strategy type is valid
	availableTypes := o.factory.GetAvailableStrategyTypes()
	validType := false
	for _, t := range availableTypes {
		if t == config.StrategyType {
			validType = true
			break
		}
	}
	if !validType {
		return errors.New("invalid strategy type")
	}

	// Check if parameters are valid
	if len(config.Parameters) == 0 {
		return errors.New("no parameters to optimize")
	}

	// Check parameter ranges
	for name, r := range config.Parameters {
		if r.Min > r.Max {
			return errors.New("invalid parameter range: min > max")
		}
		if r.Step <= 0 && config.Method == OptimizationMethodGrid {
			return errors.New("invalid parameter step: must be positive for grid search")
		}
	}

	// Check iterations
	if config.Iterations <= 0 {
		return errors.New("iterations must be positive")
	}

	// Check population size for genetic algorithm
	if config.Method == OptimizationMethodGenetic && config.PopulationSize <= 0 {
		return errors.New("population size must be positive for genetic algorithm")
	}

	return nil
}

// gridSearch performs grid search optimization
func (o *StrategyOptimizer) gridSearch(
	ctx context.Context,
	config OptimizationConfig,
) (*OptimizationResult, error) {
	// Generate parameter combinations
	parameterSets := o.generateGridParameterSets(config)

	// Evaluate parameter sets
	results, err := o.evaluateParameterSets(ctx, config, parameterSets)
	if err != nil {
		return nil, err
	}

	// Sort results
	sortedResults := o.sortResults(results, config.Metric, config.Maximize)

	// Create optimization result
	result := &OptimizationResult{
		BestParameters: sortedResults[0].Parameters,
		BestMetrics:    sortedResults[0].Metrics,
		AllResults:     sortedResults,
		Method:         config.Method,
		Iterations:     len(sortedResults),
	}

	return result, nil
}

// randomSearch performs random search optimization
func (o *StrategyOptimizer) randomSearch(
	ctx context.Context,
	config OptimizationConfig,
) (*OptimizationResult, error) {
	// Generate random parameter sets
	parameterSets := o.generateRandomParameterSets(config)

	// Evaluate parameter sets
	results, err := o.evaluateParameterSets(ctx, config, parameterSets)
	if err != nil {
		return nil, err
	}

	// Sort results
	sortedResults := o.sortResults(results, config.Metric, config.Maximize)

	// Create optimization result
	result := &OptimizationResult{
		BestParameters: sortedResults[0].Parameters,
		BestMetrics:    sortedResults[0].Metrics,
		AllResults:     sortedResults,
		Method:         config.Method,
		Iterations:     len(sortedResults),
	}

	return result, nil
}

// geneticAlgorithm performs genetic algorithm optimization
func (o *StrategyOptimizer) geneticAlgorithm(
	ctx context.Context,
	config OptimizationConfig,
) (*OptimizationResult, error) {
	// Initialize population
	population := o.generateRandomParameterSets(config)
	if len(population) > config.PopulationSize {
		population = population[:config.PopulationSize]
	}

	// Evaluate initial population
	results, err := o.evaluateParameterSets(ctx, config, population)
	if err != nil {
		return nil, err
	}

	// Sort results
	sortedResults := o.sortResults(results, config.Metric, config.Maximize)

	// Perform genetic algorithm iterations
	for i := 0; i < config.Iterations; i++ {
		// Select parents
		parents := o.selectParents(sortedResults, config.PopulationSize/2)

		// Create offspring
		offspring := o.createOffspring(parents, config)

		// Evaluate offspring
		offspringResults, err := o.evaluateParameterSets(ctx, config, offspring)
		if err != nil {
			return nil, err
		}

		// Combine parents and offspring
		combined := append(sortedResults, offspringResults...)

		// Sort combined results
		sortedResults = o.sortResults(combined, config.Metric, config.Maximize)

		// Truncate to population size
		if len(sortedResults) > config.PopulationSize {
			sortedResults = sortedResults[:config.PopulationSize]
		}
	}

	// Create optimization result
	result := &OptimizationResult{
		BestParameters: sortedResults[0].Parameters,
		BestMetrics:    sortedResults[0].Metrics,
		AllResults:     sortedResults,
		Method:         config.Method,
		Iterations:     config.Iterations,
	}

	return result, nil
}

// generateGridParameterSets generates parameter sets for grid search
func (o *StrategyOptimizer) generateGridParameterSets(config OptimizationConfig) []map[string]interface{} {
	// Calculate number of combinations
	numCombinations := 1
	for _, r := range config.Parameters {
		steps := int((r.Max-r.Min)/r.Step) + 1
		numCombinations *= steps
	}

	// Limit number of combinations
	if numCombinations > 10000 {
		o.logger.Warn("Grid search would generate too many combinations, limiting to 10000")
		numCombinations = 10000
	}

	// Generate parameter sets
	parameterSets := make([]map[string]interface{}, 0, numCombinations)
	o.generateGridParameterSetsRecursive(
		config.Parameters,
		make(map[string]interface{}),
		&parameterSets,
		config.FixedParameters,
		numCombinations,
	)

	return parameterSets
}

// generateGridParameterSetsRecursive recursively generates parameter sets for grid search
func (o *StrategyOptimizer) generateGridParameterSetsRecursive(
	ranges map[string]ParameterRange,
	current map[string]interface{},
	result *[]map[string]interface{},
	fixed map[string]interface{},
	limit int,
) {
	if len(ranges) == 0 {
		// Add fixed parameters
		paramSet := make(map[string]interface{})
		for k, v := range current {
			paramSet[k] = v
		}
		for k, v := range fixed {
			paramSet[k] = v
		}
		*result = append(*result, paramSet)
		return
	}

	if len(*result) >= limit {
		return
	}

	// Get first parameter
	var paramName string
	var paramRange ParameterRange
	for name, r := range ranges {
		paramName = name
		paramRange = r
		break
	}

	// Remove parameter from ranges
	remainingRanges := make(map[string]ParameterRange)
	for name, r := range ranges {
		if name != paramName {
			remainingRanges[name] = r
		}
	}

	// Generate values for parameter
	steps := int((paramRange.Max-paramRange.Min)/paramRange.Step) + 1
	for i := 0; i < steps; i++ {
		value := paramRange.Min + float64(i)*paramRange.Step
		if paramRange.IsInteger {
			value = float64(int(value))
		}

		// Add parameter to current set
		current[paramName] = value

		// Recursively generate remaining parameters
		o.generateGridParameterSetsRecursive(remainingRanges, current, result, fixed, limit)

		if len(*result) >= limit {
			return
		}
	}
}

// generateRandomParameterSets generates random parameter sets
func (o *StrategyOptimizer) generateRandomParameterSets(config OptimizationConfig) []map[string]interface{} {
	parameterSets := make([]map[string]interface{}, 0, config.Iterations)

	for i := 0; i < config.Iterations; i++ {
		// Generate random values for parameters
		params := make(map[string]interface{})
		for name, r := range config.Parameters {
			value := r.Min + rand.Float64()*(r.Max-r.Min)
			if r.IsInteger {
				value = float64(int(value))
			}
			params[name] = value
		}

		// Add fixed parameters
		for k, v := range config.FixedParameters {
			params[k] = v
		}

		parameterSets = append(parameterSets, params)
	}

	return parameterSets
}

// evaluateParameterSets evaluates multiple parameter sets
func (o *StrategyOptimizer) evaluateParameterSets(
	ctx context.Context,
	config OptimizationConfig,
	parameterSets []map[string]interface{},
) ([]*ParameterSet, error) {
	results := make([]*ParameterSet, len(parameterSets))
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create a worker pool
	pool, err := o.workerPool.GetWorkerPool("strategy-optimizer", 8)
	if err != nil {
		return nil, err
	}

	// Evaluate each parameter set
	for i, params := range parameterSets {
		wg.Add(1)
		i := i
		params := params

		err := pool.Submit(func() {
			defer wg.Done()

			// Create strategy config
			strategyConfig := strategy.StrategyConfig{
				Name:       config.StrategyName,
				Type:       config.StrategyType,
				Symbols:    config.Symbols,
				Parameters: params,
			}

			// Evaluate strategy
			metrics, err := o.evaluator.Evaluate(ctx, strategyConfig)
			if err != nil {
				o.logger.Error("Failed to evaluate strategy",
					zap.Error(err),
					zap.Any("parameters", params))
				return
			}

			// Create parameter set
			paramSet := &ParameterSet{
				Parameters: params,
				Metrics:    metrics,
			}

			// Store result
			mu.Lock()
			results[i] = paramSet
			mu.Unlock()
		})

		if err != nil {
			o.logger.Error("Failed to submit evaluation task",
				zap.Error(err),
				zap.Any("parameters", params))
		}
	}

	// Wait for all evaluations to complete
	wg.Wait()

	// Filter out nil results
	validResults := make([]*ParameterSet, 0, len(results))
	for _, r := range results {
		if r != nil {
			validResults = append(validResults, r)
		}
	}

	if len(validResults) == 0 {
		return nil, ErrOptimizationFailed
	}

	return validResults, nil
}

// sortResults sorts parameter sets by metric
func (o *StrategyOptimizer) sortResults(
	results []*ParameterSet,
	metric string,
	maximize bool,
) []*ParameterSet {
	// Create a copy of results
	sortedResults := make([]*ParameterSet, len(results))
	copy(sortedResults, results)

	// Sort by metric
	sort.Slice(sortedResults, func(i, j int) bool {
		metricI := sortedResults[i].Metrics[metric]
		metricJ := sortedResults[j].Metrics[metric]

		if maximize {
			return metricI > metricJ
		}
		return metricI < metricJ
	})

	// Set ranks
	for i, r := range sortedResults {
		r.Rank = i + 1
	}

	return sortedResults
}

// selectParents selects parents for genetic algorithm
func (o *StrategyOptimizer) selectParents(
	population []*ParameterSet,
	numParents int,
) []*ParameterSet {
	if numParents > len(population) {
		numParents = len(population)
	}

	// Use tournament selection
	parents := make([]*ParameterSet, numParents)
	for i := 0; i < numParents; i++ {
		// Select random individuals for tournament
		tournamentSize := 3
		if tournamentSize > len(population) {
			tournamentSize = len(population)
		}

		tournament := make([]*ParameterSet, tournamentSize)
		for j := 0; j < tournamentSize; j++ {
			tournament[j] = population[rand.Intn(len(population))]
		}

		// Find best individual in tournament
		best := tournament[0]
		for j := 1; j < tournamentSize; j++ {
			if tournament[j].Rank < best.Rank {
				best = tournament[j]
			}
		}

		parents[i] = best
	}

	return parents
}

// createOffspring creates offspring for genetic algorithm
func (o *StrategyOptimizer) createOffspring(
	parents []*ParameterSet,
	config OptimizationConfig,
) []map[string]interface{} {
	offspring := make([]map[string]interface{}, 0, len(parents))

	// Create offspring through crossover and mutation
	for i := 0; i < len(parents); i++ {
		for j := i + 1; j < len(parents); j++ {
			// Perform crossover
			if rand.Float64() < config.CrossoverRate {
				child1, child2 := o.crossover(parents[i], parents[j], config)
				offspring = append(offspring, child1, child2)
			} else {
				// No crossover, just copy parents
				child1 := o.copyParameterSet(parents[i].Parameters)
				child2 := o.copyParameterSet(parents[j].Parameters)
				offspring = append(offspring, child1, child2)
			}
		}
	}

	// Perform mutation
	for i := range offspring {
		o.mutate(offspring[i], config)
	}

	return offspring
}

// crossover performs crossover between two parameter sets
func (o *StrategyOptimizer) crossover(
	parent1, parent2 *ParameterSet,
	config OptimizationConfig,
) (map[string]interface{}, map[string]interface{}) {
	child1 := make(map[string]interface{})
	child2 := make(map[string]interface{})

	// Perform uniform crossover
	for name, r := range config.Parameters {
		if rand.Float64() < 0.5 {
			// Child 1 gets parameter from parent 1, child 2 from parent 2
			child1[name] = parent1.Parameters[name]
			child2[name] = parent2.Parameters[name]
		} else {
			// Child 1 gets parameter from parent 2, child 2 from parent 1
			child1[name] = parent2.Parameters[name]
			child2[name] = parent1.Parameters[name]
		}
	}

	// Add fixed parameters
	for k, v := range config.FixedParameters {
		child1[k] = v
		child2[k] = v
	}

	return child1, child2
}

// mutate performs mutation on a parameter set
func (o *StrategyOptimizer) mutate(
	params map[string]interface{},
	config OptimizationConfig,
) {
	for name, r := range config.Parameters {
		// Perform mutation with probability mutationRate
		if rand.Float64() < config.MutationRate {
			// Generate new random value
			value := r.Min + rand.Float64()*(r.Max-r.Min)
			if r.IsInteger {
				value = float64(int(value))
			}
			params[name] = value
		}
	}
}

// copyParameterSet creates a copy of a parameter set
func (o *StrategyOptimizer) copyParameterSet(params map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range params {
		copy[k] = v
	}
	return copy
}

