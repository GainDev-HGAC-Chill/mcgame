package cache

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
)

func transitionCache() (*PlayerCache, *PlayerData) {
	data := NewPlayerData(7)
	data.Character = character.NewCharacter(uuid.New(), "target", 1, 0)
	data.Character.ID = 7
	data.Character.RaiseLevelTo(10)
	cache := &PlayerCache{activeSessions: map[int64]*PlayerData{7: data}, logger: zap.NewNop()}
	return cache, data
}

func TestConditionalTransitionNoopErrorAndDetachedEntities(t *testing.T) {
	for _, fail := range []bool{false, true} {
		cache, data := transitionCache()
		data.AddItemWithoutDirty(&item.Item{ID: 1, SlotType: item.SlotTypeEquipped, Properties: map[string]interface{}{"bonus": 12}})
		before, revision := data.Snapshot(), data.revision
		updated, changed, err := cache.UpdateCachedCharacterConditional(7, func(char *character.Character, items []*item.Item) (bool, error) {
			char.Gold = 12345
			items[0].Properties["bonus"] = 99
			if fail {
				return true, errors.New("rejected")
			}
			return false, nil
		})
		if updated == nil || changed || (fail != (err != nil)) || data.IsDirty() || data.revision != revision || !reflect.DeepEqual(before, data.Snapshot()) {
			t.Fatalf("rejected/noop transition changed cache: changed=%v err=%v revision=%d", changed, err, data.revision)
		}
	}
}

func TestConditionalTransitionCommitsOnce(t *testing.T) {
	cache, data := transitionCache()
	revision := data.revision
	updated, changed, err := cache.UpdateCachedCharacterConditional(7, func(char *character.Character, _ []*item.Item) (bool, error) { return char.SetTargetLevel(5) })
	if err != nil || !changed || updated.Level != 5 || !data.IsDirty() || data.revision != revision+1 {
		t.Fatalf("commit failed: changed=%v err=%v", changed, err)
	}
	updated.Gold = 900
	if data.GetCharacter().Gold == 900 {
		t.Fatal("returned character aliases cache")
	}
}

func TestConditionalTransitionBlocksEvictionUntilCommit(t *testing.T) {
	cache, data := transitionCache()
	repo := newMockCharacterRepo()
	cache.charRepo = repo
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_, _, err := cache.UpdateCachedCharacterConditional(7, func(char *character.Character, _ []*item.Item) (bool, error) {
			close(entered)
			<-release
			return char.SetTargetLevel(5)
		})
		finished <- err
	}()
	<-entered
	evicted := make(chan error, 1)
	go func() { evicted <- cache.SaveAndEvict(context.Background(), 7) }()
	select {
	case err := <-evicted:
		close(release)
		t.Fatalf("eviction bypassed membership lock: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if err := <-evicted; err != nil {
		t.Fatal(err)
	}
	saved, err := repo.FindByID(context.Background(), 7)
	if err != nil || saved == nil || saved.Level != 5 || cache.HasPlayer(7) || data.IsDirty() {
		t.Fatal("eviction did not persist completed transition")
	}
}

type evictionBlockingCharacterRepo struct {
	character.Repository
	entered chan struct{}
	release chan struct{}
}

func (r evictionBlockingCharacterRepo) Update(context.Context, *character.Character) error {
	close(r.entered)
	<-r.release
	return nil
}

func TestConditionalTransitionRejectsEvictedOrphanDuringSave(t *testing.T) {
	cache, data := transitionCache()
	data.MarkCharacterDirty()
	repo := evictionBlockingCharacterRepo{entered: make(chan struct{}), release: make(chan struct{})}
	cache.charRepo = repo
	finished := make(chan error, 1)
	go func() { finished <- cache.SaveAndEvict(context.Background(), 7) }()
	<-repo.entered
	called := false
	updated, changed, err := cache.UpdateCachedCharacterConditional(7, func(char *character.Character, _ []*item.Item) (bool, error) {
		called = true
		return char.SetTargetLevel(5)
	})
	close(repo.release)
	if saveErr := <-finished; saveErr != nil {
		t.Fatal(saveErr)
	}
	if called || updated != nil || changed || err != nil || data.GetCharacter().Level != 10 {
		t.Fatal("mutated evicted orphan")
	}
}

func TestConditionalTransitionUsesLockedCurrentEquipment(t *testing.T) {
	cache, data := transitionCache()
	data.AddItemWithoutDirty(&item.Item{ID: 1, SlotType: item.SlotTypeEquipped, TemplateID: 99})
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_, _, err := cache.UpdateCachedCharacterConditional(7, func(char *character.Character, items []*item.Item) (bool, error) {
			close(entered)
			<-release
			if len(items) != 1 || items[0].SlotType != item.SlotTypeEquipped || items[0].TemplateID != 99 {
				return false, errors.New("unexpected equipment")
			}
			return false, errors.New("equipped item requires level 10")
		})
		finished <- err
	}()
	<-entered
	itemChanged := make(chan struct{})
	go func() { data.SetItem(&item.Item{ID: 1, SlotType: item.SlotTypeBag}); close(itemChanged) }()
	select {
	case <-itemChanged:
		close(release)
		t.Fatal("equipment changed during locked preflight")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	if err := <-finished; err == nil {
		t.Fatal("equipment rejection lost")
	}
	<-itemChanged
	if data.GetCharacter().Level != 10 || data.dirtyChar {
		t.Fatal("rejected equipped downgrade dirtied character")
	}
}
