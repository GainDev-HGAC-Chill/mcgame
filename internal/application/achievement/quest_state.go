// Open-sourced by BaoLT

// Quest state reader proves exact audited quest completions from durable quest history.
package achievement

import (
	"context"
	"fmt"
)

type QuestStateRepository interface {
	GetCompletedQuestIDs(ctx context.Context, characterID int64) ([]int, error)
}

type QuestStateReader struct {
	repo QuestStateRepository
}

var exactQuestRequirements = map[int64][]int{
	259: {686, 687, 688, 689, 690, 691},
	260: {677},
	261: {2471},
	262: {4045},
	263: {170},
	264: {1134},
	265: {1256},
	266: {1299},
	267: {1366},
	268: {3380},
	269: {4649},
	511: {5414},
}

func NewQuestStateReader(repo QuestStateRepository) *QuestStateReader {
	return &QuestStateReader{repo: repo}
}

func (r *QuestStateReader) Name() string {
	return "quest"
}

func (r *QuestStateReader) RequirementIDs() []int64 {
	return []int64{259, 260, 261, 262, 263, 264, 265, 266, 267, 268, 269, 511}
}

func (r *QuestStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("quest state repository not configured")
	}
	completedIDs, err := r.repo.GetCompletedQuestIDs(ctx, characterID)
	if err != nil {
		return nil, err
	}
	completed := make(map[int]struct{}, len(completedIDs))
	for _, questID := range completedIDs {
		if questID > 0 {
			completed[questID] = struct{}{}
		}
	}
	proofs := make([]StateProof, 0, len(exactQuestRequirements))
	for _, requirementID := range r.RequirementIDs() {
		done := false
		for _, questID := range exactQuestRequirements[requirementID] {
			if _, exists := completed[questID]; exists {
				done = true
				break
			}
		}
		progress := 0
		if done {
			progress = 1
		}
		proofs = append(proofs, stateProof(requirementID, progress, 1))
	}
	return proofs, nil
}
