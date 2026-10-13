package auth

import (
	"context"
	"testing"
	"time"

	appchar "mcgame-server/internal/application/character"
	domainauth "mcgame-server/internal/domain/auth"
	domainchar "mcgame-server/internal/domain/character"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type testDeletionStore struct {
	scheduledID   int64
	deadline      time.Time
	scheduleCalls int
	restoreCalls  int
	pending       bool
	pendingCalls  int
}

func (s *testDeletionStore) ScheduleDeletion(_ context.Context, _ uuid.UUID, id int64, at time.Time) (bool, error) {
	s.scheduleCalls++
	if s.pending {
		return false, nil
	}
	s.scheduledID, s.deadline, s.pending = id, at, true
	return true, nil
}
func (s *testDeletionStore) RestoreDeletion(_ context.Context, _ uuid.UUID, id int64) (bool, error) {
	s.restoreCalls++
	if id != s.scheduledID || !s.pending {
		return false, nil
	}
	s.pending = false
	return true, nil
}
func (s *testDeletionStore) CharacterDeletionTimes(_ context.Context, _ uuid.UUID) (map[int64]int64, error) {
	if !s.pending {
		return map[int64]int64{}, nil
	}
	return map[int64]int64{s.scheduledID: s.deadline.Unix()}, nil
}
func (s *testDeletionStore) IsDeletionPending(_ context.Context, _ uuid.UUID, id int64) (bool, error) {
	s.pendingCalls++
	return id == s.scheduledID && s.pending, nil
}
func (s *testDeletionStore) PurgeOneExpired(_ context.Context, _ uuid.UUID, _ int64) (bool, error) {
	return false, nil
}

func deletionHandlerFixture() (*Handler, *testDeletionStore, *domainchar.Character) {
	owner := uuid.New()
	char := &domainchar.Character{ID: 101, AccountID: owner, Name: "SafeTest", ClassID: 1, Gender: 0, Level: 1}
	repo := &authTestCharacterRepo{char: char}
	account := &domainauth.Account{ID: owner, SecondaryPassword: domainauth.DefaultSecondaryPasswordMD5}
	h := NewHandler(nil, appchar.NewService(repo, zap.NewNop()), nil, nil, nil, nil, &authTestAccountRepo{account: account}, zap.NewNop())
	store := &testDeletionStore{}
	h.SetCharacterDeletionStore(store)
	return h, store, char
}

func TestValidDeletionMD5(t *testing.T) {
	if !validDeletionMD5(domainauth.DefaultSecondaryPasswordMD5) {
		t.Fatal("default MD5 rejected")
	}
	if validDeletionMD5("123456") {
		t.Fatal("plaintext must be rejected")
	}
	if validDeletionMD5("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz") {
		t.Fatal("non hex must be rejected")
	}
}

func TestStartDeleteCharSecondaryPasswordAndRestore(t *testing.T) {
	h, store, char := deletionHandlerFixture()
	ctx, _ := characterListContext("scene/")
	ctx.AccountID = char.AccountID.String()
	if _, err := h.StartDeleteChar(ctx, []interface{}{float64(char.ID), "00000000000000000000000000000000"}); err != nil {
		t.Fatalf("wrong password result: %v", err)
	}
	if store.scheduleCalls != 0 {
		t.Fatal("wrong password scheduled deletion")
	}
	before := time.Now()
	if _, err := h.StartDeleteChar(ctx, []interface{}{float64(char.ID), domainauth.DefaultSecondaryPasswordMD5}); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if store.scheduleCalls != 1 || store.scheduledID != char.ID || !store.pending {
		t.Fatal("deletion was not scheduled")
	}
	if store.deadline.Before(before.Add(characterDeletionDelay - time.Minute)) {
		t.Fatal("deletion deadline too early")
	}
	originalDeadline := store.deadline
	if _, err := h.StartDeleteChar(ctx, []interface{}{float64(char.ID), domainauth.DefaultSecondaryPasswordMD5}); err != nil {
		t.Fatalf("repeat request: %v", err)
	}
	if !store.deadline.Equal(originalDeadline) {
		t.Fatal("replay extended deletion deadline")
	}
	if _, err := h.RestoreChar(ctx, []interface{}{float64(0), float64(char.ID)}); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if store.pending || store.restoreCalls != 1 {
		t.Fatal("restore failed")
	}
}

func TestStartDeleteCharRejectsOtherAccount(t *testing.T) {
	h, store, _ := deletionHandlerFixture()
	ctx, _ := characterListContext("scene/")
	ctx.AccountID = uuid.NewString()
	if _, err := h.StartDeleteChar(ctx, []interface{}{float64(101), domainauth.DefaultSecondaryPasswordMD5}); err != nil {
		t.Fatalf("RPC response: %v", err)
	}
	if store.scheduleCalls != 0 {
		t.Fatal("foreign account scheduled deletion")
	}
}

func TestChooseCharacterPendingDeletionNeverEntersGame(t *testing.T) {
	h, store, char := deletionHandlerFixture()
	store.pending = true
	store.scheduledID = char.ID
	ctx, sender := characterListContext("scene/")
	ctx.AccountID = char.AccountID.String()
	if _, err := h.ChooseCharacter(ctx, []interface{}{float64(char.ID)}); err != nil {
		t.Fatalf("RPC response: %v", err)
	}
	if store.pendingCalls != 1 {
		t.Fatal("pending deletion not checked on login")
	}
	for _, m := range sender.methods {
		if m == "onChooseCharactor" {
			t.Fatal("pending deletion entered game")
		}
	}
}
