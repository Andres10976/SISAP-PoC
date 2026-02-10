package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
)

type KeywordRepository struct {
	pool *pgxpool.Pool
}

func NewKeywordRepository(pool *pgxpool.Pool) *KeywordRepository {
	return &KeywordRepository{pool: pool}
}

func (r *KeywordRepository) List(ctx context.Context) ([]model.Keyword, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, value, created_at FROM keywords ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keywords []model.Keyword
	for rows.Next() {
		var kw model.Keyword
		if err := rows.Scan(&kw.ID, &kw.Value, &kw.CreatedAt); err != nil {
			return nil, err
		}
		keywords = append(keywords, kw)
	}
	return keywords, rows.Err()
}

func (r *KeywordRepository) Create(ctx context.Context, value string) (*model.Keyword, error) {
	var kw model.Keyword
	err := r.pool.QueryRow(ctx,
		`INSERT INTO keywords (value) VALUES ($1)
		 RETURNING id, value, created_at`, value,
	).Scan(&kw.ID, &kw.Value, &kw.CreatedAt)
	return &kw, err
}

func (r *KeywordRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM keywords WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
