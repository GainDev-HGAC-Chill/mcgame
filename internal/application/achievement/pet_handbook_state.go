// Open-sourced by BaoLT

// Pet-handbook state reader proves activation coverage for the eight canonical pet classes.
package achievement

import (
	"context"
	"fmt"
	"strconv"

	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"
)

const petHandbookClassSize = 10

type PetHandbookStateCatalog interface {
	GetAllCreatureHandbooks() []*models.CreatureHandbookTemplate
}

type PetHandbookStateReader struct {
	repo    CharacterFeatureStateRepository
	catalog PetHandbookStateCatalog
}

func NewPetHandbookStateReader(repo CharacterFeatureStateRepository, catalog PetHandbookStateCatalog) *PetHandbookStateReader {
	return &PetHandbookStateReader{repo: repo, catalog: catalog}
}

func (r *PetHandbookStateReader) Name() string {
	return "pet-handbook"
}

func (r *PetHandbookStateReader) RequirementIDs() []int64 {
	return requirementIDRange(539, 554)
}

func (r *PetHandbookStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil || r.catalog == nil {
		return nil, fmt.Errorf("pet-handbook state dependencies not configured")
	}
	classEntries, err := petHandbookClassEntries(r.catalog.GetAllCreatureHandbooks())
	if err != nil {
		return nil, err
	}
	states, err := r.repo.ListCharacterFeatureStates(ctx, characterID)
	if err != nil {
		return nil, err
	}
	activeState := map[string]interface{}{}
	found := false
	for _, featureState := range states {
		if featureState == nil || featureState.CharacterID != characterID || featureState.FeatureKey != domainfeature.FeatureActivePet {
			continue
		}
		if found {
			return nil, fmt.Errorf("duplicate pet-handbook state for character %d", characterID)
		}
		found = true
		activeState = featureState.State
	}
	activeByClass := make(map[int]int, 8)
	for classID := 1; classID <= 8; classID++ {
		for _, entry := range classEntries[classID] {
			active, err := petHandbookEntryActive(activeState, int(entry.ID))
			if err != nil {
				return nil, err
			}
			if active {
				activeByClass[classID]++
			}
		}
	}
	proofs := make([]StateProof, 0, 16)
	for classID := 1; classID <= 8; classID++ {
		proofs = append(proofs, stateProof(int64(538+classID), activeByClass[classID], petHandbookClassSize))
	}
	for classID := 1; classID <= 8; classID++ {
		proofs = append(proofs, stateProof(int64(546+classID), activeByClass[classID], 1))
	}
	return proofs, nil
}

func petHandbookClassEntries(entries []*models.CreatureHandbookTemplate) (map[int][]*models.CreatureHandbookTemplate, error) {
	byClass := make(map[int][]*models.CreatureHandbookTemplate, 8)
	seen := make(map[int64]struct{}, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.ID <= 0 {
			return nil, fmt.Errorf("pet-handbook catalog contains invalid entry")
		}
		if _, duplicate := seen[entry.ID]; duplicate {
			return nil, fmt.Errorf("pet-handbook catalog repeats entry %d", entry.ID)
		}
		seen[entry.ID] = struct{}{}
		classID, ok := exactPetCatalogInt(entry.ClassID)
		if !ok {
			return nil, fmt.Errorf("pet-handbook entry %d has invalid class", entry.ID)
		}
		if classID >= 1 && classID <= 8 {
			byClass[classID] = append(byClass[classID], entry)
		}
	}
	for classID := 1; classID <= 8; classID++ {
		if len(byClass[classID]) != petHandbookClassSize {
			return nil, fmt.Errorf("pet-handbook class %d has %d entries, want %d", classID, len(byClass[classID]), petHandbookClassSize)
		}
	}
	return byClass, nil
}

func petHandbookEntryActive(state map[string]interface{}, handbookID int) (bool, error) {
	rawEntry, exists := state[strconv.Itoa(handbookID)]
	if !exists {
		return false, nil
	}
	entry, ok := rawEntry.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("pet-handbook entry %d has invalid state", handbookID)
	}
	rawActive, exists := entry["actived"]
	if !exists {
		return false, nil
	}
	if active, ok := rawActive.(bool); ok {
		return active, nil
	}
	active, ok := stateInt(rawActive)
	if !ok || (active != 0 && active != 1) {
		return false, fmt.Errorf("pet-handbook entry %d has invalid activation", handbookID)
	}
	return active == 1, nil
}
