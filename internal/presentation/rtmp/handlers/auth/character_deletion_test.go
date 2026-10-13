package auth

import (
	"context"
	"testing"

	appchar "mcgame-server/internal/application/character"
	domainauth "mcgame-server/internal/domain/auth"
	domainchar "mcgame-server/internal/domain/character"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type testDeletionStore struct {
	deleteCalls int
	deletedID   int64
	accountID   uuid.UUID
	deleted     bool
}

func (s *testDeletionStore) DeleteNow(_ context.Context, accountID uuid.UUID, id int64) (bool, error) {
	s.deleteCalls++
	s.accountID = accountID
	s.deletedID = id
	if s.deleted {
		return false, nil
	}
	s.deleted = true
	return true, nil
}

type testDeletionCache struct {
	calls int
	id    int64
}

func (s *testDeletionCache) InvalidateCharacter(_ context.Context, id int64) error {
	s.calls++
	s.id = id
	return nil
}

func deletionHandlerFixture() (*Handler, *testDeletionStore, *domainchar.Character) {
	owner := uuid.New()
	char := &domainchar.Character{ID: 101, AccountID: owner, Name: "DeleteTest", ClassID: 1, Gender: 0, Level: 1}
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
	if validDeletionMD5("123456") || validDeletionMD5("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz") {
		t.Fatal("invalid password hash accepted")
	}
}

func TestStartDeleteCharDeletesImmediatelyAndInvalidatesCache(t *testing.T) {
	h, store, char := deletionHandlerFixture()
	cache := &testDeletionCache{}
	h.SetCharacterDeletionCache(cache)
	ctx, _ := characterListContext("scene/")
	ctx.AccountID = char.AccountID.String()

	if _, err := h.StartDeleteChar(ctx, []interface{}{float64(char.ID), "00000000000000000000000000000000"}); err != nil {
		t.Fatalf("wrong password RPC: %v", err)
	}
	if store.deleteCalls != 0 || cache.calls != 0 {
		t.Fatal("wrong password deleted a character")
	}

	if _, err := h.StartDeleteChar(ctx, []interface{}{float64(char.ID), domainauth.DefaultSecondaryPasswordMD5}); err != nil {
		t.Fatalf("correct password RPC: %v", err)
	}
	if store.deleteCalls != 1 || !store.deleted || store.deletedID != char.ID || store.accountID != char.AccountID {
		t.Fatal("immediate delete was not called with verified owner")
	}
	if cache.calls != 1 || cache.id != char.ID {
		t.Fatal("deleted character cache was not invalidated")
	}

	// Replaying a request must not delete again or invalidate anything twice.
	if _, err := h.StartDeleteChar(ctx, []interface{}{float64(char.ID), domainauth.DefaultSecondaryPasswordMD5}); err != nil {
		t.Fatalf("replayed request RPC: %v", err)
	}
	if store.deleteCalls != 2 || cache.calls != 1 {
		t.Fatal("replayed delete mutated cache")
	}
}

func TestStartDeleteCharRejectsOtherAccount(t *testing.T) {
	h, store, _ := deletionHandlerFixture()
	ctx, _ := characterListContext("scene/")
	ctx.AccountID = uuid.NewString()
	if _, err := h.StartDeleteChar(ctx, []interface{}{float64(101), domainauth.DefaultSecondaryPasswordMD5}); err != nil {
		t.Fatalf("RPC response: %v", err)
	}
	if store.deleteCalls != 0 {
		t.Fatal("foreign account deleted a character")
	}
}
