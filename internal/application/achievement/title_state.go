// Open-sourced by BaoLT

// Title state reader proves distinct owned-title count milestones.
package achievement

import (
	"context"
	"fmt"
)

type TitleStateRepository interface {
	GetCharacterTitles(ctx context.Context, characterID int64) ([]int, error)
	GetCharacterSpecialTitles(ctx context.Context, characterID int64) ([]int, error)
}

type TitleStateReader struct {
	repo TitleStateRepository
}

func NewTitleStateReader(repo TitleStateRepository) *TitleStateReader {
	return &TitleStateReader{repo: repo}
}

func (r *TitleStateReader) Name() string {
	return "title"
}

func (r *TitleStateReader) RequirementIDs() []int64 {
	return []int64{50, 51, 52}
}

func (r *TitleStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("title state repository not configured")
	}
	titles, err := r.repo.GetCharacterTitles(ctx, characterID)
	if err != nil {
		return nil, err
	}
	specialTitles, err := r.repo.GetCharacterSpecialTitles(ctx, characterID)
	if err != nil {
		return nil, err
	}
	owned := make(map[int]struct{}, len(titles)+len(specialTitles))
	for _, titleID := range titles {
		if titleID > 0 {
			owned[titleID] = struct{}{}
		}
	}
	for _, titleID := range specialTitles {
		if titleID > 0 {
			owned[titleID] = struct{}{}
		}
	}
	count := len(owned)
	return []StateProof{
		stateProof(50, count, 5),
		stateProof(51, count, 10),
		stateProof(52, count, 15),
	}, nil
}
