// Open-sourced by BaoLT

// Repository defines persistence operations for farm plots.
package farm

import (
	"context"
	"time"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
)

type Repository interface {
	FindByCharacter(ctx context.Context, characterID int64) ([]*Plot, error)
	FindByCharacterAndPlot(ctx context.Context, characterID int64, plotNPCID int) (*Plot, error)
	FindByPlot(ctx context.Context, plotNPCID int) (*Plot, error)
	FindByPlots(ctx context.Context, plotNPCIDs []int) ([]*Plot, error)
	Upsert(ctx context.Context, plot *Plot) error
	Delete(ctx context.Context, characterID int64, plotNPCID int) error
}

type PlantCommand struct {
	CharacterID           int64
	PlotNPCID             int
	CropNPCID             int
	HarvestItemTemplateID int
	HarvestCount          int
	SeedTemplateID        int
	VigorCost             int
	PlantedAt             time.Time
	ReadyAt               time.Time
}

type PlantCommandResult struct {
	Character *domainchar.Character
	Plot      *Plot
	Seed      *domainitem.Item
}

type HarvestCommand struct {
	OwnerCharacterID     int64
	HarvesterCharacterID int64
	PlotNPCID            int
}

type HarvestCommandResult struct {
	Plot             *Plot
	Reward           *domainitem.Item
	RewardTemplateID int
	RewardCount      int
	Shared           bool
	PlotDeleted      bool
}

type RipenAndHarvestCommand struct {
	CharacterID int64
	PlotNPCID   int
	GoldCost    int64
}

type RipenAndHarvestCommandResult struct {
	Character        *domainchar.Character
	Plot             *Plot
	Reward           *domainitem.Item
	RewardTemplateID int
	RewardCount      int
	GoldCharged      int64
}

type UprootCommand struct {
	CharacterID int64
	PlotNPCID   int
	GoldCost    int64
}

type UprootCommandResult struct {
	Character *domainchar.Character
	Plot      *Plot
}

type CommandRepository interface {
	PlantAtomic(ctx context.Context, command PlantCommand) (*PlantCommandResult, error)
	HarvestAtomic(ctx context.Context, command HarvestCommand) (*HarvestCommandResult, error)
	RipenAndHarvestAtomic(ctx context.Context, command RipenAndHarvestCommand) (*RipenAndHarvestCommandResult, error)
	UprootAtomic(ctx context.Context, command UprootCommand) (*UprootCommandResult, error)
}
