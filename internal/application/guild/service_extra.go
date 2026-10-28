// Open-sourced by BaoLT

// Guild service extra: public application and member operations.
// Guild management (name/rank/capacity/leadership) lives in service_guild_management.go.
// Skill/warehouse/contribution operations live in service_guild_skills.go.
// Private helpers and DefaultRankSettings live in service_guild_helpers.go.
package guild

import (
	"context"
	"fmt"

	domainguild "mcgame-server/internal/domain/guild"
)

func (s *Service) GetGuildForClient(ctx context.Context, guildID int64) (*domainguild.Guild, error) {
	g, err := s.GetGuild(ctx, guildID)
	if err != nil {
		return nil, err
	}

	return s.enrichGuild(ctx, g)
}

func (s *Service) GetGuildByMemberForClient(ctx context.Context, characterID int64) (*domainguild.Guild, *domainguild.GuildMember, error) {
	g, err := s.GetGuildByMember(ctx, characterID)
	if err != nil || g == nil {
		return g, nil, err
	}

	g, err = s.enrichGuild(ctx, g)
	if err != nil {
		return nil, nil, err
	}

	for _, member := range g.Members {
		if member.CharacterID == characterID {
			return g, member, nil
		}
	}

	return g, nil, nil
}

func (s *Service) ListGuilds(ctx context.Context) ([]*domainguild.Guild, error) {
	guilds, err := s.repo.ListGuilds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list guilds: %w", err)
	}

	for _, guildItem := range guilds {
		if guildItem == nil {
			continue
		}
		if err := s.loadGuildSettings(ctx, guildItem); err != nil {
			return nil, err
		}
	}

	return guilds, nil
}

func (s *Service) GetMemberByID(ctx context.Context, memberID int64) (*domainguild.GuildMember, error) {
	member, err := s.repo.GetMemberByID(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild member: %w", err)
	}
	return member, nil
}

func (s *Service) ApplyToGuild(ctx context.Context, guildID, characterID int64) (*domainguild.GuildApplication, error) {
	commands, ok := s.repo.(LifecycleCommandRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	return commands.ApplyGuildAtomic(ctx, guildID, characterID)
}

func (s *Service) GetPendingApplicationByCharacterID(ctx context.Context, characterID int64) (*domainguild.GuildApplication, *domainguild.Guild, error) {
	application, err := s.repo.GetApplicationByCharacterID(ctx, characterID)
	if err != nil || application == nil {
		return application, nil, err
	}

	guildItem, err := s.GetGuildForClient(ctx, application.GuildID)
	if err != nil {
		return nil, nil, err
	}

	return application, guildItem, nil
}

func (s *Service) ApproveApplication(ctx context.Context, applicationID, approverID int64) (*domainguild.GuildApplication, *domainguild.GuildMember, error) {
	commands, ok := s.repo.(LifecycleCommandRepository)
	if !ok {
		return nil, nil, ErrUnsupportedAction
	}
	result, err := commands.ApproveGuildApplicationAtomic(ctx, applicationID, approverID)
	if err != nil {
		return nil, nil, err
	}
	return result.Application, result.Member, nil
}

func (s *Service) RefuseApplication(ctx context.Context, applicationID, approverID int64) (*domainguild.GuildApplication, error) {
	commands, ok := s.repo.(LifecycleCommandRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	return commands.RefuseGuildApplicationAtomic(ctx, applicationID, approverID)
}

func (s *Service) CancelApplication(ctx context.Context, applicationID, characterID int64) (*domainguild.GuildApplication, error) {
	commands, ok := s.repo.(LifecycleCommandRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	return commands.CancelGuildApplicationAtomic(ctx, applicationID, characterID)
}
