// Open-sourced by BaoLT

// PRS state reader proves activation of audited divine-form IDs.
package achievement

import (
	"context"
	"fmt"
	"strconv"
)

type PRSStateProvider interface {
	GetPanel(ctx context.Context, characterID int64) (map[string]interface{}, error)
}

type PRSStateReader struct {
	provider PRSStateProvider
}

func NewPRSStateReader(provider PRSStateProvider) *PRSStateReader {
	return &PRSStateReader{provider: provider}
}

func (r *PRSStateReader) Name() string {
	return "prs"
}

func (r *PRSStateReader) RequirementIDs() []int64 {
	ids := requirementIDRange(715, 723)
	return append(ids, 725, 734, 735, 736, 758, 759)
}

func (r *PRSStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.provider == nil {
		return nil, fmt.Errorf("prs state provider not configured")
	}
	panel, err := r.provider.GetPanel(ctx, characterID)
	if err != nil {
		return nil, err
	}
	activated := activatedPRSShowIDs(panel)
	proofs := make([]StateProof, 0, 15)
	for requirementID := int64(715); requirementID <= 723; requirementID++ {
		proofs = append(proofs, activatedIDProof(requirementID, activated, int(requirementID-714)))
	}
	proofs = append(proofs,
		activatedIDProof(725, activated, 10),
		activatedIDProof(734, activated, 11),
		activatedIDProof(735, activated, 12),
		activatedIDProof(736, activated, 13),
		activatedAnyIDProof(758, activated, 16, 17),
		activatedIDProof(759, activated, 18),
	)
	return proofs, nil
}

func activatedPRSShowIDs(panel map[string]interface{}) map[int]struct{} {
	activated := make(map[int]struct{})
	if permanent, ok := panel["actArr"].(map[string]interface{}); ok {
		for _, rawShowID := range permanent {
			if showID, ok := stateInt(rawShowID); ok && showID > 0 {
				activated[showID] = struct{}{}
			}
		}
	}
	if limited, ok := panel["actLimitObj"].(map[string]interface{}); ok {
		for rawShowID := range limited {
			showID, err := strconv.Atoi(rawShowID)
			if err == nil && showID > 0 {
				activated[showID] = struct{}{}
			}
		}
	}
	return activated
}

func activatedIDProof(requirementID int64, activated map[int]struct{}, showID int) StateProof {
	_, exists := activated[showID]
	progress := 0
	if exists {
		progress = 1
	}
	return stateProof(requirementID, progress, 1)
}

func activatedAnyIDProof(requirementID int64, activated map[int]struct{}, showIDs ...int) StateProof {
	for _, showID := range showIDs {
		if _, exists := activated[showID]; exists {
			return stateProof(requirementID, 1, 1)
		}
	}
	return stateProof(requirementID, 0, 1)
}

func stateInt(value interface{}) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		result := int(typed)
		return result, float64(result) == typed
	default:
		return 0, false
	}
}
