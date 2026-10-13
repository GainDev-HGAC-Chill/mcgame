package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"mcgame-server/internal/domain/character"
)

func (s *Store) NewCharacterDefaults(ctx context.Context) (character.NewCharacterDefaults, error) {
	return loadNewCharacterDefaults(ctx, s.pool)
}

type defaultsQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadNewCharacterDefaults(ctx context.Context, queryer defaultsQueryer) (character.NewCharacterDefaults, error) {
	var raw []byte
	err := queryer.QueryRow(ctx, "select value from public.server_settings where key=$1", character.NewCharacterDefaultsKey).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return character.DefaultNewCharacterDefaults(), nil
	}
	if err != nil {
		return character.NewCharacterDefaults{}, fmt.Errorf("read creation defaults: %w", err)
	}
	return character.DecodeNewCharacterDefaults(raw)
}
