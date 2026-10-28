// Open-sourced by BaoLT

package guild

import (
	"context"
	"fmt"

	domainguild "mcgame-server/internal/domain/guild"
)

func (s *Service) GetGuildSkillData(ctx context.Context, guildID int64) (map[string]interface{}, error) {
	skillIDs, err := s.repo.GetGuildSkillIDs(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to list guild skills: %w", err)
	}

	result := map[string]interface{}{}
	for _, skillID := range skillIDs {
		result[fmt.Sprintf("%d", skillID)] = 1
	}

	return result, nil
}

func (s *Service) LoadGuildWarehouse(ctx context.Context, guildID int64) (map[string]interface{}, error) {
	slots, err := s.repo.ListWarehouseSlots(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to list guild warehouse slots: %w", err)
	}

	result := map[string]interface{}{}
	for _, slot := range slots {
		if slot == nil {
			continue
		}
		result[fmt.Sprintf("%d", slot.ID)] = slot.ToDTO()
	}

	return result, nil
}

func (s *Service) AddWarehouseMaterial(ctx context.Context, guildID int64, templateID int, count int) (*domainguild.GuildWarehouseSlot, error) {
	return nil, ErrUnsupportedAction
}

func (s *Service) UpdateWarehouseSlot(ctx context.Context, guildID, slotID int64, slotIndex int, stackCount int) error {
	return ErrUnsupportedAction
}

func (s *Service) RemoveWarehouseSlot(ctx context.Context, guildID, slotID int64) error {
	return ErrUnsupportedAction
}

func (s *Service) AddContribution(ctx context.Context, guildID, characterID int64, guildFunds, normalContrib, donateContrib int64) (*domainguild.Guild, error) {
	return nil, ErrUnsupportedAction
}

func (s *Service) DevelopSkill(ctx context.Context, guildID, actorID int64, skillID int) error {
	g, err := s.repo.GetByID(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild: %w", err)
	}
	if g == nil {
		return ErrGuildNotFound
	}

	if err := s.AuthorizeLeader(ctx, guildID, actorID); err != nil {
		return err
	}
	learned, err := s.repo.GetGuildSkillIDs(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to list guild skills: %w", err)
	}
	development, err := ResolveGuildSkillDevelopment(s.skillCatalog, skillID, learned)
	if err != nil {
		return err
	}
	commands, ok := s.repo.(GuildSkillCommandRepository)
	if !ok {
		return ErrUnsupportedAction
	}
	return commands.DevelopGuildSkillAtomic(ctx, guildID, actorID, development)
}
