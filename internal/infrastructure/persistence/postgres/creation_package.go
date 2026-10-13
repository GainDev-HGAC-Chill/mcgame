package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"mcgame-server/internal/domain/character"
)

type creationQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type creationTransaction interface {
	creationQueryer
	Commit(context.Context) error
	Rollback(context.Context) error
}

func (r *CharacterRepository) CreateWithPackage(ctx context.Context, char *character.Character, bundle *character.CreationPackage) error {
	return persistCreationPackage(ctx, char, bundle, func(ctx context.Context) (creationTransaction, error) {
		return r.db.pool.Begin(ctx)
	})
}

func persistCreationPackage(ctx context.Context, char *character.Character, bundle *character.CreationPackage, begin func(context.Context) (creationTransaction, error)) error {
	if char == nil || char.ID != 0 || bundle == nil {
		return fmt.Errorf("creation package can only be persisted for a new character")
	}
	if err := bundle.Defaults.Validate(); err != nil {
		return err
	}
	hash, err := bundle.Defaults.SnapshotHash()
	if err != nil || hash != bundle.Hash {
		return fmt.Errorf("creation package snapshot hash mismatch")
	}
	tx, err := begin(ctx)
	if err != nil {
		return fmt.Errorf("begin creation package: %w", err)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	newChar := *char
	if err := createCharacter(ctx, tx, &newChar); err != nil {
		return err
	}
	for _, it := range bundle.Items {
		if it == nil || it.ID != 0 || it.CharacterID != 0 {
			return fmt.Errorf("invalid new creation item")
		}
		newItem := *it
		newItem.CharacterID = newChar.ID
		if err := createItem(ctx, tx, &newItem); err != nil {
			return err
		}
	}
	for _, p := range bundle.Pets {
		if p == nil || p.ID != 0 || p.CharacterID != 0 {
			return fmt.Errorf("invalid new creation pet")
		}
		newPet := *p
		newPet.CharacterID = newChar.ID
		if err := createPet(ctx, tx, &newPet); err != nil {
			return err
		}
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		return fmt.Errorf("marshal creation snapshot: %w", err)
	}
	var stored bool
	if err := tx.QueryRow(ctx, "select player.upsert_character_feature_state($1, $2, $3::jsonb)", newChar.ID, character.CreationPackageFeatureKey, raw).Scan(&stored); err != nil {
		return fmt.Errorf("persist creation snapshot: %w", err)
	}
	if !stored {
		return fmt.Errorf("creation snapshot was not stored")
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit creation package: %w", err)
	}
	char.ID = newChar.ID
	return nil
}
