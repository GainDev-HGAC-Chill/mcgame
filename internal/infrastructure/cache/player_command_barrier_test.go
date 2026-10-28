package cache

import (
	"context"
	"errors"
	"testing"

	domainachievement "mcgame-server/internal/domain/achievement"
	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
)

type renameCommandBarrierRepo struct {
	item.Repository
	entered   chan struct{}
	release   chan struct{}
	err       error
	remaining int
	called    bool
	commit    func()
}

func (r *renameCommandBarrierRepo) CommitCharacterRename(context.Context, int64, *item.Item, string, domainachievement.Fact) (int, error) {
	r.called = true
	if r.entered != nil {
		close(r.entered)
		<-r.release
	}
	if r.commit != nil {
		r.commit()
	}
	return r.remaining, r.err
}

type petCommandBarrierRepo struct {
	pet.Repository
	entered   chan struct{}
	release   chan struct{}
	err       error
	remaining int
}

func (r *petCommandBarrierRepo) wait() error {
	close(r.entered)
	<-r.release
	return r.err
}

func (r *petCommandBarrierRepo) ApplyFusion(context.Context, *pet.Pet, []int64) error {
	return r.wait()
}

func (r *petCommandBarrierRepo) ApplyIllusion(context.Context, *pet.Pet, int64, int64, int, bool) (int, error) {
	return r.remaining, r.wait()
}

func (r *petCommandBarrierRepo) ApplySkillBook(context.Context, pet.SkillBookMutation) (int, error) {
	return r.remaining, r.wait()
}

