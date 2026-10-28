// Open-sourced by BaoLT

package guild

import (
	"context"
	"fmt"

	domainguild "mcgame-server/internal/domain/guild"
)

func (s *Service) UpdateGuildName(ctx context.Context, guildID, updaterID int64, name string) (*domainguild.Guild, error) {
	existing, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check guild name: %w", err)
	}
	if existing != nil && existing.ID != guildID {
		return nil, ErrGuildNameTaken
	}

	g, err := s.repo.GetByID(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild: %w", err)
	}
	if g == nil {
		return nil, ErrGuildNotFound
	}
	if g.LeaderID != updaterID {
		return nil, ErrNotGuildLeader
	}

	commands, ok := s.repo.(GuildMetadataRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	if err := commands.SetGuildName(ctx, guildID, updaterID, name); err != nil {
		return nil, fmt.Errorf("failed to update guild name: %w", err)
	}

	return s.GetGuildForClient(ctx, guildID)
}

func (s *Service) UpdateGuildRankSettings(ctx context.Context, guildID, updaterID int64, settings []domainguild.RankSetting) error {
	if err := domainguild.ValidateRankSettings(settings); err != nil {
		return err
	}
	g, err := s.repo.GetByID(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild: %w", err)
	}
	if g == nil {
		return ErrGuildNotFound
	}
	if g.LeaderID != updaterID {
		return ErrNotGuildLeader
	}

	if err := s.repo.SaveRankSettings(ctx, guildID, updaterID, settings); err != nil {
		return fmt.Errorf("failed to save guild rank settings: %w", err)
	}

	return nil
}

func (s *Service) TransferLeadership(ctx context.Context, guildID, oldLeaderID, newLeaderID int64) (*domainguild.Guild, error) {
	commands, ok := s.repo.(LifecycleCommandRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	if _, err := commands.TransferGuildLeadershipAtomic(ctx, guildID, oldLeaderID, newLeaderID); err != nil {
		return nil, err
	}
	return s.GetGuildForClient(ctx, guildID)
}

func (s *Service) UpgradeGuildLevel(ctx context.Context, guildID, actorID int64) (*domainguild.Guild, error) {
	if err := s.AuthorizeLeader(ctx, guildID, actorID); err != nil {
		return nil, err
	}
	return nil, ErrUnsupportedAction
}

func (s *Service) IncreaseMemberCapacity(ctx context.Context, guildID, actorID int64) (*domainguild.Guild, error) {
	if err := s.AuthorizeLeader(ctx, guildID, actorID); err != nil {
		return nil, err
	}
	return nil, ErrUnsupportedAction
}

func (s *Service) IncreaseBankPageCount(ctx context.Context, guildID, actorID int64) (int, error) {
	if err := s.AuthorizeLeader(ctx, guildID, actorID); err != nil {
		return 0, err
	}
	return 0, ErrUnsupportedAction
}
