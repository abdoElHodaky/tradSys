package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.uber.org/zap"
)

// BaseRepository provides common repository functionality
type BaseRepository struct {
	db     *sql.DB
	logger *zap.Logger
	table  string
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(db *sql.DB, logger *zap.Logger, table string) *BaseRepository {
	return &BaseRepository{
		db:     db,
		logger: logger,
		table:  table,
	}
}

// RepositoryInterface defines the common repository interface
type RepositoryInterface[T any] interface {
	Create(ctx context.Context, entity *T) error
	GetByID(ctx context.Context, id string) (*T, error)
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*T, error)
	Count(ctx context.Context) (int64, error)
}

// OptimizedRepository provides optimized CRUD operations
type OptimizedRepository[T any] struct {
	*BaseRepository
	entityType reflect.Type
}

// NewOptimizedRepository creates a new optimized repository
func NewOptimizedRepository[T any](db *sql.DB, logger *zap.Logger, table string) *OptimizedRepository[T] {
	var zero T
	return &OptimizedRepository[T]{
		BaseRepository: NewBaseRepository(db, logger, table),
		entityType:     reflect.TypeOf(zero),
	}
}

// Create inserts a new entity
func (r *OptimizedRepository[T]) Create(ctx context.Context, entity *T) error {
	startTime := time.Now()
	defer func() {
		r.logger.Debug("Repository operation completed",
			zap.String("operation", "create"),
			zap.String("table", r.table),
			zap.Duration("duration", time.Since(startTime)))
	}()

	fields, values, placeholders := r.extractFieldsAndValues(entity, false)
	
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		r.table,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := r.db.ExecContext(ctx, query, values...)
	if err != nil {
		r.logger.Error("Failed to create entity",
			zap.String("table", r.table),
			zap.Error(err))
		return fmt.Errorf("failed to create entity in %s: %w", r.table, err)
	}

	return nil
}

// GetByID retrieves an entity by ID
func (r *OptimizedRepository[T]) GetByID(ctx context.Context, id string) (*T, error) {
	startTime := time.Now()
	defer func() {
		r.logger.Debug("Repository operation completed",
			zap.String("operation", "get_by_id"),
			zap.String("table", r.table),
			zap.Duration("duration", time.Since(startTime)))
	}()

	query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", r.table)
	
	row := r.db.QueryRowContext(ctx, query, id)
	
	entity := new(T)
	err := r.scanRowIntoEntity(row, entity)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entity not found in %s with id %s", r.table, id)
		}
		r.logger.Error("Failed to get entity by ID",
			zap.String("table", r.table),
			zap.String("id", id),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get entity from %s: %w", r.table, err)
	}

	return entity, nil
}

// Update updates an existing entity
func (r *OptimizedRepository[T]) Update(ctx context.Context, entity *T) error {
	startTime := time.Now()
	defer func() {
		r.logger.Debug("Repository operation completed",
			zap.String("operation", "update"),
			zap.String("table", r.table),
			zap.Duration("duration", time.Since(startTime)))
	}()

	fields, values, _ := r.extractFieldsAndValues(entity, true)
	
	// Build SET clause
	setClauses := make([]string, len(fields))
	for i, field := range fields {
		setClauses[i] = fmt.Sprintf("%s = $%d", field, i+1)
	}
	
	// Assume ID is the last value for WHERE clause
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d",
		r.table,
		strings.Join(setClauses[:len(setClauses)-1], ", "),
		len(values),
	)

	result, err := r.db.ExecContext(ctx, query, values...)
	if err != nil {
		r.logger.Error("Failed to update entity",
			zap.String("table", r.table),
			zap.Error(err))
		return fmt.Errorf("failed to update entity in %s: %w", r.table, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no entity found to update in %s", r.table)
	}

	return nil
}

// Delete removes an entity by ID
func (r *OptimizedRepository[T]) Delete(ctx context.Context, id string) error {
	startTime := time.Now()
	defer func() {
		r.logger.Debug("Repository operation completed",
			zap.String("operation", "delete"),
			zap.String("table", r.table),
			zap.Duration("duration", time.Since(startTime)))
	}()

	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", r.table)
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete entity",
			zap.String("table", r.table),
			zap.String("id", id),
			zap.Error(err))
		return fmt.Errorf("failed to delete entity from %s: %w", r.table, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no entity found to delete in %s with id %s", r.table, id)
	}

	return nil
}

// List retrieves entities with pagination
func (r *OptimizedRepository[T]) List(ctx context.Context, limit, offset int) ([]*T, error) {
	startTime := time.Now()
	defer func() {
		r.logger.Debug("Repository operation completed",
			zap.String("operation", "list"),
			zap.String("table", r.table),
			zap.Duration("duration", time.Since(startTime)))
	}()

	query := fmt.Sprintf("SELECT * FROM %s ORDER BY created_at DESC LIMIT $1 OFFSET $2", r.table)
	
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		r.logger.Error("Failed to list entities",
			zap.String("table", r.table),
			zap.Error(err))
		return nil, fmt.Errorf("failed to list entities from %s: %w", r.table, err)
	}
	defer rows.Close()

	var entities []*T
	for rows.Next() {
		entity := new(T)
		err := r.scanRowIntoEntity(rows, entity)
		if err != nil {
			r.logger.Error("Failed to scan entity",
				zap.String("table", r.table),
				zap.Error(err))
			continue
		}
		entities = append(entities, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return entities, nil
}

// Count returns the total number of entities
func (r *OptimizedRepository[T]) Count(ctx context.Context) (int64, error) {
	startTime := time.Now()
	defer func() {
		r.logger.Debug("Repository operation completed",
			zap.String("operation", "count"),
			zap.String("table", r.table),
			zap.Duration("duration", time.Since(startTime)))
	}()

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", r.table)
	
	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		r.logger.Error("Failed to count entities",
			zap.String("table", r.table),
			zap.Error(err))
		return 0, fmt.Errorf("failed to count entities in %s: %w", r.table, err)
	}

	return count, nil
}

