// Open-sourced by BaoLT

package achievement

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

type stateProofReaderFixture struct {
	name   string
	proofs []StateProof
	err    error
}

func (r stateProofReaderFixture) Name() string {
	return r.name
}

func (r stateProofReaderFixture) RequirementIDs() []int64 {
	ids := make([]int64, 0, len(r.proofs))
	for _, proof := range r.proofs {
		ids = append(ids, proof.RequirementID)
	}
	return ids
}

func (r stateProofReaderFixture) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	return r.proofs, r.err
}

func TestReconcileStateBuildsCanonicalPositiveUpdate(t *testing.T) {
	repo := &achievementTestRepo{}
	service := NewService(repo, zap.NewNop())
	service.SetGameDataManager(loadAchievementCatalogFixture(t))
	if err := service.RegisterStateProofReader(stateProofReaderFixture{
		name: "character",
		proofs: []StateProof{
			{RequirementID: 1, Progress: 52, Target: 10},
			{RequirementID: 59, Progress: 1},
		},
	}); err != nil {
		t.Fatalf("RegisterStateProofReader() error = %v", err)
	}

	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	if _, err := service.ReconcileState(context.Background(), 99, now); err != nil {
		t.Fatalf("ReconcileState() error = %v", err)
	}
	if len(repo.events) != 1 || !strings.HasPrefix(repo.events[0].key, "state:v1:character:") {
		t.Fatalf("events = %#v", repo.events)
	}
	updates := repo.events[0].updates
	if len(updates) != 2 {
		t.Fatalf("updates = %#v", updates)
	}
	if updates[0].RequirementID != 1 || updates[0].AchievementID != 1001 || updates[0].Target != 10 || updates[0].Absolute == nil || *updates[0].Absolute != 10 || !updates[0].Done {
		t.Fatalf("level update = %#v", updates[0])
	}
	if updates[1].RequirementID != 59 || updates[1].Target != 30 {
		t.Fatalf("catalog-target update = %#v", updates[1])
	}

	if _, err := service.ReconcileState(context.Background(), 99, now.Add(time.Minute)); err != nil {
		t.Fatalf("second ReconcileState() error = %v", err)
	}
	if len(repo.events) != 2 || repo.events[0].key != repo.events[1].key {
		t.Fatalf("state event keys are not stable: %#v", repo.events)
	}
}

func TestReconcileStateRejectsNonStateAndMissingAuthorityTarget(t *testing.T) {
	tests := []struct {
		name             string
		proof            StateProof
		want             string
		registrationFail bool
	}{
		{name: "event", proof: StateProof{RequirementID: 53, Progress: 1, Target: 8}, want: "non-state", registrationFail: true},
		{name: "unsupported", proof: StateProof{RequirementID: 216, Progress: 1, Target: 1}, want: "non-state", registrationFail: true},
		{name: "missing target", proof: StateProof{RequirementID: 1, Progress: 10}, want: "omitted authority target"},
		{name: "unknown", proof: StateProof{RequirementID: 999999, Progress: 1, Target: 1}, want: "unknown requirement", registrationFail: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &achievementTestRepo{}
			service := NewService(repo, zap.NewNop())
			service.SetGameDataManager(loadAchievementCatalogFixture(t))
			registrationErr := service.RegisterStateProofReader(stateProofReaderFixture{name: "fixture", proofs: []StateProof{test.proof}})
			if test.registrationFail {
				if registrationErr == nil || !strings.Contains(registrationErr.Error(), test.want) {
					t.Fatalf("RegisterStateProofReader() error = %v, want %q", registrationErr, test.want)
				}
				return
			}
			if registrationErr != nil {
				t.Fatalf("RegisterStateProofReader() error = %v", registrationErr)
			}
			_, err := service.ReconcileState(context.Background(), 99, time.Now())
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ReconcileState() error = %v, want %q", err, test.want)
			}
			if len(repo.events) != 0 {
				t.Fatalf("invalid proof recorded events: %#v", repo.events)
			}
		})
	}
}

func TestReconcileStateValidatesAllReadersBeforeWriting(t *testing.T) {
	repo := &achievementTestRepo{}
	service := NewService(repo, zap.NewNop())
	service.SetGameDataManager(loadAchievementCatalogFixture(t))
	first := stateProofReaderFixture{name: "first", proofs: []StateProof{{RequirementID: 1, Progress: 10, Target: 10}}}
	second := stateProofReaderFixture{name: "second", proofs: []StateProof{{RequirementID: 1, Progress: 10, Target: 10}}}
	if err := service.RegisterStateProofReader(first); err != nil {
		t.Fatalf("RegisterStateProofReader(first) error = %v", err)
	}
	if err := service.RegisterStateProofReader(second); err == nil || !strings.Contains(err.Error(), "owned by both") {
		t.Fatalf("RegisterStateProofReader(second) error = %v", err)
	}
	if len(repo.events) != 0 {
		t.Fatalf("events written during registration: %#v", repo.events)
	}
}

func TestStateReaderOwnershipMatchesImplementedCoverage(t *testing.T) {
	service := NewService(&achievementTestRepo{}, zap.NewNop())
	service.SetGameDataManager(loadAchievementCatalogFixture(t))
	readers := []StateProofReader{
		NewCharacterStateReader(nil),
		NewSocialStateReader(nil),
		NewGuildStateReader(nil),
		NewTitleStateReader(nil),
		NewAchievementPointsStateReader(nil, nil),
		NewStarInstanceStateReader(nil),
		NewHeiyaoshiStateReader(nil),
		NewPRSStateReader(nil),
		NewMysteryTreasureStateReader(nil, nil),
		NewQuestStateReader(nil),
		NewLifeSkillStateReader(nil, nil),
	}
	for _, reader := range readers {
		if err := service.RegisterStateProofReader(reader); err != nil {
			t.Fatalf("RegisterStateProofReader(%s) error = %v", reader.Name(), err)
		}
	}
	if len(service.stateOwners) != 177 {
		t.Fatalf("state reader coverage = %d, want 177", len(service.stateOwners))
	}
	for requirementID, owner := range service.stateOwners {
		definition, ok := service.requirements.Get(requirementID)
		if !ok || definition.Authority != RequirementAuthorityState || owner == "" {
			t.Fatalf("requirement %d owner=%q definition=%#v found=%t", requirementID, owner, definition, ok)
		}
	}
}

var _ StateProofReader = stateProofReaderFixture{}
