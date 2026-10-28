// Guild permission checks use the same rank settings returned to the Flash client.
package guild

import (
	"context"
	"fmt"

	domainguild "mcgame-server/internal/domain/guild"
	pkgerrors "mcgame-server/pkg/errors"
)

func (s *Service) AuthorizeAction(ctx context.Context, guildID, actorID int64, permission domainguild.Permission) error {
	switch permission {
	case domainguild.PermissionAdd, domainguild.PermissionQuest, domainguild.PermissionSlot, domainguild.PermissionInfo, domainguild.PermissionDel, domainguild.PermissionDuty:
	default:
		return pkgerrors.ErrUnauthorized
	}
	guildItem, member, err := s.loadGuildActor(ctx, guildID, actorID)
	if err != nil {
		return err
	}
	return s.authorizeMember(ctx, guildItem, member, permission, false)
}

func (s *Service) AuthorizeLeader(ctx context.Context, guildID, actorID int64) error {
	guildItem, member, err := s.loadGuildActor(ctx, guildID, actorID)
	if err != nil {
		return err
	}
	return s.authorizeMember(ctx, guildItem, member, "", true)
}

func (s *Service) loadGuildActor(ctx context.Context, guildID, actorID int64) (*domainguild.Guild, *domainguild.GuildMember, error) {
	guildItem, err := s.repo.GetByID(ctx, guildID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get guild: %w", err)
	}
	if guildItem == nil {
		return nil, nil, ErrGuildNotFound
	}
	members, err := s.repo.GetMembers(ctx, guildID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get guild members: %w", err)
	}
	for _, member := range members {
		if member != nil && member.CharacterID == actorID {
			return guildItem, member, nil
		}
	}
	return nil, nil, ErrNotInGuild
}

func (s *Service) authorizeMember(ctx context.Context, guildItem *domainguild.Guild, member *domainguild.GuildMember, permission domainguild.Permission, leaderOnly bool) error {
	if guildItem == nil || member == nil || member.GuildID != guildItem.ID {
		return ErrNotInGuild
	}
	if guildItem.LeaderID == member.CharacterID && member.Rank == domainguild.RankLeader {
		return nil
	}
	if leaderOnly {
		return pkgerrors.ErrUnauthorized
	}
	settings, err := s.repo.LoadRankSettings(ctx, guildItem.ID)
	if err != nil {
		return fmt.Errorf("failed to load guild rank settings: %w", err)
	}
	if len(settings) == 0 {
		settings = DefaultRankSettings()
	}
	if err := domainguild.ValidateRankSettings(settings); err != nil {
		return fmt.Errorf("invalid guild rank settings: %w", err)
	}
	for _, setting := range settings {
		if setting.Rank == member.Rank && setting.Allows(permission) {
			return nil
		}
	}
	return pkgerrors.ErrUnauthorized
}
