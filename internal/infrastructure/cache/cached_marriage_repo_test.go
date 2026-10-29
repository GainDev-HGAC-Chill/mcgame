package cache

import (
	"context"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	domainmarriage "mcgame-server/internal/domain/marriage"
)

type marriageCommandTestRepo struct {
	domainmarriage.Repository
	bookResult    *domainmarriage.BookWeddingResult
	divorceResult *domainmarriage.DivorceResult
}

func (r *marriageCommandTestRepo) BookWedding(context.Context, domainmarriage.BookWeddingCommand) (*domainmarriage.BookWeddingResult, error) {
	return r.bookResult, nil
}

func (r *marriageCommandTestRepo) DivorceWithCard(context.Context, int64) (*domainmarriage.DivorceResult, error) {
	return r.divorceResult, nil
}

func TestCachedMarriageRepositoryReconcilesCommittedMoney(t *testing.T) {
	data := NewPlayerData(7)
	data.SetCharacter(&domainchar.Character{ID: 7, Money: 5000000})
	data.ClearDirty()
	cache := NewPlayerCache(nil, nil, nil, nil, nil, nil, nil)
	cache.coldLoader = nil
	cache.warmLoader = nil
	cache.activeSessions[7] = data
	repo := NewCachedMarriageRepository(cache, &marriageCommandTestRepo{bookResult: &domainmarriage.BookWeddingResult{Money: 2010000}})

	if _, err := repo.BookWedding(context.Background(), domainmarriage.BookWeddingCommand{CharacterID: 7}); err != nil {
		t.Fatal(err)
	}
	if got := data.GetCharacter(); got == nil || got.Money != 2010000 || data.IsDirty() {
		t.Fatalf("character = %#v, dirty = %t", got, data.IsDirty())
	}
}

func TestCachedMarriageRepositoryReconcilesDivorceCard(t *testing.T) {
	data := NewPlayerData(7)
	data.SetCharacter(&domainchar.Character{ID: 7})
	data.AddItemWithoutDirty(&domainitem.Item{ID: 9, CharacterID: 7, StackCount: 2})
	data.ClearDirty()
	cache := NewPlayerCache(nil, nil, nil, nil, nil, nil, nil)
	cache.coldLoader = nil
	cache.warmLoader = nil
	cache.activeSessions[7] = data
	repo := NewCachedMarriageRepository(cache, &marriageCommandTestRepo{divorceResult: &domainmarriage.DivorceResult{ItemID: 9, Remaining: 1}})

	if _, err := repo.DivorceWithCard(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if got := data.GetItem(9); got == nil || got.StackCount != 1 || data.IsDirty() {
		t.Fatalf("item = %#v, dirty = %t", got, data.IsDirty())
	}
}
