// Open-sourced by BaoLT

package skill

import (
	"context"
	"reflect"
	"testing"

	"go.uber.org/zap"
	domainskill "mcgame-server/internal/domain/skill"
)

type testVariantAuthorizer struct {
	requested int
	learned   []int
	allowed   bool
}

func (a *testVariantAuthorizer) IsAuthorizedSkillVariant(_ context.Context, _ int64, requestedSkillID int, learnedSkillIDs []int) (bool, error) {
	a.requested = requestedSkillID
	a.learned = append([]int(nil), learnedSkillIDs...)
	return a.allowed, nil
}

func TestHasSkillUsesAwakeningVariantFallbackAfterExactOwnership(t *testing.T) {
	repo := newLifeSkillTestRepo(&domainskill.CharacterSkill{CharacterID: 7, SkillID: 101})
	service := NewService(repo, zap.NewNop())
	authorizer := &testVariantAuthorizer{allowed: true}
	service.SetAwakeningVariantAuthorizer(authorizer)

	hasSkill, err := service.HasSkill(context.Background(), 7, 201)
	if err != nil {
		t.Fatalf("HasSkill() error = %v", err)
	}
	if !hasSkill {
		t.Fatal("HasSkill() = false, want authorized variant")
	}
	if authorizer.requested != 201 || !reflect.DeepEqual(authorizer.learned, []int{101}) {
		t.Fatalf("authorizer received requested=%d learned=%v", authorizer.requested, authorizer.learned)
	}

	authorizer.requested = 0
	hasSkill, err = service.HasSkill(context.Background(), 7, 101)
	if err != nil || !hasSkill {
		t.Fatalf("exact HasSkill() = %v, %v", hasSkill, err)
	}
	if authorizer.requested != 0 {
		t.Fatal("exact ownership should not invoke variant authorizer")
	}
}
