package guild

import (
	"context"
	"testing"

	domainguild "mcgame-server/internal/domain/guild"

	"go.uber.org/zap"
)

type warehouseCommandFixture struct {
	permissionRepository
	moveCalls int
	dropCalls int
}

func (r *warehouseCommandFixture) MoveGuildWarehouseAtomic(_ context.Context, cmd WarehouseMoveCommand) (*WarehouseMoveResult, error) {
	r.moveCalls++
	return &WarehouseMoveResult{SourceID: cmd.SourceID, Mode: "move"}, nil
}

func (r *warehouseCommandFixture) DropGuildWarehouseAtomic(_ context.Context, _ WarehouseDropCommand) (*WarehouseDropResult, error) {
	r.dropCalls++
	return &WarehouseDropResult{TemplateID: 2026, StackCount: 5}, nil
}

func TestWarehouseCommandsAuthorizeAndValidateBoundsBeforeMutation(t *testing.T) {
	settings := DefaultRankSettings()
	settings[domainguild.RankMember-1].CanSlot = true
	repo := &warehouseCommandFixture{permissionRepository: permissionRepository{
		guild:    &domainguild.Guild{ID: 8, LeaderID: 1},
		members:  []*domainguild.GuildMember{{GuildID: 8, CharacterID: 2, Rank: domainguild.RankMember}},
		settings: settings,
	}}
	service := NewService(repo, nil, zap.NewNop())
	command := WarehouseMoveCommand{GuildID: 8, ActorID: 2, SourceID: 9, SourceIndex: 0, TargetIndex: 29, TemplateID: 2026, MaxStack: 100, PageCount: 1}
	if _, err := service.MoveGuildWarehouseAtomic(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	command.TargetIndex = 30
	if _, err := service.MoveGuildWarehouseAtomic(context.Background(), command); err == nil {
		t.Fatal("target outside opened pages must fail")
	}
	command.TargetIndex = 0
	if _, err := service.MoveGuildWarehouseAtomic(context.Background(), command); err == nil {
		t.Fatal("same-slot request must fail")
	}
	repo.settings[domainguild.RankMember-1].CanSlot = false
	if _, err := service.DropGuildWarehouseAtomic(context.Background(), WarehouseDropCommand{GuildID: 8, ActorID: 2, SlotID: 9}); err == nil {
		t.Fatal("unauthorized drop must fail")
	}
	if repo.moveCalls != 1 || repo.dropCalls != 0 {
		t.Fatalf("repository calls: move=%d drop=%d", repo.moveCalls, repo.dropCalls)
	}
}
