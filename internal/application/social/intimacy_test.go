package social

import (
	"context"
	"testing"

	"go.uber.org/zap"
	domainsocial "mcgame-server/internal/domain/social"
)

type intimacyTestRepo struct {
	relationships map[int64][]*domainsocial.Relationship
	updates       map[int64]int
}

func (r *intimacyTestRepo) FindByCharacterID(context.Context, int64) ([]*domainsocial.Relationship, error) {
	return nil, nil
}

func (r *intimacyTestRepo) FindByCharacterAndType(context.Context, int64, int) ([]*domainsocial.Relationship, error) {
	return nil, nil
}

func (r *intimacyTestRepo) FindFriends(_ context.Context, characterID int64) ([]*domainsocial.Relationship, error) {
	return r.relationships[characterID], nil
}

func (r *intimacyTestRepo) FindBlacklist(context.Context, int64) ([]*domainsocial.Relationship, error) {
	return nil, nil
}

func (r *intimacyTestRepo) FindByID(context.Context, int64) (*domainsocial.Relationship, error) {
	return nil, nil
}

func (r *intimacyTestRepo) Create(context.Context, *domainsocial.Relationship) error { return nil }

func (r *intimacyTestRepo) Delete(context.Context, int64, int64, int) error { return nil }

func (r *intimacyTestRepo) DeleteByID(context.Context, int64, int64, int) error { return nil }

func (r *intimacyTestRepo) Exists(context.Context, int64, int64, int) (bool, error) {
	return false, nil
}

func (r *intimacyTestRepo) UpdateIntimacy(_ context.Context, id int64, intimacy int) error {
	r.updates[id] = intimacy
	return nil
}

func TestAddFriendIntimacyUpdatesBothDirections(t *testing.T) {
	repo := &intimacyTestRepo{
		relationships: map[int64][]*domainsocial.Relationship{
			1: {{ID: 11, CharacterID: 1, OtherID: 2, Type: domainsocial.RelationshipTypeFriend, Intimacy: 10}},
			2: {{ID: 22, CharacterID: 2, OtherID: 1, Type: domainsocial.RelationshipTypeFriend, Intimacy: 20}},
		},
		updates: map[int64]int{},
	}
	service := NewService(repo, nil, zap.NewNop())

	result, err := service.AddFriendIntimacy(context.Background(), 1, 2, 660)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceTotal != 670 || result.TargetTotal != 680 || !result.TargetSaved {
		t.Fatalf("result = %+v", result)
	}
	if repo.updates[11] != 670 || repo.updates[22] != 680 {
		t.Fatalf("updates = %#v", repo.updates)
	}
}

func TestPreviewFriendIntimacyRejectsNonFriend(t *testing.T) {
	repo := &intimacyTestRepo{relationships: map[int64][]*domainsocial.Relationship{}, updates: map[int64]int{}}
	service := NewService(repo, nil, zap.NewNop())

	if _, err := service.PreviewFriendIntimacy(context.Background(), 1, 2, 60); err == nil {
		t.Fatal("expected non-friend rejection")
	}
	if len(repo.updates) != 0 {
		t.Fatalf("updates = %#v", repo.updates)
	}
}
