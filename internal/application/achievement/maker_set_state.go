// Open-sourced by BaoLT

// Maker-set state reader proves equipped orange signature-set quality tiers.
package achievement

import (
	"context"
	"fmt"

	appitem "mcgame-server/internal/application/item"
)

type MakerSetStateProvider interface {
	CalculateMakerSet(ctx context.Context, characterID int64) appitem.MakerSetResult
}

type MakerSetStateReader struct {
	provider MakerSetStateProvider
}

func NewMakerSetStateReader(provider MakerSetStateProvider) *MakerSetStateReader {
	return &MakerSetStateReader{provider: provider}
}

func (r *MakerSetStateReader) Name() string {
	return "maker-set"
}

func (r *MakerSetStateReader) RequirementIDs() []int64 {
	return []int64{34, 35}
}

func (r *MakerSetStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.provider == nil {
		return nil, fmt.Errorf("maker-set state provider not configured")
	}
	result := r.provider.CalculateMakerSet(ctx, characterID)
	hasOrangeSet := 0
	hasExcellentOrangeSet := 0
	if result.Active && result.QualityType >= 16 {
		hasOrangeSet = 1
	}
	if result.Active && result.QualityType >= 20 {
		hasExcellentOrangeSet = 1
	}
	return []StateProof{
		stateProof(34, hasOrangeSet, 1),
		stateProof(35, hasExcellentOrangeSet, 1),
	}, nil
}
