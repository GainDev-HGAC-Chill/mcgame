// Open-sourced by BaoLT

// Social state reader proves friend, apprentice, and sworn-sibling achievements.
package achievement

import (
	"context"
	"fmt"

	domainsocial "mcgame-server/internal/domain/social"
)

type SocialStateRepository interface {
	FindByCharacterAndType(ctx context.Context, characterID int64, relationshipType int) ([]*domainsocial.Relationship, error)
}

type SocialStateReader struct {
	repo SocialStateRepository
}

func NewSocialStateReader(repo SocialStateRepository) *SocialStateReader {
	return &SocialStateReader{repo: repo}
}

func (r *SocialStateReader) Name() string {
	return "social"
}

func (r *SocialStateReader) RequirementIDs() []int64 {
	return []int64{111, 112, 113, 114, 398}
}

func (r *SocialStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("social state repository not configured")
	}
	friends, err := r.repo.FindByCharacterAndType(ctx, characterID, domainsocial.RelationshipTypeFriend)
	if err != nil {
		return nil, err
	}
	students, err := r.repo.FindByCharacterAndType(ctx, characterID, domainsocial.RelationshipTypeTutor)
	if err != nil {
		return nil, err
	}
	brothers, err := r.repo.FindByCharacterAndType(ctx, characterID, domainsocial.RelationshipTypeBrother)
	if err != nil {
		return nil, err
	}
	friendCount := uniqueRelationshipCount(friends, characterID, domainsocial.RelationshipTypeFriend)
	studentCount := uniqueRelationshipCount(students, characterID, domainsocial.RelationshipTypeTutor)
	brotherCount := uniqueRelationshipCount(brothers, characterID, domainsocial.RelationshipTypeBrother)
	return []StateProof{
		stateProof(111, friendCount, 1),
		stateProof(112, friendCount, 80),
		stateProof(113, studentCount, 1),
		stateProof(114, studentCount, 30),
		stateProof(398, brotherCount, 1),
	}, nil
}

func uniqueRelationshipCount(relationships []*domainsocial.Relationship, characterID int64, relationshipType int) int {
	seen := make(map[int64]struct{})
	for _, relationship := range relationships {
		if relationship == nil || relationship.CharacterID != characterID || relationship.Type != relationshipType || relationship.OtherID <= 0 || relationship.OtherID == characterID {
			continue
		}
		seen[relationship.OtherID] = struct{}{}
	}
	return len(seen)
}
