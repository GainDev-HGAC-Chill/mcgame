// Open-sourced by BaoLT

// Star-instance state reader maps the authoritative 12x12 completion grid to requirement IDs.
package achievement

import (
	"context"
	"fmt"
	"strconv"
)

type StarInstanceStateProvider interface {
	WarMapLoginGrid(ctx context.Context, characterID int64) (map[string]any, error)
}

type StarInstanceStateReader struct {
	provider StarInstanceStateProvider
}

func NewStarInstanceStateReader(provider StarInstanceStateProvider) *StarInstanceStateReader {
	return &StarInstanceStateReader{provider: provider}
}

func (r *StarInstanceStateReader) Name() string {
	return "star-instance"
}

func (r *StarInstanceStateReader) RequirementIDs() []int64 {
	ids := requirementIDRange(425, 461)
	return append(ids, requirementIDRange(463, 498)...)
}

func (r *StarInstanceStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.provider == nil {
		return nil, fmt.Errorf("star-instance state provider not configured")
	}
	grid, err := r.provider.WarMapLoginGrid(ctx, characterID)
	if err != nil {
		return nil, err
	}
	proofs := make([]StateProof, 0, 73)
	proofs = append(proofs, stateProof(425, completedStarInstanceCount(grid), 1))
	for requirementID := int64(426); requirementID <= 437; requirementID++ {
		proofs = append(proofs, starInstanceProof(requirementID, grid, 1, int(requirementID-425)))
	}
	for requirementID := int64(438); requirementID <= 449; requirementID++ {
		proofs = append(proofs, starInstanceProof(requirementID, grid, 6, int(450-requirementID)))
	}
	for requirementID := int64(450); requirementID <= 461; requirementID++ {
		proofs = append(proofs, starInstanceProof(requirementID, grid, 12, int(462-requirementID)))
	}
	for requirementID := int64(463); requirementID <= 486; requirementID++ {
		offset := int(requirementID - 463)
		level := 6
		if offset%2 == 1 {
			level = 12
		}
		proofs = append(proofs, starInstanceProof(requirementID, grid, level, (offset/2)+1))
	}
	for requirementID := int64(487); requirementID <= 498; requirementID++ {
		proofs = append(proofs, starInstanceProof(requirementID, grid, 12, int(requirementID-486)))
	}
	return proofs, nil
}

func starInstanceProof(requirementID int64, grid map[string]any, level, sid int) StateProof {
	progress := 0
	if starInstanceScore(grid, level, sid) > 0 {
		progress = 1
	}
	return stateProof(requirementID, progress, 1)
}

func completedStarInstanceCount(grid map[string]any) int {
	count := 0
	for level := 1; level <= 12; level++ {
		for sid := 1; sid <= 12; sid++ {
			if starInstanceScore(grid, level, sid) > 0 {
				count++
			}
		}
	}
	return count
}

func starInstanceScore(grid map[string]any, level, sid int) float64 {
	levelData, ok := grid[strconv.Itoa(level)].(map[string]any)
	if !ok {
		return 0
	}
	switch score := levelData[strconv.Itoa(sid)].(type) {
	case float64:
		return score
	case float32:
		return float64(score)
	case int:
		return float64(score)
	case int64:
		return float64(score)
	default:
		return 0
	}
}
