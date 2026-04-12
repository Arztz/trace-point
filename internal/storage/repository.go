package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Repository provides CRUD operations for database entities.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository instance.
func NewRepository(db *Database) *Repository {
	return &Repository{
		db: db.db,
	}
}

// CreateSpikeEvent inserts a new spike event into the database.
func (r *Repository) CreateSpikeEvent(ctx context.Context, event *SpikeEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO spike_events (
			id, timestamp, pod_name, namespace,
			cpu_usage_percent, cpu_limit_percent,
			ram_usage_percent, ram_limit_percent,
			threshold_percent, moving_average_percent,
			route_name, trace_id, culprit_function, culprit_file_path,
			alert_sent, cooldown_end, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		event.ID,
		event.Timestamp.Format(time.RFC3339),
		event.PodName,
		event.Namespace,
		event.CPUUsagePercent,
		event.CPULimitPercent,
		event.RAMUsagePercent,
		event.RAMLimitPercent,
		event.ThresholdPercent,
		event.MovingAveragePercent,
		nullString(event.RouteName),
		nullString(event.TraceID),
		nullString(event.CulpritFunction),
		nullString(event.CulpritFilePath),
		event.AlertSent,
		nullTime(event.CooldownEnd),
		event.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create spike event: %w", err)
	}

	return nil
}

// GetSpikeEvent retrieves a spike event by ID.
func (r *Repository) GetSpikeEvent(ctx context.Context, id string) (*SpikeEvent, error) {
	var event SpikeEvent
	var timestampStr, createdAtStr string
	var routeName, traceID, culpritFunction, culpritFilePath sql.NullString
	var cooldownEnd sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT
			id, timestamp, pod_name, namespace,
			cpu_usage_percent, cpu_limit_percent,
			ram_usage_percent, ram_limit_percent,
			threshold_percent, moving_average_percent,
			route_name, trace_id, culprit_function, culprit_file_path,
			alert_sent, cooldown_end, created_at
		FROM spike_events
		WHERE id = ?
	`, id).Scan(
		&event.ID,
		&timestampStr,
		&event.PodName,
		&event.Namespace,
		&event.CPUUsagePercent,
		&event.CPULimitPercent,
		&event.RAMUsagePercent,
		&event.RAMLimitPercent,
		&event.ThresholdPercent,
		&event.MovingAveragePercent,
		&routeName,
		&traceID,
		&culpritFunction,
		&culpritFilePath,
		&event.AlertSent,
		&cooldownEnd,
		&createdAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get spike event: %w", err)
	}

	// Parse timestamps
	event.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
	event.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

	// Set nullable fields
	event.RouteName = nullStringPtr(routeName)
	event.TraceID = nullStringPtr(traceID)
	event.CulpritFunction = nullStringPtr(culpritFunction)
	event.CulpritFilePath = nullStringPtr(culpritFilePath)
	event.CooldownEnd = nullTimePtr(cooldownEnd)

	return &event, nil
}

// ListSpikeEvents retrieves a list of spike events with filtering and pagination.
func (r *Repository) ListSpikeEvents(ctx context.Context, limit, offset int, namespace, podFilter string, startTime, endTime time.Time) ([]SpikeEvent, error) {
	// Build query with filters
	query := `
		SELECT
			id, timestamp, pod_name, namespace,
			cpu_usage_percent, cpu_limit_percent,
			ram_usage_percent, ram_limit_percent,
			threshold_percent, moving_average_percent,
			route_name, trace_id, culprit_function, culprit_file_path,
			alert_sent, cooldown_end, created_at
		FROM spike_events
		WHERE 1=1
	`
	args := []interface{}{}

	if namespace != "" {
		query += " AND namespace = ?"
		args = append(args, namespace)
	}

	if podFilter != "" {
		query += " AND pod_name LIKE ?"
		args = append(args, "%"+podFilter+"%")
	}

	if !startTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, startTime.Format(time.RFC3339))
	}

	if !endTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, endTime.Format(time.RFC3339))
	}

	// Add ordering and pagination
	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list spike events: %w", err)
	}
	defer rows.Close()

	var events []SpikeEvent
	for rows.Next() {
		var event SpikeEvent
		var timestampStr, createdAtStr string
		var routeName, traceID, culpritFunction, culpritFilePath sql.NullString
		var cooldownEnd sql.NullString

		err := rows.Scan(
			&event.ID,
			&timestampStr,
			&event.PodName,
			&event.Namespace,
			&event.CPUUsagePercent,
			&event.CPULimitPercent,
			&event.RAMUsagePercent,
			&event.RAMLimitPercent,
			&event.ThresholdPercent,
			&event.MovingAveragePercent,
			&routeName,
			&traceID,
			&culpritFunction,
			&culpritFilePath,
			&event.AlertSent,
			&cooldownEnd,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan spike event: %w", err)
		}

		// Parse timestamps
		event.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
		event.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

		// Set nullable fields
		event.RouteName = nullStringPtr(routeName)
		event.TraceID = nullStringPtr(traceID)
		event.CulpritFunction = nullStringPtr(culpritFunction)
		event.CulpritFilePath = nullStringPtr(culpritFilePath)
		event.CooldownEnd = nullTimePtr(cooldownEnd)

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating spike events: %w", err)
	}

	if events == nil {
		events = []SpikeEvent{}
	}

	return events, nil
}

// UpdateSpikeEvent updates an existing spike event.
func (r *Repository) UpdateSpikeEvent(ctx context.Context, event *SpikeEvent) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE spike_events SET
			timestamp = ?,
			pod_name = ?,
			namespace = ?,
			cpu_usage_percent = ?,
			cpu_limit_percent = ?,
			ram_usage_percent = ?,
			ram_limit_percent = ?,
			threshold_percent = ?,
			moving_average_percent = ?,
			route_name = ?,
			trace_id = ?,
			culprit_function = ?,
			culprit_file_path = ?,
			alert_sent = ?,
			cooldown_end = ?
		WHERE id = ?
	`,
		event.Timestamp.Format(time.RFC3339),
		event.PodName,
		event.Namespace,
		event.CPUUsagePercent,
		event.CPULimitPercent,
		event.RAMUsagePercent,
		event.RAMLimitPercent,
		event.ThresholdPercent,
		event.MovingAveragePercent,
		nullString(event.RouteName),
		nullString(event.TraceID),
		nullString(event.CulpritFunction),
		nullString(event.CulpritFilePath),
		event.AlertSent,
		nullTime(event.CooldownEnd),
		event.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update spike event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("spike event not found: %s", event.ID)
	}

	return nil
}

