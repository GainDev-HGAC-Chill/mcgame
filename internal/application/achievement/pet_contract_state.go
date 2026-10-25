// Open-sourced by BaoLT

// Pet-contract state reader proves soul-contract levels from persisted feature state.
package achievement

import (
	"context"
	"fmt"

	domainfeature "mcgame-server/internal/domain/statfeature"
)

const maxPetContractLevel = 50

type petContractRequirement struct {
	requirementID int64
	stateKey      string
	target        int
}

var petContractRequirements = [...]petContractRequirement{
	{667, "hp", 25},
	{668, "speed", 25},
	{669, "physics", 25},
	{670, "magic", 25},
	{671, "hp", 50},
	{672, "speed", 50},
	{673, "physics", 50},
	{674, "magic", 50},
}

type PetContractStateReader struct {
	repo CharacterFeatureStateRepository
}

func NewPetContractStateReader(repo CharacterFeatureStateRepository) *PetContractStateReader {
	return &PetContractStateReader{repo: repo}
}

func (r *PetContractStateReader) Name() string {
	return "pet-contract"
}

func (r *PetContractStateReader) RequirementIDs() []int64 {
	return requirementIDRange(667, 674)
}

func (r *PetContractStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("pet-contract state repository not configured")
	}
	states, err := r.repo.ListCharacterFeatureStates(ctx, characterID)
	if err != nil {
		return nil, err
	}
	levels := map[string]int{"hp": 0, "speed": 0, "physics": 0, "magic": 0}
	found := false
	for _, featureState := range states {
		if featureState == nil || featureState.CharacterID != characterID || featureState.FeatureKey != domainfeature.FeatureContractPet {
			continue
		}
		if found {
			return nil, fmt.Errorf("duplicate pet-contract state for character %d", characterID)
		}
		found = true
		for stateKey := range levels {
			rawLevel, exists := featureState.State[stateKey]
			if !exists {
				continue
			}
			level, ok := stateInt(rawLevel)
			if !ok || level < 0 || level > maxPetContractLevel {
				return nil, fmt.Errorf("invalid pet-contract %s level", stateKey)
			}
			levels[stateKey] = level
		}
	}
	proofs := make([]StateProof, 0, len(petContractRequirements))
	for _, definition := range petContractRequirements {
		proofs = append(proofs, stateProof(definition.requirementID, levels[definition.stateKey], definition.target))
	}
	return proofs, nil
}
