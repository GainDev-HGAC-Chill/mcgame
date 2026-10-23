// Open-sourced by BaoLT

package soul

import (
	"context"
	"errors"
	"sync"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/domain/stats"
)

func TestPutInToExpUsesFractionOfCurrentBoundedNeed(t *testing.T) {
	svc, chars := newTestService(t, &domainchar.Character{ID: 1, SoulExp: 300})
	if err := svc.AddSoulToSlot(context.Background(), 1, 1, 94, 100); err != nil {
		t.Fatal(err)
	}
	data, remaining, err := svc.PutInToExp(context.Background(), 1, 1, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if data["exp"] != int64(250) || remaining != 150 || chars.chars[1].SoulExp != 150 {
		t.Fatalf("data=%v remaining=%d persisted=%d", data, remaining, chars.chars[1].SoulExp)
	}
	if _, _, err := svc.PutInToExp(context.Background(), 1, 1, 1.1); !errors.Is(err, ErrInvalidFraction) {
		t.Fatalf("fraction > 1 error=%v", err)
	}
}

func TestSoulLevelUpSpendsOnlyRemainingAndRejectsLevelTen(t *testing.T) {
	svc, chars := newTestService(t, &domainchar.Character{ID: 1, SoulExp: 500})
	if err := svc.AddSoulToSlot(context.Background(), 1, 1, 94, 400); err != nil {
		t.Fatal(err)
	}
	data, remaining, err := svc.SoulLevelUp(context.Background(), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if data["sid"] != 95 || data["exp"] != int64(0) || remaining != 400 {
		t.Fatalf("data=%v remaining=%d", data, remaining)
	}
	if err := svc.AddSoulToSlot(context.Background(), 1, 2, 104, 0); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.SoulLevelUp(context.Background(), 1, 2); !errors.Is(err, ErrSoulMaxLevel) {
		t.Fatalf("level ten error=%v", err)
	}
	if chars.chars[1].SoulExp != 400 {
		t.Fatalf("level ten changed pool=%d", chars.chars[1].SoulExp)
	}
}

func TestTransformAllExpUsesColorThresholdAndSkipsLocked(t *testing.T) {
	svc, chars := newTestService(t, &domainchar.Character{ID: 1})
	ctx := context.Background()
	for _, seed := range []struct {
		slot int
		sid  int
		exp  int64
	}{{1, 11, 5}, {2, 21, 7}, {3, 11, 9}} {
		if err := svc.AddSoulToSlot(ctx, 1, seed.slot, seed.sid, seed.exp); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.LockSoul(ctx, 1, 3); err != nil {
		t.Fatal(err)
	}
	payload, err := svc.TransformAllExp(ctx, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := svc.loadCharState(ctx, 1)
	if _, ok := state.Slots[1]; ok || state.Slots[2].Sid != 21 || !state.Slots[3].Lock {
		t.Fatalf("remaining slots=%+v", state.Slots)
	}
	if chars.chars[1].SoulExp != 15 || chars.chars[1].SoulPoints != 1 {
		t.Fatalf("currencies=%d/%d payload=%v", chars.chars[1].SoulExp, chars.chars[1].SoulPoints, payload)
	}
}

func TestTransformTempExpSingleAndThreshold(t *testing.T) {
	svc, chars := newTestService(t, &domainchar.Character{ID: 1})
	ctx := context.Background()
	state := defaultCharSoulState()
	state.TempBag[1] = 11
	state.TempBag[2] = 21
	state.TempBag[3] = 11
	if err := svc.saveCharState(ctx, 1, state); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.TransformTempExp(ctx, 1, 1); err != nil {
		t.Fatal(err)
	}
	payload, err := svc.TransformAllTempExp(ctx, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	state, _ = svc.loadCharState(ctx, 1)
	if len(state.TempBag) != 1 || state.TempBag[2] != 21 {
		t.Fatalf("temp bag=%+v payload=%v", state.TempBag, payload)
	}
	if chars.chars[1].SoulExp != 20 || chars.chars[1].SoulPoints != 2 {
		t.Fatalf("currencies=%d/%d", chars.chars[1].SoulExp, chars.chars[1].SoulPoints)
	}
}

func TestSoulLevelUpPetRejectsForeignOwner(t *testing.T) {
	svc, chars := newTestService(t, &domainchar.Character{ID: 1, SoulExp: 500})
	svc.SetPetRepository(&fakePetRepo{pets: map[int64]*domainpet.Pet{77: {ID: 77, CharacterID: 2, Level: 80}}})
	state := defaultPetSoulState()
	state.Slots[1] = SoulSlot{Sid: 94}
	if err := svc.savePetState(context.Background(), 77, state); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.SoulLevelUpPet(context.Background(), 1, 77, 1); !errors.Is(err, ErrPetNotOwned) {
		t.Fatalf("foreign owner error=%v", err)
	}
	if chars.chars[1].SoulExp != 500 {
		t.Fatalf("foreign mutation spent=%d", chars.chars[1].SoulExp)
	}
}

func TestAggregatePetBonusesPreservesIntegralAndFractionalProperties(t *testing.T) {
	svc, _ := newTestService(t, &domainchar.Character{ID: 1})
	pet := &domainpet.Pet{ID: 77, CharacterID: 1, Level: 80, Property: map[string]interface{}{}}
	svc.SetPetRepository(&fakePetRepo{pets: map[int64]*domainpet.Pet{77: pet}})
	state := defaultPetSoulState()
	state.Slots[1] = SoulSlot{Sid: 11}
	state.Slots[2] = SoulSlot{Sid: 21}
	if err := svc.savePetState(context.Background(), 77, state); err != nil {
		t.Fatal(err)
	}
	equipment, _ := svc.AggregatePetBonuses(context.Background(), 77)
	if equipment.Flat[1] != 100 || equipment.Feature[32] != 1.5 {
		t.Fatalf("bonuses=%+v", equipment)
	}
	pet.SetEquipmentBonuses(equipment)
	pet.RecalculateStats()
	if pet.Property["soulProp32"] != 1.5 || pet.Property["finalDebuffSuccRate"] != 1.5 {
		t.Fatalf("pet properties=%+v", pet.Property)
	}
	effective := pet.Effective()
	if got := effective.Get(stats.PropDebuffHit); got != 15 {
		t.Fatalf("effective debuff hit=%d want 15", got)
	}
}

func TestSoulExpMutationsSerializePerCharacter(t *testing.T) {
	svc, chars := newTestService(t, &domainchar.Character{ID: 1, SoulExp: 600})
	if err := svc.AddSoulToSlot(context.Background(), 1, 1, 94, 0); err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _, err := svc.PutInToExp(context.Background(), 1, 1, 0.5)
			errs <- err
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	state, err := svc.loadCharState(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if state.Slots[1].Exp != 375 || chars.chars[1].SoulExp != 225 {
		t.Fatalf("serialized exp=%d pool=%d", state.Slots[1].Exp, chars.chars[1].SoulExp)
	}
}

func TestMoveSoulRejectsForeignPet(t *testing.T) {
	svc, _ := newTestService(t, &domainchar.Character{ID: 1})
	svc.SetPetRepository(&fakePetRepo{pets: map[int64]*domainpet.Pet{77: {ID: 77, CharacterID: 2, Level: 80}}})
	payload, err := svc.MoveSoul(context.Background(), 1, 77, 101, 102)
	if err != nil || payload["f"] != 0 {
		t.Fatalf("payload=%v error=%v", payload, err)
	}
}
