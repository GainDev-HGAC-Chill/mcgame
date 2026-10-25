// Open-sourced by BaoLT

// Guild state reader proves membership and persisted contribution milestones.
package achievement

import (
	"context"
	"fmt"

	domainguild "mcgame-server/internal/domain/guild"
)

type GuildStateRepository interface {
	GetByMemberID(ctx context.Context, characterID int64) (*domainguild.Guild, error)
	GetMembers(ctx context.Context, guildID int64) ([]*domainguild.GuildMember, error)
}

type GuildStateReader struct {
	repo GuildStateRepository
}

func NewGuildStateReader(repo GuildStateRepository) *GuildStateReader {
	return &GuildStateReader{repo: repo}
}

func (r *GuildStateReader) Name() string {
	return "guild"
}

func (r *GuildStateReader) RequirementIDs() []int64 {
	return []int64{115, 116, 117, 118}
}

func (r *GuildStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("guild state repository not configured")
	}
	guild, err := r.repo.GetByMemberID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if guild == nil {
		return guildStateProofs(0, false), nil
	}
	members, err := r.repo.GetMembers(ctx, guild.ID)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		if member != nil && member.CharacterID == characterID && member.GuildID == guild.ID {
			return guildStateProofs(member.ContributionTotal, true), nil
		}
	}
	return nil, fmt.Errorf("character %d missing from guild %d member list", characterID, guild.ID)
}

func guildStateProofs(contribution int64, member bool) []StateProof {
	membership := 0
	if member {
		membership = 1
	}
	progress := boundedStateProgress(contribution)
	return []StateProof{
		stateProof(115, membership, 1),
		stateProof(116, progress, 100000),
		stateProof(117, progress, 1000000),
		stateProof(118, progress, 10000000),
	}
}
