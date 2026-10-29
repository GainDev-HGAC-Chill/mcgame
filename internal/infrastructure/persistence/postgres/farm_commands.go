// PostgreSQL farm commands return committed snapshots from atomic database functions.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	appfarm "mcgame-server/internal/application/farm"
	domainchar "mcgame-server/internal/domain/character"
	domainfarm "mcgame-server/internal/domain/farm"
	domainitem "mcgame-server/internal/domain/item"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func (r *FarmRepository) PlantAtomic(ctx context.Context, command domainfarm.PlantCommand) (*domainfarm.PlantCommandResult, error) {
	var characterJSON, plotJSON, seedJSON []byte
	err := r.db.pool.QueryRow(ctx,
		"select * from player.plant_farm_crop_atomic($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		command.CharacterID, command.PlotNPCID, command.CropNPCID, command.HarvestItemTemplateID,
		command.HarvestCount, command.SeedTemplateID, command.VigorCost, command.PlantedAt, command.ReadyAt,
	).Scan(&characterJSON, &plotJSON, &seedJSON)
	if err != nil {
		return nil, mapFarmCommandError(err)
	}
	result := &domainfarm.PlantCommandResult{}
	if err := decodeFarmCharacter(characterJSON, &result.Character); err != nil {
		return nil, err
	}
	if err := decodeFarmPlot(plotJSON, &result.Plot); err != nil {
		return nil, err
	}
	if err := decodeFarmItem(seedJSON, &result.Seed); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *FarmRepository) HarvestAtomic(ctx context.Context, command domainfarm.HarvestCommand) (*domainfarm.HarvestCommandResult, error) {
	var plotJSON, rewardJSON []byte
	result := &domainfarm.HarvestCommandResult{}
	err := r.db.pool.QueryRow(ctx,
		"select * from player.harvest_farm_crop_atomic($1, $2, $3)",
		command.OwnerCharacterID, command.HarvesterCharacterID, command.PlotNPCID,
	).Scan(&plotJSON, &rewardJSON, &result.RewardTemplateID, &result.RewardCount, &result.Shared, &result.PlotDeleted)
	if err != nil {
		return nil, mapFarmCommandError(err)
	}
	if err := decodeFarmPlot(plotJSON, &result.Plot); err != nil {
		return nil, err
	}
	if err := decodeFarmItem(rewardJSON, &result.Reward); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *FarmRepository) RipenAndHarvestAtomic(ctx context.Context, command domainfarm.RipenAndHarvestCommand) (*domainfarm.RipenAndHarvestCommandResult, error) {
	var characterJSON, plotJSON, rewardJSON []byte
	result := &domainfarm.RipenAndHarvestCommandResult{}
	err := r.db.pool.QueryRow(ctx,
		"select * from player.ripen_and_harvest_farm_crop_atomic($1, $2, $3)",
		command.CharacterID, command.PlotNPCID, command.GoldCost,
	).Scan(&characterJSON, &plotJSON, &rewardJSON, &result.RewardTemplateID, &result.RewardCount, &result.GoldCharged)
	if err != nil {
		return nil, mapFarmCommandError(err)
	}
	if err := decodeFarmCharacter(characterJSON, &result.Character); err != nil {
		return nil, err
	}
	if err := decodeFarmPlot(plotJSON, &result.Plot); err != nil {
		return nil, err
	}
	if err := decodeFarmItem(rewardJSON, &result.Reward); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *FarmRepository) UprootAtomic(ctx context.Context, command domainfarm.UprootCommand) (*domainfarm.UprootCommandResult, error) {
	var characterJSON, plotJSON []byte
	err := r.db.pool.QueryRow(ctx,
		"select * from player.uproot_farm_crop_atomic($1, $2, $3)",
		command.CharacterID, command.PlotNPCID, command.GoldCost,
	).Scan(&characterJSON, &plotJSON)
	if err != nil {
		return nil, mapFarmCommandError(err)
	}
	result := &domainfarm.UprootCommandResult{}
	if err := decodeFarmCharacter(characterJSON, &result.Character); err != nil {
		return nil, err
	}
	if err := decodeFarmPlot(plotJSON, &result.Plot); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeFarmCharacter(raw []byte, target **domainchar.Character) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	*target = &domainchar.Character{}
	if err := json.Unmarshal(raw, *target); err != nil {
		return fmt.Errorf("decode committed farm character: %w", err)
	}
	return nil
}

func decodeFarmPlot(raw []byte, target **domainfarm.Plot) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	*target = &domainfarm.Plot{}
	if err := json.Unmarshal(raw, *target); err != nil {
		return fmt.Errorf("decode committed farm plot: %w", err)
	}
	return nil
}

func decodeFarmItem(raw []byte, target **domainitem.Item) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	*target = &domainitem.Item{}
	if err := json.Unmarshal(raw, *target); err != nil {
		return fmt.Errorf("decode committed farm item: %w", err)
	}
	return nil
}

func mapFarmCommandError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Message {
		case "FARM_INVALID_PLOT":
			return appfarm.ErrInvalidPlotNPCID
		case "FARM_INVALID_CROP":
			return appfarm.ErrInvalidCropNPCID
		case "FARM_PLOT_OCCUPIED":
			return appfarm.ErrPlotOccupied
		case "FARM_ANOTHER_PLOT_ACTIVE":
			return appfarm.ErrAnotherPlotActive
		case "FARM_PLOT_NOT_FOUND":
			return appfarm.ErrPlotNotFound
		case "FARM_CROP_NOT_READY":
			return appfarm.ErrCropNotReady
		case "FARM_SEED_NOT_FOUND":
			return appfarm.ErrSeedNotFound
		case "FARM_PLOT_ALREADY_SHARED":
			return appfarm.ErrPlotAlreadyShared
		case "FARM_PLOT_SHARED_OUT":
			return appfarm.ErrPlotSharedOut
		case "FARM_VIGOR_NOT_ENOUGH", "INSUFFICIENT_FUNDS":
			return pkgerrors.ErrInsufficientFunds
		case "INVENTORY_FULL":
			return pkgerrors.ErrInventoryFull
		case "CHARACTER_NOT_FOUND":
			return pkgerrors.ErrCharacterNotFound
		}
	}
	return fmt.Errorf("execute farm command atomically: %w", err)
}

var _ domainfarm.CommandRepository = (*FarmRepository)(nil)
