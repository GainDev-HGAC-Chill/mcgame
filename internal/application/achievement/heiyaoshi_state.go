// Open-sourced by BaoLT

// Heiyaoshi state reader proves completed formation milestones from persisted feature state.
package achievement

import (
	"context"
	"fmt"

	appheiyaoshi "mcgame-server/internal/application/heiyaoshi"
	domainfeature "mcgame-server/internal/domain/statfeature"
)

type CharacterFeatureStateRepository interface {
	ListCharacterFeatureStates(ctx context.Context, characterID int64) ([]*domainfeature.CharacterFeatureState, error)
}

type HeiyaoshiStateReader struct {
	repo CharacterFeatureStateRepository
}

func NewHeiyaoshiStateReader(repo CharacterFeatureStateRepository) *HeiyaoshiStateReader {
	return &HeiyaoshiStateReader{repo: repo}
}

func (r *HeiyaoshiStateReader) Name() string {
	return "heiyaoshi"
}

func (r *HeiyaoshiStateReader) RequirementIDs() []int64 {
	return requirementIDRange(699, 709)
}

func (r *HeiyaoshiStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("heiyaoshi state repository not configured")
	}
	states, err := r.repo.ListCharacterFeatureStates(ctx, characterID)
	if err != nil {
		return nil, err
	}
	completed := 0
	for _, featureState := range states {
		if featureState == nil || featureState.CharacterID != characterID || featureState.FeatureKey != domainfeature.FeatureHeiyaoshi {
			continue
		}
		state := appheiyaoshi.DecodeState(featureState.State)
		completed = state.LastActFigure - appheiyaoshi.MinFigure
		if completed < 0 {
			completed = 0
		}
		if completed > appheiyaoshi.Stage1MaxFigure {
			completed = appheiyaoshi.Stage1MaxFigure
		}
		break
	}
	proofs := make([]StateProof, 0, 11)
	for requirementID := int64(699); requirementID <= 708; requirementID++ {
		proofs = append(proofs, stateProof(requirementID, completed, int(requirementID-698)))
	}
	proofs = append(proofs, stateProof(709, completed, appheiyaoshi.Stage1MaxFigure))
	return proofs, nil
}