// FindByField finds entities by a specific field value
func (r *OptimizedRepository[T]) FindByField(ctx context.Context, field string, value interface{}, limit int) ([]*T, error) {
	startTime := time.Now()
	defer func() {
		r.logger.Debug("Repository operation completed",
			zap.String("operation", "find_by_field"),
			zap.String("table", r.table),
			zap.String("field", field),
			zap.Duration("duration", time.Since(startTime)))
	}()

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1 ORDER BY created_at DESC LIMIT $2", r.table, field)
	
	rows, err := r.db.QueryContext(ctx, query, value, limit)
	if err != nil {
		r.logger.Error("Failed to find entities by field",
			zap.String("table", r.table),
			zap.String("field", field),
			zap.Error(err))
		return nil, fmt.Errorf("failed to find entities by field in %s: %w", r.table, err)
	}
	defer rows.Close()

	var entities []*T
	for rows.Next() {
		entity := new(T)
		err := r.scanRowIntoEntity(rows, entity)
		if err != nil {
			r.logger.Error("Failed to scan entity",
				zap.String("table", r.table),
				zap.Error(err))
			continue
		}
		entities = append(entities, entity)
	}

	return entities, nil
}

// FindByTimeRange finds entities within a time range
func (r *OptimizedRepository[T]) FindByTimeRange(ctx context.Context, timeField string, from, to time.Time, limit int) ([]*T, error) {
	startTime := time.Now()
	defer func() {
		r.logger.Debug("Repository operation completed",
			zap.String("operation", "find_by_time_range"),
			zap.String("table", r.table),
			zap.String("time_field", timeField),
			zap.Duration("duration", time.Since(startTime)))
	}()

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s BETWEEN $1 AND $2 ORDER BY %s DESC LIMIT $3", r.table, timeField, timeField)
	
	rows, err := r.db.QueryContext(ctx, query, from, to, limit)
	if err != nil {
		r.logger.Error("Failed to find entities by time range",
			zap.String("table", r.table),
			zap.String("time_field", timeField),
			zap.Error(err))
		return nil, fmt.Errorf("failed to find entities by time range in %s: %w", r.table, err)
	}
	defer rows.Close()

	var entities []*T
	for rows.Next() {
		entity := new(T)
		err := r.scanRowIntoEntity(rows, entity)
		if err != nil {
			r.logger.Error("Failed to scan entity",
				zap.String("table", r.table),
				zap.Error(err))
			continue
		}
		entities = append(entities, entity)
	}

	return entities, nil
}

// BatchCreate inserts multiple entities in a single transaction
func (r *OptimizedRepository[T]) BatchCreate(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}

	startTime := time.Now()
	defer func() {
		r.logger.Debug("Repository operation completed",
			zap.String("operation", "batch_create"),
			zap.String("table", r.table),
			zap.Int("count", len(entities)),
			zap.Duration("duration", time.Since(startTime)))
	}()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	fields, _, placeholders := r.extractFieldsAndValues(entities[0], false)
	
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		r.table,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, entity := range entities {
		_, values, _ := r.extractFieldsAndValues(entity, false)
		_, err := stmt.ExecContext(ctx, values...)
		if err != nil {
			return fmt.Errorf("failed to execute batch insert: %w", err)
		}
	}

	return tx.Commit()
}

// Helper methods

// extractFieldsAndValues extracts field names and values from an entity using reflection
func (r *OptimizedRepository[T]) extractFieldsAndValues(entity *T, includeID bool) ([]string, []interface{}, []string) {
	v := reflect.ValueOf(entity).Elem()
	t := v.Type()

	var fields []string
	var values []interface{}
	var placeholders []string

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Skip unexported fields
		if !value.CanInterface() {
			continue
		}

		// Get field name from db tag or use field name
		fieldName := field.Tag.Get("db")
		if fieldName == "" {
			fieldName = strings.ToLower(field.Name)
		}

		// Skip ID field for inserts unless explicitly included
		if fieldName == "id" && !includeID {
			continue
		}

		fields = append(fields, fieldName)
		values = append(values, value.Interface())
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(placeholders)+1))
	}

	return fields, values, placeholders
}

// scanRowIntoEntity scans a database row into an entity using reflection
func (r *OptimizedRepository[T]) scanRowIntoEntity(scanner interface{ Scan(...interface{}) error }, entity *T) error {
	v := reflect.ValueOf(entity).Elem()
	_ = v.Type() // Type information available if needed for debugging

	var scanArgs []interface{}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.CanAddr() {
			scanArgs = append(scanArgs, field.Addr().Interface())
		}
	}

	return scanner.Scan(scanArgs...)
}

// ExecuteInTransaction executes a function within a database transaction
func (r *BaseRepository) ExecuteInTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// GetDB returns the database connection
func (r *BaseRepository) GetDB() *sql.DB {
	return r.db
}

// GetLogger returns the logger
func (r *BaseRepository) GetLogger() *zap.Logger {
	return r.logger
}

// GetTable returns the table name
func (r *BaseRepository) GetTable() string {
	return r.table
}
