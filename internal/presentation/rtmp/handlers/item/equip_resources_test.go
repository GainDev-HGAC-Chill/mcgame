package item

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appitem "mcgame-server/internal/application/item"
	"mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
)

type resourcePatchRepository struct {
	domainitem.Repository
	current *character.Character
}

func (r *resourcePatchRepository) FindEquipped(context.Context, int64) ([]*domainitem.Item, error) {
	return nil, nil
}

func (r *resourcePatchRepository) UpdateCachedCharacterConditional(_ int64, patch func(*character.Character, []*domainitem.Item) (bool, error)) (*character.Character, bool, error) {
	updated := *r.current
	changed, err := patch(&updated, nil)
	if err == nil && changed {
		*r.current = updated
	}
	result := *r.current
	return &result, changed, err
}

func TestEquipResourceSyncCannotRestoreStaleLevel(t *testing.T) {
	current := character.NewCharacter(uuid.New(), "target", 1, 0)
	current.RaiseLevelTo(20)
	stale := *current
	if _, err := current.SetTargetLevel(5); err != nil {
		t.Fatal(err)
	}
	current.Gold = 8888
	current.CurrentHP = 99999
	current.CurrentMP = 99999
	repo := &resourcePatchRepository{current: current}
	service := appitem.NewService(repo, zap.NewNop())
	handler := &Handler{itemService: service, logger: zap.NewNop()}
	handler.syncHPMPAfterEquipChange(context.Background(), nil, &stale)
	if current.Level != 5 || stale.Level != 5 || current.Gold != 8888 || stale.Gold != 8888 || current.Strength != 14 || current.CurrentHP > current.MaxHP || current.CurrentMP > current.MaxMP {
		t.Fatal("equip HP/MP sync resurrected stale character state")
	}
}
