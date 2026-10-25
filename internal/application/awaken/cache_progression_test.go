package awaken

import (
	"context"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
)

type copyingCharacterRepo struct {
	fakeCharRepo
}

func (r *copyingCharacterRepo) FindByID(context.Context, int64) (*domainchar.Character, error) {
	copy := *r.char
	return &copy, nil
}

func (r *copyingCharacterRepo) Update(_ context.Context, char *domainchar.Character) error {
	copy := *char
	r.char = &copy
	return nil
}

func TestAwakenSynchronizesCopiedCharacterAcrossSuccessiveAttempts(t *testing.T) {
	char := testChar()
	char.AwakenLevel = 0
	items := &fakeItemPort{count: 50, consumeOK: true}
	progress := &fakeProgress{}
	service, _ := newGachaService(t, char, items, progress, fixedRoller{v: 0})
	repo := &copyingCharacterRepo{fakeCharRepo{char: char}}
	service.SetCharacterRepository(repo)
	for level := 1; level <= 2; level++ {
		result, err := service.Awaken(context.Background(), char.ID, false)
		if err != nil || result["awakenLevel"] != level || repo.char.AwakenLevel != level {
			t.Fatalf("attempt %d: result=%v cachedLevel=%d error=%v", level, result, repo.char.AwakenLevel, err)
		}
	}
	if progress.level != 2 || items.consumed != 22 || repo.char.Money != char.Money-580000 {
		t.Fatalf("progress=%d consumed=%d money=%d", progress.level, items.consumed, repo.char.Money)
	}
}
