// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"

	domainguild "mcgame-server/internal/domain/guild"
)

type guildStateRepositoryFixture struct {
	guild   *domainguild.Guild
	members []*domainguild.GuildMember
}

func (r guildStateRepositoryFixture) GetByMemberID(ctx context.Context, characterID int64) (*domainguild.Guild, error) {
	return r.guild, nil
}

func (r guildStateRepositoryFixture) GetMembers(ctx context.Context, guildID int64) ([]*domainguild.GuildMember, error) {
	return r.members, nil
}

func TestGuildStateReaderUsesCharacterMembershipContribution(t *testing.T) {
	repo := guildStateRepositoryFixture{
		guild: &domainguild.Guild{ID: 7},
		members: []*domainguild.GuildMember{
			{GuildID: 7, CharacterID: 99, ContributionTotal: 1000000},
			{GuildID: 7, CharacterID: 100, ContributionTotal: 10000000},
		},
	}
	proofs, err := NewGuildStateReader(repo).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	if !byID[115].Done || !byID[116].Done || !byID[117].Done || byID[118].Done {
		t.Fatalf("guild proofs = %#v", byID)
	}
}

func TestGuildStateReaderReturnsZeroProofsWithoutMembership(t *testing.T) {
	proofs, err := NewGuildStateReader(guildStateRepositoryFixture{}).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	for _, proof := range proofs {
		if proof.Progress != 0 || proof.Done {
			t.Fatalf("non-member proof = %#v", proof)
		}
	}
}

var _ GuildStateRepository = guildStateRepositoryFixture{}
