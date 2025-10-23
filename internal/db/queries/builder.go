package queries

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Builder provides a fluent interface for building optimized queries
type Builder struct {
	db        *gorm.DB
	logger    *zap.Logger
	table     string
	fields    []string
	joins     []string
	wheres    []string
	whereArgs []interface{}
	orderBy   string
	limit     int
	offset    int
	groupBy   string
	having    string
	hints     []string // Query hints for optimization
}

// NewBuilder creates a new query builder
func NewBuilder(db *gorm.DB, logger *zap.Logger) *Builder {
	return &Builder{
		db:        db,
		logger:    logger,
		fields:    []string{"*"},
		wheres:    make([]string, 0),
		whereArgs: make([]interface{}, 0),
		limit:     -1,
		offset:    -1,
		hints:     make([]string, 0),
	}
}

// Table sets the table to query
func (b *Builder) Table(table string) *Builder {
	b.table = table
	return b
}

// GetTable returns the current table
func (b *Builder) GetTable() string {
	return b.table
}

// Select sets the fields to select
func (b *Builder) Select(fields ...string) *Builder {
	b.fields = fields
	return b
}

// GetFields returns the current fields
func (b *Builder) GetFields() []string {
	return b.fields
}

// Join adds a join clause
func (b *Builder) Join(join string) *Builder {
	b.joins = append(b.joins, join)
	return b
}

// Where adds a where condition
func (b *Builder) Where(condition string, args ...interface{}) *Builder {
	b.wheres = append(b.wheres, condition)
	b.whereArgs = append(b.whereArgs, args...)
	return b
}

// GetWheres returns the current where conditions
func (b *Builder) GetWheres() []string {
	return b.wheres
}

// GetWhereArgs returns the current where arguments
func (b *Builder) GetWhereArgs() []interface{} {
	return b.whereArgs
}

// OrderBy sets the order by clause
func (b *Builder) OrderBy(orderBy string) *Builder {
	b.orderBy = orderBy
	return b
}

// Limit sets the limit
func (b *Builder) Limit(limit int) *Builder {
	b.limit = limit
	return b
}

// Offset sets the offset
func (b *Builder) Offset(offset int) *Builder {
	b.offset = offset
	return b
}

// GroupBy sets the group by clause
func (b *Builder) GroupBy(groupBy string) *Builder {
	b.groupBy = groupBy
	return b
}

// Having sets the having clause
func (b *Builder) Having(having string) *Builder {
	b.having = having
	return b
}

// AddHint adds a query optimization hint
func (b *Builder) AddHint(hint string) *Builder {
	b.hints = append(b.hints, hint)
	return b
}

// UseIndex adds an index hint
func (b *Builder) UseIndex(indexName string) *Builder {
	return b.AddHint(fmt.Sprintf("INDEXED BY %s", indexName))
}

// Build constructs the query
func (b *Builder) Build() (string, []interface{}) {
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(b.fields, ", "), b.table)

	// Add hints
	if len(b.hints) > 0 {
		query = fmt.Sprintf("%s %s", query, strings.Join(b.hints, " "))
	}

	// Add joins
	if len(b.joins) > 0 {
		query = fmt.Sprintf("%s %s", query, strings.Join(b.joins, " "))
	}

	// Add where conditions
	if len(b.wheres) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(b.wheres, " AND "))
	}

	// Add group by
	if b.groupBy != "" {
		query = fmt.Sprintf("%s GROUP BY %s", query, b.groupBy)
	}

	// Add having
	if b.having != "" {
		query = fmt.Sprintf("%s HAVING %s", query, b.having)
	}

	// Add order by
	if b.orderBy != "" {
		query = fmt.Sprintf("%s ORDER BY %s", query, b.orderBy)
	}

	// Add limit
	if b.limit >= 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, b.limit)
	}

	// Add offset
	if b.offset >= 0 {
		query = fmt.Sprintf("%s OFFSET %d", query, b.offset)
	}

	return query, b.whereArgs
}

// Execute runs the query and scans results into dest
func (b *Builder) Execute(dest interface{}) error {
	query, args := b.Build()

	start := time.Now()
	result := b.db.Raw(query, args...).Scan(dest)
	duration := time.Since(start)

	// Log query performance
	if duration > 100*time.Millisecond {
		b.logger.Warn("Slow query detected",
			zap.String("query", query),
			zap.Duration("duration", duration))
	}

	if result.Error != nil {
		b.logger.Error("Query execution failed",
			zap.String("query", query),
			zap.Error(result.Error))
	}

	return result.Error
}

// Count returns the count of records matching the query
func (b *Builder) Count() (int64, error) {
	var count int64
	countBuilder := &Builder{
		db:        b.db,
		logger:    b.logger,
		table:     b.table,
		fields:    []string{"COUNT(*) as count"},
		joins:     b.joins,
		wheres:    b.wheres,
		whereArgs: b.whereArgs,
		hints:     b.hints,
	}

	query, args := countBuilder.Build()
	result := b.db.Raw(query, args...).Scan(&count)
	return count, result.Error
}

// First executes the query and returns the first result
func (b *Builder) First(dest interface{}) error {
	b.Limit(1)
	return b.Execute(dest)
}