func TestCharacterRenameCommandBarrierRejectsPreCommandSnapshots(t *testing.T) {
	data := NewPlayerData(7)
	data.SetCharacter(&domainchar.Character{ID: 7, Name: "before"})
	data.AddItemWithoutDirty(&item.Item{ID: 9, CharacterID: 7, StackCount: 2})
	data.ClearDirty()
	staleCharacter := data.GetCharacter()
	staleItem := data.GetItem(9)
	staleCharacter.Name = "stale"
	staleItem.StackCount = 99

	cache := NewPlayerCache(nil, nil, nil, nil, nil, nil, nil)
	cache.coldLoader = nil
	cache.warmLoader = nil
	cache.activeSessions[7] = data
	delegate := &renameCommandBarrierRepo{entered: make(chan struct{}), release: make(chan struct{}), remaining: 1}
	repo := NewCachedItemRepository(cache, delegate)

	done := make(chan error, 1)
	go func() {
		_, err := repo.CommitCharacterRename(context.Background(), 7, data.GetItem(9), "after", domainachievement.Fact{})
		done <- err
	}()
	<-delegate.entered

	characterWrite := make(chan bool, 1)
	itemWrite := make(chan bool, 1)
	go func() { characterWrite <- data.SetCharacterAtRevision(staleCharacter) }()
	go func() { itemWrite <- data.SetItemAtRevision(staleItem) }()
	assertMutationBlocked(t, characterWrite)
	assertMutationBlocked(t, itemWrite)
	close(delegate.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if <-characterWrite || <-itemWrite {
		t.Fatal("pre-command snapshot was accepted after rename reconciliation")
	}
	if got := data.GetCharacter(); got.Name != "after" {
		t.Fatalf("character name = %q, want after", got.Name)
	}
	if got := data.GetItem(9); got == nil || got.StackCount != 1 {
		t.Fatalf("rename card = %#v", got)
	}
}

func TestPetFusionCommandBarrierRejectsPreCommandPetSnapshot(t *testing.T) {
	data := NewPlayerData(7)
	data.AddPetWithoutDirty(&pet.Pet{ID: 10, CharacterID: 7, Name: "before"})
	data.AddPetWithoutDirty(&pet.Pet{ID: 11, CharacterID: 7})
	stale := data.GetPet(10)
	stale.Name = "stale"
	updated := data.GetPet(10)
	updated.Name = "after"

	cache := NewPlayerCache(nil, nil, nil, nil, nil, nil, nil)
	cache.coldLoader = nil
	cache.warmLoader = nil
	cache.activeSessions[7] = data
	delegate := &petCommandBarrierRepo{entered: make(chan struct{}), release: make(chan struct{})}
	repo := NewCachedPetRepository(cache, delegate)

	done := make(chan error, 1)
	go func() { done <- repo.ApplyFusion(context.Background(), updated, []int64{11}) }()
	<-delegate.entered
	staleWrite := make(chan bool, 1)
	go func() { staleWrite <- data.SetPetAtRevision(stale) }()
	assertMutationBlocked(t, staleWrite)
	close(delegate.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if <-staleWrite {
		t.Fatal("pre-command Pet snapshot was accepted after fusion reconciliation")
	}
	if got := data.GetPet(10); got == nil || got.Name != "after" {
		t.Fatalf("main Pet = %#v", got)
	}
	if got := data.GetPet(11); got != nil {
		t.Fatalf("consumed Pet remained cached: %#v", got)
	}
}

func TestPetItemCommandsUseSharedMutationBarrier(t *testing.T) {
	tests := []struct {
		name string
		run  func(context.Context, *CachedPetRepository, *pet.Pet) error
	}{
		{
			name: "illusion",
			run: func(ctx context.Context, repo *CachedPetRepository, main *pet.Pet) error {
				_, err := repo.ApplyIllusion(ctx, main, 11, 9, 2, true)
				return err
			},
		},
		{
			name: "skill book",
			run: func(ctx context.Context, repo *CachedPetRepository, main *pet.Pet) error {
				_, err := repo.ApplySkillBook(ctx, pet.SkillBookMutation{Pet: main, BookItemID: 9})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := NewPlayerData(7)
			data.AddPetWithoutDirty(&pet.Pet{ID: 10, CharacterID: 7, Name: "before"})
			data.AddPetWithoutDirty(&pet.Pet{ID: 11, CharacterID: 7})
			data.AddItemWithoutDirty(&item.Item{ID: 9, CharacterID: 7, StackCount: 2})
			staleItem := data.GetItem(9)
			staleItem.StackCount = 99
			updated := data.GetPet(10)
			updated.Name = "after"

			cache := NewPlayerCache(nil, nil, nil, nil, nil, nil, nil)
			cache.coldLoader = nil
			cache.warmLoader = nil
			cache.activeSessions[7] = data
			delegate := &petCommandBarrierRepo{entered: make(chan struct{}), release: make(chan struct{}), remaining: 1}
			repo := NewCachedPetRepository(cache, delegate)

			done := make(chan error, 1)
			go func() { done <- tt.run(context.Background(), repo, updated) }()
			<-delegate.entered
			staleWrite := make(chan bool, 1)
			go func() { staleWrite <- data.SetItemAtRevision(staleItem) }()
			assertMutationBlocked(t, staleWrite)
			close(delegate.release)
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if <-staleWrite {
				t.Fatal("pre-command item snapshot was accepted after reconciliation")
			}
			if got := data.GetItem(9); got == nil || got.StackCount != 1 {
				t.Fatalf("command item = %#v", got)
			}
		})
	}
}

func TestCharacterRenameDoesNotRunAfterFlushFailure(t *testing.T) {
	cache, charRepo, itemRepo, _, _, _ := newTestPlayerCache(t)
	charRepo.characters[7] = &domainchar.Character{ID: 7, Name: "before"}
	itemRepo.items[9] = &item.Item{ID: 9, CharacterID: 7, StackCount: 2}
	data, err := cache.LoadPlayer(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	card := data.GetItem(9)
	card.StackCount = 3
	if !data.SetItemAtRevision(card) {
		t.Fatal("failed to create dirty item state")
	}
	itemRepo.updateErr = errors.New("flush failed")
	delegate := &renameCommandBarrierRepo{Repository: itemRepo, remaining: 1}
	repo := NewCachedItemRepository(cache, delegate)

	_, err = repo.CommitCharacterRename(context.Background(), 7, card, "after", domainachievement.Fact{})
	if err == nil {
		t.Fatal("expected flush failure")
	}
	if delegate.called {
		t.Fatal("rename command ran after write-behind flush failure")
	}
	if got := data.GetCharacter(); got.Name != "before" {
		t.Fatalf("character name changed to %q", got.Name)
	}
}

func TestCharacterRenameFailureAfterFlushDoesNotReconcile(t *testing.T) {
	cache, charRepo, itemRepo, _, _, _ := newTestPlayerCache(t)
	charRepo.characters[7] = &domainchar.Character{ID: 7, Name: "before"}
	itemRepo.items[9] = &item.Item{ID: 9, CharacterID: 7, StackCount: 2}
	data, err := cache.LoadPlayer(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	card := data.GetItem(9)
	card.StackCount = 3
	if !data.SetItemAtRevision(card) {
		t.Fatal("failed to create dirty item state")
	}
	commandErr := errors.New("rename failed")
	delegate := &renameCommandBarrierRepo{Repository: itemRepo, err: commandErr, remaining: 1}
	repo := NewCachedItemRepository(cache, delegate)

	_, err = repo.CommitCharacterRename(context.Background(), 7, card, "after", domainachievement.Fact{})
	if !errors.Is(err, commandErr) {
		t.Fatalf("error = %v, want command failure", err)
	}
	if !delegate.called {
		t.Fatal("rename command did not run after successful flush")
	}
	if got := data.GetCharacter(); got.Name != "before" {
		t.Fatalf("failed command reconciled character name to %q", got.Name)
	}
	if got := data.GetItem(9); got == nil || got.StackCount != 3 {
		t.Fatalf("failed command reconciled card: %#v", got)
	}
	if data.IsDirty() {
		t.Fatal("successfully flushed pre-command state remained dirty")
	}
}

func TestOfflineCharacterLoadWaitsForCommandCommit(t *testing.T) {
	cache, charRepo, itemRepo, _, _, _ := newTestPlayerCache(t)
	charRepo.characters[7] = &domainchar.Character{ID: 7, Name: "before"}
	itemRepo.items[9] = &item.Item{ID: 9, CharacterID: 7, StackCount: 2}
	delegate := &renameCommandBarrierRepo{
		Repository: itemRepo,
		entered:    make(chan struct{}),
		release:    make(chan struct{}),
		remaining:  1,
		commit: func() {
			charRepo.mu.Lock()
			charRepo.characters[7].Name = "after"
			charRepo.mu.Unlock()
			itemRepo.mu.Lock()
			itemRepo.items[9].StackCount = 1
			itemRepo.mu.Unlock()
		},
	}
	repo := NewCachedItemRepository(cache, delegate)

	commandDone := make(chan error, 1)
	go func() {
		_, err := repo.CommitCharacterRename(context.Background(), 7, itemRepo.items[9], "after", domainachievement.Fact{})
		commandDone <- err
	}()
	<-delegate.entered
	loadDone := make(chan *PlayerData, 1)
	go func() {
		data, _ := cache.LoadPlayer(context.Background(), 7)
		loadDone <- data
	}()
	select {
	case <-loadDone:
		t.Fatal("offline player loaded while command was uncommitted")
	default:
	}
	close(delegate.release)
	if err := <-commandDone; err != nil {
		t.Fatal(err)
	}
	loaded := <-loadDone
	if got := loaded.GetCharacter(); got == nil || got.Name != "after" {
		t.Fatalf("loaded character = %#v", got)
	}
	if got := loaded.GetItem(9); got == nil || got.StackCount != 1 {
		t.Fatalf("loaded rename card = %#v", got)
	}
}

func assertMutationBlocked(t *testing.T, result <-chan bool) {
	t.Helper()
	select {
	case <-result:
		t.Fatal("cached mutation crossed the command barrier")
	default:
	}
}
