// Open-sourced by BaoLT

// State reconciliation converts authoritative positive proofs into idempotent requirement updates.
package achievement

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	domainachievement "mcgame-server/internal/domain/achievement"
)

type StateProof struct {
	RequirementID int64
	Progress      int
	Target        int
	Done          bool
}

type StateProofReader interface {
	Name() string
	RequirementIDs() []int64
	ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error)
}

type preparedStateBatch struct {
	readerName string
	updates    []domainachievement.RequirementUpdate
}

func (s *Service) RegisterStateProofReader(reader StateProofReader) error {
	if reader == nil {
		return fmt.Errorf("achievement state reader is nil")
	}
	name := strings.TrimSpace(reader.Name())
	if name == "" {
		return fmt.Errorf("achievement state reader name is empty")
	}
	if name != reader.Name() {
		return fmt.Errorf("achievement state reader name %q is not canonical", reader.Name())
	}
	for _, existing := range s.stateReaders {
		if existing != nil && existing.Name() == name {
			return fmt.Errorf("achievement state reader %q already registered", name)
		}
	}
	if s.requirements == nil {
		return fmt.Errorf("achievement requirement registry not configured")
	}
	ids := reader.RequirementIDs()
	if len(ids) == 0 {
		return fmt.Errorf("achievement state reader %q has no requirements", name)
	}
	local := make(map[int64]struct{}, len(ids))
	for _, requirementID := range ids {
		if _, duplicate := local[requirementID]; duplicate {
			return fmt.Errorf("achievement state reader %q repeats requirement %d", name, requirementID)
		}
		definition, ok := s.requirements.Get(requirementID)
		if !ok {
			return fmt.Errorf("achievement state reader %q owns unknown requirement %d", name, requirementID)
		}
		if definition.Authority != RequirementAuthorityState {
			return fmt.Errorf("achievement state reader %q owns non-state requirement %d", name, requirementID)
		}
		if owner, duplicate := s.stateOwners[requirementID]; duplicate {
			return fmt.Errorf("state requirement %d owned by both %s and %s", requirementID, owner, name)
		}
		local[requirementID] = struct{}{}
	}
	s.stateReaders = append(s.stateReaders, reader)
	for requirementID := range local {
		s.stateOwners[requirementID] = name
	}
	return nil
}

func (s *Service) ReconcileState(ctx context.Context, characterID int64, now time.Time) (domainachievement.ProgressUpdate, error) {
	if characterID <= 0 {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("invalid achievement character")
	}
	if s.requirements == nil {
		return domainachievement.ProgressUpdate{}, fmt.Errorf("achievement requirement registry not configured")
	}

	batches := make([]preparedStateBatch, 0, len(s.stateReaders))
	seenRequirements := make(map[int64]string)
	for _, reader := range s.stateReaders {
		proofs, err := reader.ReadStateProofs(ctx, characterID)
		if err != nil {
			return domainachievement.ProgressUpdate{}, fmt.Errorf("read achievement state %s: %w", reader.Name(), err)
		}
		updates, err := s.prepareStateProofs(reader.Name(), proofs, seenRequirements)
		if err != nil {
			return domainachievement.ProgressUpdate{}, err
		}
		if len(updates) > 0 {
			batches = append(batches, preparedStateBatch{readerName: reader.Name(), updates: updates})
		}
	}

	result := emptyProgressUpdate()
	for _, batch := range batches {
		eventKey := stateBatchEventKey(batch.readerName, batch.updates)
		updated, err := s.recordEvent(ctx, characterID, eventKey, batch.updates, now)
		if err != nil {
			return result, fmt.Errorf("record achievement state %s: %w", batch.readerName, err)
		}
		result = mergeProgressUpdates(result, updated)
	}
	return result, nil
}

func (s *Service) prepareStateProofs(readerName string, proofs []StateProof, seen map[int64]string) ([]domainachievement.RequirementUpdate, error) {
	sorted := append([]StateProof(nil), proofs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].RequirementID < sorted[j].RequirementID })
	updates := make([]domainachievement.RequirementUpdate, 0, len(sorted))
	for _, proof := range sorted {
		if owner := s.stateOwners[proof.RequirementID]; owner != readerName {
			return nil, fmt.Errorf("state reader %s returned unowned requirement %d", readerName, proof.RequirementID)
		}
		definition, ok := s.requirements.Get(proof.RequirementID)
		if !ok {
			return nil, fmt.Errorf("state reader %s returned unknown requirement %d", readerName, proof.RequirementID)
		}
		if definition.Authority != RequirementAuthorityState {
			return nil, fmt.Errorf("state reader %s returned non-state requirement %d", readerName, proof.RequirementID)
		}
		if previous, duplicate := seen[proof.RequirementID]; duplicate {
			return nil, fmt.Errorf("state requirement %d returned by both %s and %s", proof.RequirementID, previous, readerName)
		}
		if proof.Progress < 0 || proof.Target < 0 {
			return nil, fmt.Errorf("state reader %s returned negative proof for requirement %d", readerName, proof.RequirementID)
		}
		target := definition.Target
		if definition.TargetSource == RequirementTargetAuthority {
			target = proof.Target
			if target <= 0 {
				return nil, fmt.Errorf("state reader %s omitted authority target for requirement %d", readerName, proof.RequirementID)
			}
		} else if proof.Target > 0 && proof.Target != definition.Target {
			return nil, fmt.Errorf("state reader %s target drift for requirement %d: got %d want %d", readerName, proof.RequirementID, proof.Target, definition.Target)
		}
		progress := proof.Progress
		if progress > target {
			progress = target
		}
		seen[proof.RequirementID] = readerName
		updates = append(updates, domainachievement.RequirementUpdate{
			RequirementID: definition.RequirementID,
			AchievementID: definition.AchievementID,
			Absolute:      &progress,
			Target:        target,
			Done:          proof.Done || progress >= target,
		})
	}
	return updates, nil
}

func stateBatchEventKey(readerName string, updates []domainachievement.RequirementUpdate) string {
	hash := sha256.New()
	for _, update := range updates {
		progress := 0
		if update.Absolute != nil {
			progress = *update.Absolute
		}
		fmt.Fprintf(hash, "%d:%d:%d:%t\n", update.RequirementID, progress, update.Target, update.Done)
	}
	return fmt.Sprintf("state:v1:%s:%x", readerName, hash.Sum(nil))
}
