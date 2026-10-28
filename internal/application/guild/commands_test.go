package guild

import (
	"context"
	"testing"

	domainguild "mcgame-server/internal/domain/guild"

	"go.uber.org/zap"
)

type commandRepository struct {
	Repository
	createCommand     CreateGuildCommand
	contributeCommand ContributeMoneyCommand
	materialCommand   ContributeMaterialCommand
}

func (r *commandRepository) CreateGuildAtomic(_ context.Context, command CreateGuildCommand) (*CreateGuildResult, error) {
	r.createCommand = command
	return &CreateGuildResult{GuildID: 5}, nil
}

func (r *commandRepository) ContributeMoneyAtomic(_ context.Context, command ContributeMoneyCommand) (*ContributeMoneyResult, error) {
	r.contributeCommand = command
	return &ContributeMoneyResult{GuildFunds: command.FundsGain}, nil
}

func (r *commandRepository) ContributeMaterialAtomic(_ context.Context, command ContributeMaterialCommand) (*ContributeMaterialResult, error) {
	r.materialCommand = command
	return &ContributeMaterialResult{ItemID: command.ItemID, ItemRemaining: 1}, nil
}

func TestCreateGuildAtomicSuppliesCanonicalCostAndRanks(t *testing.T) {
	repo := &commandRepository{}
	service := NewService(repo, nil, zap.NewNop())
	result, err := service.CreateGuildAtomic(context.Background(), 7, "Guild")
	if err != nil {
		t.Fatal(err)
	}
	if result.GuildID != 5 || repo.createCommand.Cost != GuildCreationCost || len(repo.createCommand.Ranks) != domainguild.RankMember {
		t.Fatalf("command = %#v, result = %#v", repo.createCommand, result)
	}
}

func TestContributeMoneyAtomicComputesServerOwnedGains(t *testing.T) {
	repo := &commandRepository{}
	service := NewService(repo, nil, zap.NewNop())
	if _, err := service.ContributeMoneyAtomic(context.Background(), 2, 7, 80, 2); err != nil {
		t.Fatal(err)
	}
	if repo.contributeCommand.FundsGain != 4080 || repo.contributeCommand.DonateGain != 510 {
		t.Fatalf("command = %#v", repo.contributeCommand)
	}
	if _, err := service.ContributeMoneyAtomic(context.Background(), 2, 7, -1, 0); err == nil {
		t.Fatal("negative contribution must fail")
	}
}

func TestContributeMaterialAtomicValidatesWarehouseCapacity(t *testing.T) {
	repo := &commandRepository{}
	service := NewService(repo, nil, zap.NewNop())
	valid := ContributeMaterialCommand{
		GuildID: 2, CharacterID: 7, ItemID: 9, ItemTemplateID: 16, ItemCount: 1,
		WarehouseTemplateID: 2026, WarehouseCount: 4, WarehouseMaxStack: 99, WarehousePageCount: 1,
	}
	if _, err := service.ContributeMaterialAtomic(context.Background(), valid); err != nil {
		t.Fatal(err)
	}
	if repo.materialCommand.WarehouseMaxStack != 99 || repo.materialCommand.WarehousePageCount != 1 {
		t.Fatalf("command = %#v", repo.materialCommand)
	}
	invalid := valid
	invalid.WarehouseCount = 100
	if _, err := service.ContributeMaterialAtomic(context.Background(), invalid); err != ErrInvalidContribution {
		t.Fatalf("overflow contribution error = %v", err)
	}
	invalid = valid
	invalid.WarehousePageCount = 5
	if _, err := service.ContributeMaterialAtomic(context.Background(), invalid); err != ErrInvalidContribution {
		t.Fatalf("page count error = %v", err)
	}
}

var _ CommandRepository = (*commandRepository)(nil)
