package postgres

import (
	"context"
	"encoding/json"
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
	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

	metadataJSON, err := json.Marshal(key.Metadata)
	if err != nil {
		return apperror.InvalidArgument("failed to marshal metadata", err)
	}

	_, err = r.Pool.Exec(ctx, `
		INSERT INTO api_keys (id, key_hash, name, scopes, metadata, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, key.ID, key.KeyHash, key.Name, key.Scopes, metadataJSON, key.IsActive, key.CreatedAt)
	if err != nil {
		return mapDBError(err)
	}
	return nil
}

func (r APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (entities.APIKey, error) {
	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

	var k entities.APIKey
	var metadataJSON []byte
	err := r.Pool.QueryRow(ctx, `
		SELECT id, key_hash, name, scopes, metadata, is_active, created_at, last_used_at
		FROM api_keys
		WHERE key_hash = $1
	`, keyHash).Scan(&k.ID, &k.KeyHash, &k.Name, &k.Scopes, &metadataJSON, &k.IsActive, &k.CreatedAt, &k.LastUsedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.APIKey{}, apperror.NotFound("api key not found", err)
		}
		return entities.APIKey{}, mapDBError(err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &k.Metadata); err != nil {
			return entities.APIKey{}, apperror.Internal("failed to unmarshal metadata", err)
		}
	}

	return k, nil
}

func (r APIKeyRepository) List(ctx context.Context) ([]entities.APIKey, error) {
	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

	rows, err := r.Pool.Query(ctx, `
		SELECT id, name, scopes, metadata, is_active, created_at, last_used_at
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
		var metadataJSON []byte
		if err := rows.Scan(&k.ID, &k.Name, &k.Scopes, &metadataJSON, &k.IsActive, &k.CreatedAt, &k.LastUsedAt); err != nil {
			return nil, mapDBError(err)
		}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &k.Metadata); err != nil {
				return nil, apperror.Internal("failed to unmarshal metadata", err)
			}
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r APIKeyRepository) Revoke(ctx context.Context, id string) error {
	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

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
	ctx, cancel := withQueryTimeout(ctx, r.QueryTimeout)
	defer cancel()

	_, err := r.Pool.Exec(ctx, `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return mapDBError(err)
	}
	return nil
}
