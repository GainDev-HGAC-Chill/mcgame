// Open-sourced by BaoLT

// Deco-show state reader proves active decoration ownership from persisted hole flags.
package achievement

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"
)

type DecoShowStateCatalog interface {
	GetDecoShow(id int) *models.DecoShowTemplate
}

type decoShowRequirement struct {
	requirementID int64
	showID        int
	position      int
	suitID        int
}

var decoShowRequirements = [...]decoShowRequirement{
	{687, 1, 1, 2},
	{688, 2, 2, 2},
	{689, 3, 3, 2},
	{690, 4, 4, 2},
	{691, 9, 1, 1},
	{692, 10, 2, 1},
	{693, 11, 3, 1},
	{694, 12, 4, 1},
	{695, 5, 1, 3},
	{696, 6, 2, 3},
	{697, 7, 3, 3},
	{698, 8, 4, 3},
	{741, 13, 1, 4},
	{742, 14, 2, 4},
	{743, 15, 3, 4},
	{744, 16, 4, 4},
	{754, 17, 1, 5},
	{755, 18, 2, 5},
	{756, 19, 3, 5},
	{757, 20, 4, 5},
}

type DecoShowStateReader struct {
	repo    CharacterFeatureStateRepository
	catalog DecoShowStateCatalog
	now     func() time.Time
}

func NewDecoShowStateReader(repo CharacterFeatureStateRepository, catalog DecoShowStateCatalog) *DecoShowStateReader {
	return &DecoShowStateReader{repo: repo, catalog: catalog, now: time.Now}
}

func (r *DecoShowStateReader) Name() string {
	return "deco-show"
}

func (r *DecoShowStateReader) RequirementIDs() []int64 {
	ids := make([]int64, 0, len(decoShowRequirements))
	for _, definition := range decoShowRequirements {
		ids = append(ids, definition.requirementID)
	}
	return ids
}

func (r *DecoShowStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil || r.catalog == nil || r.now == nil {
		return nil, fmt.Errorf("deco-show state dependencies not configured")
	}
	if err := r.validateCatalog(); err != nil {
		return nil, err
	}
	states, err := r.repo.ListCharacterFeatureStates(ctx, characterID)
	if err != nil {
		return nil, err
	}
	owned := make(map[int]struct{})
	found := false
	for _, featureState := range states {
		if featureState == nil || featureState.CharacterID != characterID || featureState.FeatureKey != domainfeature.FeatureDecoHole {
			continue
		}
		if found {
			return nil, fmt.Errorf("duplicate deco-show state for character %d", characterID)
		}
		found = true
		owned, err = decodeOwnedDecoShows(featureState.State, r.now().UnixMilli())
		if err != nil {
			return nil, err
		}
	}
	proofs := make([]StateProof, 0, len(decoShowRequirements))
	for _, definition := range decoShowRequirements {
		_, exists := owned[definition.showID]
		progress := 0
		if exists {
			progress = 1
		}
		proofs = append(proofs, stateProof(definition.requirementID, progress, 1))
	}
	return proofs, nil
}

func (r *DecoShowStateReader) validateCatalog() error {
	for _, definition := range decoShowRequirements {
		template := r.catalog.GetDecoShow(definition.showID)
		position, positionOK := exactPetCatalogInt(templateDecoShowPosition(template))
		suitID, suitOK := exactPetCatalogInt(templateDecoShowSuitID(template))
		if template == nil || template.ID != int64(definition.showID) || !positionOK || !suitOK || position != definition.position || suitID != definition.suitID {
			return fmt.Errorf("deco-show requirement %d identity drift", definition.requirementID)
		}
	}
	return nil
}

func templateDecoShowPosition(template *models.DecoShowTemplate) float64 {
	if template == nil {
		return 0
	}
	return template.Position
}

func templateDecoShowSuitID(template *models.DecoShowTemplate) float64 {
	if template == nil {
		return 0
	}
	return template.SuitID
}

func decodeOwnedDecoShows(state map[string]interface{}, nowMs int64) (map[int]struct{}, error) {
	owned := make(map[int]struct{})
	positions, exists := state["positions"]
	if !exists || positions == nil {
		return owned, nil
	}
	positionMap, ok := positions.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid deco-show positions state")
	}
	for rawPosition, rawHole := range positionMap {
		position, err := strconv.Atoi(rawPosition)
		if err != nil || position < 1 || position > 4 {
			return nil, fmt.Errorf("invalid deco-show position %q", rawPosition)
		}
		hole, ok := rawHole.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid deco-show hole %d", position)
		}
		rawFlag, exists := hole["activeFlag"]
		if !exists || rawFlag == nil || rawFlag == "" || rawFlag == "{r:{},s:{}}" {
			continue
		}
		flag, ok := rawFlag.(string)
		if !ok {
			return nil, fmt.Errorf("invalid deco-show active flag at position %d", position)
		}
		shows, err := decodeDecoShowFlag(flag)
		if err != nil {
			return nil, fmt.Errorf("decode deco-show position %d: %w", position, err)
		}
		for rawShowID, due := range shows {
			showID, err := strconv.Atoi(rawShowID)
			if err != nil || showID <= 0 || showID > len(decoShowRequirements) {
				return nil, fmt.Errorf("invalid deco-show id %q", rawShowID)
			}
			expectedPosition := (showID-1)%4 + 1
			if expectedPosition != position {
				return nil, fmt.Errorf("deco-show %d stored under position %d", showID, position)
			}
			if due <= 0 {
				return nil, fmt.Errorf("invalid deco-show %d expiry", showID)
			}
			if due == 1 || due > nowMs {
				if _, duplicate := owned[showID]; duplicate {
					return nil, fmt.Errorf("duplicate deco-show %d ownership", showID)
				}
				owned[showID] = struct{}{}
			}
		}
	}
	return owned, nil
}

func decodeDecoShowFlag(raw string) (map[string]int64, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var flag struct {
		Shows map[string]json.Number `json:"s"`
	}
	if err := decoder.Decode(&flag); err != nil {
		return nil, err
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("trailing deco-show flag data")
	}
	shows := make(map[string]int64, len(flag.Shows))
	for showID, rawDue := range flag.Shows {
		due, err := rawDue.Int64()
		if err != nil {
			return nil, fmt.Errorf("invalid deco-show %s expiry", showID)
		}
		shows[showID] = due
	}
	return shows, nil
}
