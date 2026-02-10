package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
)

type MonitorRepository struct {
	pool *pgxpool.Pool
}

func NewMonitorRepository(pool *pgxpool.Pool) *MonitorRepository {
	return &MonitorRepository{pool: pool}
}

func (r *MonitorRepository) Get(ctx context.Context) (*model.MonitorState, error) {
	var s model.MonitorState
	err := r.pool.QueryRow(ctx,
		`SELECT last_processed_index, last_tree_size, last_run_at,
			total_processed, certs_in_last_cycle, matches_in_last_cycle,
			parse_errors_in_last_cycle, is_running, last_error, updated_at
		FROM monitor_state WHERE id = 1`,
	).Scan(
		&s.LastProcessedIndex, &s.LastTreeSize, &s.LastRunAt,
		&s.TotalProcessed, &s.CertsInLastCycle, &s.MatchesInLastCycle,
		&s.ParseErrorsInLastCycle, &s.IsRunning, &s.LastError, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *MonitorRepository) Update(ctx context.Context, state *model.MonitorState) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE monitor_state SET
			last_processed_index = $1,
			last_tree_size = $2,
			last_run_at = $3,
			total_processed = $4,
			certs_in_last_cycle = $5,
			matches_in_last_cycle = $6,
			parse_errors_in_last_cycle = $7,
			is_running = $8,
			last_error = $9,
			updated_at = $10
		WHERE id = 1`,
		state.LastProcessedIndex, state.LastTreeSize, now,
		state.TotalProcessed, state.CertsInLastCycle, state.MatchesInLastCycle,
		state.ParseErrorsInLastCycle, state.IsRunning, state.LastError, now,
	)
	return err
}

func (r *MonitorRepository) SetRunning(ctx context.Context, running bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE monitor_state SET is_running = $1, updated_at = $2 WHERE id = 1`,
		running, time.Now(),
	)
	return err
}

func (r *MonitorRepository) SetError(ctx context.Context, errMsg string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE monitor_state SET last_error = $1, updated_at = $2 WHERE id = 1`,
		errMsg, time.Now(),
	)
	return err
}
