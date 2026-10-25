// Open-sourced by BaoLT

// Mystery Treasure state reader proves complete codex collections against the canonical catalog.
package achievement

import (
	"context"
	"fmt"
	"strconv"

	appmystre "mcgame-server/internal/application/mysterytreasure"
	"mcgame-server/internal/gamedata/models"
)

type MysteryTreasureStateProvider interface {
	Load(ctx context.Context, characterID int64) (*appmystre.MysteryTreasureState, error)
}

type MysteryTreasureCatalog interface {
	MysGetAllMystres() []*models.MystreTemplate
}

type MysteryTreasureStateReader struct {
	provider MysteryTreasureStateProvider
	catalog  MysteryTreasureCatalog
}

func NewMysteryTreasureStateReader(provider MysteryTreasureStateProvider, catalog MysteryTreasureCatalog) *MysteryTreasureStateReader {
	return &MysteryTreasureStateReader{provider: provider, catalog: catalog}
}

func (r *MysteryTreasureStateReader) Name() string {
	return "mystery-treasure"
}

func (r *MysteryTreasureStateReader) RequirementIDs() []int64 {
	return requirementIDRange(675, 686)
}

func (r *MysteryTreasureStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.provider == nil || r.catalog == nil {
		return nil, fmt.Errorf("mystery-treasure state dependencies not configured")
	}
	state, err := r.provider.Load(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("mystery-treasure state is nil")
	}

	expectedByKind, err := mysteryTreasureCatalogIDs(r.catalog.MysGetAllMystres())
	if err != nil {
		return nil, err
	}
	proofs := make([]StateProof, 0, 12)
	totalProgress := 0
	totalTarget := 0
	for kind := 1; kind <= 11; kind++ {
		expected := expectedByKind[kind]
		progress := collectedMysteryTreasureCount(state.MysBook[strconv.Itoa(kind)], expected)
		target := len(expected)
		totalProgress += progress
		totalTarget += target
		proofs = append(proofs, stateProof(int64(675+kind), progress, target))
	}
	proofs = append([]StateProof{stateProof(675, totalProgress, totalTarget)}, proofs...)
	return proofs, nil
}

func mysteryTreasureCatalogIDs(templates []*models.MystreTemplate) (map[int]map[int]struct{}, error) {
	result := make(map[int]map[int]struct{}, 11)
	seen := make(map[int]struct{}, len(templates))
	for _, template := range templates {
		if template == nil || template.ID <= 0 {
			return nil, fmt.Errorf("invalid mystery-treasure template")
		}
		id := int(template.ID)
		if int64(id) != template.ID {
			return nil, fmt.Errorf("mystery-treasure template ID overflow: %d", template.ID)
		}
		kind := int(template.Kind)
		if float64(kind) != template.Kind || kind < 1 || kind > 11 {
			return nil, fmt.Errorf("mystery-treasure template %d has invalid kind %s", template.ID, catalogNumber(template.Kind))
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, fmt.Errorf("duplicate mystery-treasure template %d", id)
		}
		seen[id] = struct{}{}
		if result[kind] == nil {
			result[kind] = make(map[int]struct{})
		}
		result[kind][id] = struct{}{}
	}
	for kind := 1; kind <= 11; kind++ {
		if len(result[kind]) == 0 {
			return nil, fmt.Errorf("mystery-treasure catalog missing kind %d", kind)
		}
	}
	return result, nil
}

func collectedMysteryTreasureCount(book map[string]int, expected map[int]struct{}) int {
	count := 0
	for id := range expected {
		if book[strconv.Itoa(id)] == id {
			count++
		}
	}
	return count
}
