// PostgreSQL gathering commands return committed character, skill, reward, and tool snapshots.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	appgathering "mcgame-server/internal/application/gathering"
	domainchar "mcgame-server/internal/domain/character"
	domaingathering "mcgame-server/internal/domain/gathering"
	domainitem "mcgame-server/internal/domain/item"
	domainskill "mcgame-server/internal/domain/skill"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/jackc/pgx/v5/pgconn"
)

type GatheringRepository struct {
	db *Database
}

func NewGatheringRepository(db *Database) *GatheringRepository {
	return &GatheringRepository{db: db}
}

func (r *GatheringRepository) CompleteGathering(ctx context.Context, command domaingathering.CompletionCommand) (*domaingathering.CompletionResult, error) {
	var characterJSON []byte
	var skillJSON []byte
	var rewardJSON []byte
	var toolJSON []byte
	result := &domaingathering.CompletionResult{}
	err := r.db.pool.QueryRow(ctx,
		"select * from player.complete_life_gathering_atomic($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
		command.AttemptID, command.CharacterID, int(command.Kind), command.NPCID, command.ResourceLevel,
		command.RewardTemplateID, command.ToolItemID, command.Succeeded, command.VigorCost, command.MasteryGain,
	).Scan(&characterJSON, &skillJSON, &rewardJSON, &toolJSON, &result.Succeeded)
	if err != nil {
		return nil, mapGatheringCommandError(err)
	}
	result.Character = &domainchar.Character{}
	if err := json.Unmarshal(characterJSON, result.Character); err != nil {
		return nil, fmt.Errorf("decode committed gathering character: %w", err)
	}
	if len(skillJSON) > 0 && string(skillJSON) != "null" {
		result.Skill = &domainskill.CharacterSkill{}
		if err := json.Unmarshal(skillJSON, result.Skill); err != nil {
			return nil, fmt.Errorf("decode committed gathering skill: %w", err)
		}
	}
	if len(rewardJSON) > 0 && string(rewardJSON) != "null" {
		result.Reward = &domainitem.Item{}
		if err := json.Unmarshal(rewardJSON, result.Reward); err != nil {
			return nil, fmt.Errorf("decode committed gathering reward: %w", err)
		}
	}
	if len(toolJSON) > 0 && string(toolJSON) != "null" {
		result.Tool = &domainitem.Item{}
		if err := json.Unmarshal(toolJSON, result.Tool); err != nil {
			return nil, fmt.Errorf("decode committed gathering tool: %w", err)
		}
	}
	if result.Character.ID != command.CharacterID || result.Reward != nil && result.Reward.CharacterID != command.CharacterID || result.Tool != nil && result.Tool.CharacterID != command.CharacterID {
		return nil, pkgerrors.ErrSystemError
	}
	return result, nil
}

func mapGatheringCommandError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Message {
		case "GATHERING_RESOURCE_UNAVAILABLE", "INVALID_GATHERING_COMMAND":
			return appgathering.ErrResourceUnavailable
		case "GATHERING_ATTEMPT_ALREADY_COMPLETED":
			return appgathering.ErrResourceUnavailable
		case "GATHERING_SKILL_LEVEL_NOT_ENOUGH", "GATHERING_SKILL_NOT_LEARNED":
			return appgathering.ErrSkillLevelNotEnough
		case "GATHERING_TOOL_NOT_EQUIPPED":
			return appgathering.ErrToolNotEquipped
		case "GATHERING_TOOL_INVALID":
			return appgathering.ErrToolInvalid
		case "GATHERING_TOOL_BROKEN":
			return appgathering.ErrToolBroken
		case "GATHERING_VIGOR_NOT_ENOUGH":
			return appgathering.ErrVigorNotEnough
		case "INVENTORY_FULL":
			return pkgerrors.ErrInventoryFull
		case "CHARACTER_NOT_FOUND":
			return pkgerrors.ErrCharacterNotFound
		}
	}
	return fmt.Errorf("complete life gathering atomically: %w", err)
}

var _ domaingathering.CommandRepository = (*GatheringRepository)(nil)
