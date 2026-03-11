package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rachelJG/event-notification-service/internal/domain/entities"
	apperror "github.com/rachelJG/event-notification-service/internal/pkg/apperror"
)

// APIKeyRepository implements ports.APIKeyRepository using pgxpool.
type APIKeyRepository struct {
	Pool         *pgxpool.Pool
	QueryTimeout time.Duration
}

func (r APIKeyRepository) Create(ctx context.Context, key entities.APIKey) error {
	if r.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.QueryTimeout)
		defer cancel()
	}

	_, err := r.Pool.Exec(ctx, `
		INSERT INTO api_keys (id, key_hash, name, scopes, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, key.ID, key.KeyHash, key.Name, key.Scopes, key.IsActive, key.CreatedAt)
	if err != nil {
		return mapDBError(err)
	}
	return nil
}

func (r APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (entities.APIKey, error) {
	if r.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.QueryTimeout)
		defer cancel()
	}

	var k entities.APIKey
	err := r.Pool.QueryRow(ctx, `
		SELECT id, key_hash, name, scopes, is_active, created_at, last_used_at
		FROM api_keys
		WHERE key_hash = $1
	`, keyHash).Scan(&k.ID, &k.KeyHash, &k.Name, &k.Scopes, &k.IsActive, &k.CreatedAt, &k.LastUsedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.APIKey{}, apperror.NotFound("api key not found", err)
		}
		return entities.APIKey{}, mapDBError(err)
	}
	return k, nil
}

func (r APIKeyRepository) List(ctx context.Context) ([]entities.APIKey, error) {
	if r.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.QueryTimeout)
		defer cancel()
	}

	rows, err := r.Pool.Query(ctx, `
		SELECT id, name, scopes, is_active, created_at, last_used_at
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()

	var keys []entities.APIKey
	for rows.Next() {
		var k entities.APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.Scopes, &k.IsActive, &k.CreatedAt, &k.LastUsedAt); err != nil {
			return nil, mapDBError(err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r APIKeyRepository) Revoke(ctx context.Context, id string) error {
	if r.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.QueryTimeout)
		defer cancel()
	}

	tag, err := r.Pool.Exec(ctx, `UPDATE api_keys SET is_active = FALSE WHERE id = $1`, id)
	if err != nil {
		return mapDBError(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.NotFound("api key not found", nil)
	}
	return nil
}

func (r APIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	if r.QueryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.QueryTimeout)
		defer cancel()
	}

	_, err := r.Pool.Exec(ctx, `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return mapDBError(err)
	}
	return nil
}
