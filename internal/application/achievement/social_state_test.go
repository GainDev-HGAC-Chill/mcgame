// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"

	domainsocial "mcgame-server/internal/domain/social"
)

type socialStateRepositoryFixture struct {
	byType map[int][]*domainsocial.Relationship
}

func (r socialStateRepositoryFixture) FindByCharacterAndType(ctx context.Context, characterID int64, relationshipType int) ([]*domainsocial.Relationship, error) {
	return r.byType[relationshipType], nil
}

func TestSocialStateReaderCountsUniqueOwnedRelationships(t *testing.T) {
	repo := socialStateRepositoryFixture{byType: map[int][]*domainsocial.Relationship{
		domainsocial.RelationshipTypeFriend: {
			{CharacterID: 99, OtherID: 1, Type: domainsocial.RelationshipTypeFriend},
			{CharacterID: 99, OtherID: 1, Type: domainsocial.RelationshipTypeFriend},
			{CharacterID: 98, OtherID: 2, Type: domainsocial.RelationshipTypeFriend},
		},
		domainsocial.RelationshipTypeTutor: {
			{CharacterID: 99, OtherID: 3, Type: domainsocial.RelationshipTypeTutor},
		},
		domainsocial.RelationshipTypeBrother: {
			{CharacterID: 99, OtherID: 4, Type: domainsocial.RelationshipTypeBrother},
		},
	}}
	proofs, err := NewSocialStateReader(repo).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	if byID[111].Progress != 1 || !byID[111].Done || byID[112].Progress != 1 || byID[112].Done {
		t.Fatalf("friend proofs = %#v %#v", byID[111], byID[112])
	}
	if !byID[113].Done || byID[114].Done || !byID[398].Done {
		t.Fatalf("relationship proofs = %#v", byID)
	}
}

var _ SocialStateRepository = socialStateRepositoryFixture{}
