// Open-sourced by BaoLT

package guild

import (
	"context"
	"fmt"

	domainguild "mcgame-server/internal/domain/guild"

	"go.uber.org/zap"
)

func (s *Service) cleanupGuildDeparture(ctx context.Context, characterID int64) {
	if s.questSvc == nil {
		return
	}
	if _, err := s.questSvc.AbandonGuildQuests(ctx, characterID); err != nil {
		s.logger.Error("failed post-commit guild quest cleanup", zap.Int64("character_id", characterID), zap.Error(err))
	}
}

func (s *Service) enrichGuild(ctx context.Context, guildItem *domainguild.Guild) (*domainguild.Guild, error) {
	if guildItem == nil {
		return nil, nil
	}

	if err := s.loadGuildSettings(ctx, guildItem); err != nil {
		return nil, err
	}

	applications, err := s.repo.ListApplications(ctx, guildItem.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list guild applications: %w", err)
	}
	guildItem.Applications = applications

	skillData, err := s.GetGuildSkillData(ctx, guildItem.ID)
	if err != nil {
		return nil, err
	}
	guildItem.SkillDevData = skillData

	skillLevels, err := s.repo.GetGuildSkillLevels(ctx, guildItem.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to load guild skill levels: %w", err)
	}
	guildItem.SkillLevels = skillLevels

	warehouse, err := s.LoadGuildWarehouse(ctx, guildItem.ID)
	if err != nil {
		return nil, err
	}
	guildItem.Warehouse = warehouse

	return guildItem, nil
}

func (s *Service) loadGuildSettings(ctx context.Context, guildItem *domainguild.Guild) error {
	rankSettings, err := s.repo.LoadRankSettings(ctx, guildItem.ID)
	if err != nil {
		return fmt.Errorf("failed to load guild rank settings: %w", err)
	}
	if len(rankSettings) == 0 {
		rankSettings = DefaultRankSettings()
	}
	guildItem.RankSettings = rankSettings

	pageCount, err := s.repo.LoadBankPageCount(ctx, guildItem.ID)
	if err != nil {
		return fmt.Errorf("failed to load guild bank page count: %w", err)
	}
	if pageCount < 1 {
		pageCount = 1
	}
	guildItem.BankPageCount = pageCount

	return nil
}

func (s *Service) loadApplicationAndAuthority(ctx context.Context, applicationID, actorID int64) (*domainguild.GuildApplication, *domainguild.Guild, *domainguild.GuildMember, error) {
	application, err := s.repo.GetApplicationByID(ctx, applicationID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get guild application: %w", err)
	}
	if application == nil {
		return nil, nil, nil, ErrGuildNotFound
	}

	guildItem, err := s.repo.GetByID(ctx, application.GuildID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get guild: %w", err)
	}
	if guildItem == nil {
		return nil, nil, nil, ErrGuildNotFound
	}

	members, err := s.repo.GetMembers(ctx, guildItem.ID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get guild members: %w", err)
	}
	var approverMember *domainguild.GuildMember
	for _, member := range members {
		if member != nil && member.CharacterID == actorID {
			approverMember = member
			break
		}
	}

	return application, guildItem, approverMember, nil
}

func DefaultRankSettings() []domainguild.RankSetting {
	return []domainguild.RankSetting{
		{Rank: domainguild.RankLeader, Name: "Bang chủ", CanAdd: true, CanQuest: true, CanSlot: true, CanInfo: true, CanDel: true, CanDuty: true},
		{Rank: domainguild.RankViceLeader, Name: "Phó bang", CanAdd: true, CanQuest: true, CanSlot: true, CanInfo: true, CanDel: true, CanDuty: true},
		{Rank: domainguild.RankOfficer, Name: "Trưởng lão", CanAdd: true, CanQuest: true, CanSlot: true, CanInfo: true},
		{Rank: domainguild.RankElite, Name: "Tinh anh", CanAdd: true, CanQuest: true, CanInfo: true},
		{Rank: domainguild.RankVeteran, Name: "Thành viên cũ", CanQuest: true},
		{Rank: domainguild.RankMember, Name: "Thành viên"},
	}
}