// DeleteSpikeEvent deletes a spike event by ID.
func (r *Repository) DeleteSpikeEvent(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM spike_events WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete spike event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("spike event not found: %s", id)
	}

	return nil
}

// PurgeOldSpikes removes spike events older than the specified number of days.
func (r *Repository) PurgeOldSpikes(ctx context.Context, days int) (int64, error) {
	if days <= 0 {
		days = 7 // Default 7 days
	}

	result, err := r.db.ExecContext(ctx, `
		DELETE FROM spike_events
		WHERE created_at < datetime('now', '-' || ? || ' days')
	`, days)
	if err != nil {
		return 0, fmt.Errorf("failed to purge old spikes: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// SaveTimelineMetrics saves timeline metrics to the cache.
func (r *Repository) SaveTimelineMetrics(ctx context.Context, metrics []TimelineMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	// Use a transaction for batch insert
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metrics_cache (
			pod_name, namespace, cpu_percent, ram_percent, timestamp, created_at
		) VALUES (?, ?, ?, ?, ?, datetime('now'))
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, m := range metrics {
		_, err := stmt.ExecContext(ctx, m.PodName, m.Namespace, m.CPUPercent, m.RAMPercent, m.Timestamp.Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("failed to save timeline metric: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetTimelineMetrics retrieves timeline metrics from cache within a time range.
func (r *Repository) GetTimelineMetrics(ctx context.Context, startTime, endTime time.Time, namespace, podFilter string) ([]TimelineMetric, error) {
	// Build query with filters
	query := `
		SELECT
			id, timestamp, pod_name, namespace, cpu_percent, ram_percent, created_at
		FROM metrics_cache
		WHERE 1=1
	`
	args := []interface{}{}

	if namespace != "" {
		query += " AND namespace = ?"
		args = append(args, namespace)
	}

	if podFilter != "" {
		query += " AND pod_name LIKE ?"
		args = append(args, "%"+podFilter+"%")
	}

	if !startTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, startTime.Format(time.RFC3339))
	}

	if !endTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, endTime.Format(time.RFC3339))
	}

	// Add ordering
	query += " ORDER BY timestamp ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get timeline metrics: %w", err)
	}
	defer rows.Close()

	var metrics []TimelineMetric
	for rows.Next() {
		var m TimelineMetric
		var timestampStr, createdAtStr string

		err := rows.Scan(
			&m.ID,
			&timestampStr,
			&m.PodName,
			&m.Namespace,
			&m.CPUPercent,
			&m.RAMPercent,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timeline metric: %w", err)
		}

		// Parse timestamps
		m.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

		metrics = append(metrics, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating timeline metrics: %w", err)
	}

	if metrics == nil {
		metrics = []TimelineMetric{}
	}

	return metrics, nil
}

// PruneOldMetrics removes metrics older than the specified hours.
func (r *Repository) PruneOldMetrics(ctx context.Context, retentionHours int) (int64, error) {
	if retentionHours <= 0 {
		retentionHours = 24 // Default 24 hours
	}

	result, err := r.db.ExecContext(ctx, `
		DELETE FROM metrics_cache
		WHERE created_at < datetime('now', '-' || ? || ' hours')
	`, retentionHours)
	if err != nil {
		return 0, fmt.Errorf("failed to prune old metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// Helper functions

// nullString converts a string pointer to sql.NullString.
func nullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// nullTime converts a time pointer to sql.NullString.
func nullTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.Format(time.RFC3339), Valid: true}
}

// nullStringPtr converts sql.NullString to string pointer.
func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// nullTimePtr converts sql.NullString to time pointer.
func nullTimePtr(ns sql.NullString) *time.Time {
	if !ns.Valid {
		return nil
	}
	t, err := time.Parse(time.RFC3339, ns.String)
	if err != nil {
		return nil
	}
	return &t
}

// Ensure strings package is used
var _ = strings.TrimSpace
