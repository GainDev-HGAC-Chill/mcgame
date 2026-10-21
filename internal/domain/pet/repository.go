// Open-sourced by BaoLT

// Repository interface for pet persistence.
// Handles pet CRUD, follow state management, and ownership queries.
// Supports clearing follow state across all character pets.
package pet

import (
	"context"
)

type Repository interface {
	Save(ctx context.Context, pet *Pet) error
	FindByID(ctx context.Context, id int64) (*Pet, error)
	FindByCharacterID(ctx context.Context, characterID int64) ([]*Pet, error)
	FindFollowingPet(ctx context.Context, characterID int64) (*Pet, error)
	Delete(ctx context.Context, id int64) error
	Count(ctx context.Context, characterID int64) (int, error)
	UpdateFollowState(ctx context.Context, petID int64, isFollowing bool) error
	ClearFollowing(ctx context.Context, characterID int64) error
}

type FusionRepository interface {
	ApplyFusion(ctx context.Context, mainPet *Pet, consumedPetIDs []int64) error
}

type IllusionRepository interface {
	ApplyIllusion(ctx context.Context, mainPet *Pet, consumedPetID, luckItemID int64, luckItemCount int, updateMain bool) (int, error)
}

type SkillBookMutation struct {
	Pet                *Pet
	BookItemID         int64
	ExpectedBookStack  int
	ExpectedSkillSlots []int
}

type SkillBookRepository interface {
	ApplySkillBook(ctx context.Context, mutation SkillBookMutation) (int, error)
}
