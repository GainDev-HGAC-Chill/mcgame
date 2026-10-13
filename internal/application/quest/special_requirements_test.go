package quest

import (
	"context"
	"encoding/json"
	"errors"
	domainchar "mcgame-server/internal/domain/character"
	"testing"

	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/domain/social"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type questFriendStub struct {
	friends []*social.Relationship
	err     error
}

func (r *questFriendStub) FindFriends(context.Context, int64) ([]*social.Relationship, error) {
	return r.friends, r.err
}

func specialQuestService(t *testing.T, id int, name string, kind int) *Service {
	t.Helper()
	svc := NewService(newQuestTestQuestRepo(), zap.NewNop())
	manager := gamedata.NewManager(nil, zap.NewNop())
	if err := manager.GetCache().LoadTable(models.TableQuest, []json.RawMessage{mustQuestJSON(t, models.QuestTemplate{ID: int64(id), Name: name, Type: float64(kind)})}); err != nil {
		t.Fatal(err)
	}
	svc.SetGameDataManager(manager)
	return svc
}

func TestFriendQuestUsesRealRelationshipsForUIAndRewardGate(t *testing.T) {
	friend := func(other int64, intimacy int) *social.Relationship {
		return &social.Relationship{CharacterID: 2, OtherID: other, Type: social.RelationshipTypeFriend, Intimacy: intimacy}
	}
	for _, tc := range []struct {
		name     string
		friends  []*social.Relationship
		complete bool
	}{
		{"empty", nil, false},
		{"too few", []*social.Relationship{friend(3, 50)}, false},
		{"intimacy below threshold", []*social.Relationship{friend(3, 50), friend(4, 49)}, false},
		{"exact threshold", []*social.Relationship{friend(3, 50), friend(4, 50)}, true},
		{"extra low intimacy", []*social.Relationship{friend(3, 50), friend(4, 60), friend(5, 0)}, false},
		{"duplicate", []*social.Relationship{friend(3, 50), friend(3, 50)}, false},
		{"self", []*social.Relationship{friend(2, 50), friend(3, 50)}, false},
		{"blacklist is not a friend", []*social.Relationship{friend(3, 50), {CharacterID: 2, OtherID: 4, Type: social.RelationshipTypeBlack, Intimacy: 50}}, false},
		{"foreign owner", []*social.Relationship{friend(3, 50), {CharacterID: 9, OtherID: 4, Intimacy: 50}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := specialQuestService(t, 2696, "Kết giao 2 người bạn", 10)
			svc.SetFriendRepository(&questFriendStub{friends: tc.friends})
			qp := domainquest.NewQuestProgress(2, 2696, nil)
			if err := svc.questRepo.Save(context.Background(), qp); err != nil {
				t.Fatal(err)
			}
			active, err := svc.GetActiveQuests(context.Background(), 2)
			if err != nil {
				t.Fatal(err)
			}
			if len(active) != 1 || active[0].CanComplete() != tc.complete {
				t.Fatalf("incorrect completion: %#v", active)
			}
			dto := svc.EnrichQuestDTO(context.Background(), active[0])
			want := 103
			if tc.complete {
				want = 104
			}
			if dto["state"] != want {
				t.Fatalf("state=%v want=%v", dto["state"], want)
			}
			if !tc.complete {
				if _, _, err := svc.CompleteQuest(context.Background(), 2, 2696); !errors.Is(err, pkgerrors.ErrInvalidInput) {
					t.Fatalf("reward gate err=%v", err)
				}
			}
		})
	}
}

func TestSpecialRequirementsFailClosedWithoutResettingHistory(t *testing.T) {
	svc := specialQuestService(t, 2697, "Kết giao 3 người bạn", 10)
	qp := domainquest.NewQuestProgress(2, 2697, nil)
	if dto := svc.EnrichQuestDTO(context.Background(), qp); dto["state"] != 103 {
		t.Fatal("missing repository reported complete")
	}
	svc.SetFriendRepository(&questFriendStub{err: errors.New("unavailable")})
	if err := svc.syncSpecialRequirements(context.Background(), qp); err == nil || qp.CanComplete() {
		t.Fatal("repository failure did not fail closed")
	}
	qp.Status = domainquest.QuestStatusCompleted
	qp.CompletionBlocked = false
	if err := svc.syncSpecialRequirements(context.Background(), qp); err != nil || qp.CompletionBlocked {
		t.Fatal("rewrote completed quest")
	}
	unknown := specialQuestService(t, 2670, "Tăng số lượng pet lên 9", 10)
	if dto := unknown.EnrichQuestDTO(context.Background(), domainquest.NewQuestProgress(2, 2670, nil)); dto["state"] != 103 {
		t.Fatal("unsupported empty growup quest reported complete")
	}
	talk := specialQuestService(t, 1, "Bắt Đầu Thực Tập", 1)
	if err := talk.gameDataManager.GetCache().LoadTable(models.TableQuest, []json.RawMessage{mustQuestJSON(t, models.QuestTemplate{ID: 1, Name: "Bắt Đầu Thực Tập", Type: 1, FinishNPC: 231})}); err != nil {
		t.Fatal(err)
	}
	if err := talk.gameDataManager.GetCache().LoadTable(models.TableNpc, []json.RawMessage{mustQuestJSON(t, models.NpcTemplate{ID: 231, PosMapID: 1, PosX: 100, PosY: 100})}); err != nil {
		t.Fatal(err)
	}
	talk.SetCharacterRepository(&questTestCharacterRepo{char: &domainchar.Character{ID: 2, MapID: 1, PosX: 100, PosY: 100}})
	if dto := talk.EnrichQuestDTO(context.Background(), domainquest.NewQuestProgress(2, 1, nil)); dto["state"] != 104 {
		t.Fatal("broke objective-free NPC conversation")
	}
}

func TestFriendQuestRechecksDeletionAndDoesNotPersistVerification(t *testing.T) {
	svc := specialQuestService(t, 2695, "Kết giao 1 người bạn", 10)
	repo := &questFriendStub{friends: []*social.Relationship{{CharacterID: 2, OtherID: 3, Intimacy: 50}}}
	svc.SetFriendRepository(repo)
	qp := domainquest.NewQuestProgress(2, 2695, nil)
	if dto := svc.EnrichQuestDTO(context.Background(), qp); dto["state"] != 104 {
		t.Fatal("qualifying friend did not complete requirement")
	}
	repo.friends = nil
	if dto := svc.EnrichQuestDTO(context.Background(), qp); dto["state"] != 103 || qp.CanComplete() {
		t.Fatal("deleted friend retained stale completion")
	}
	encoded, err := json.Marshal(qp)
	if err != nil {
		t.Fatal(err)
	}
	var stored map[string]interface{}
	if err := json.Unmarshal(encoded, &stored); err != nil {
		t.Fatal(err)
	}
	if _, exists := stored["CompletionBlocked"]; exists {
		t.Fatal("persisted runtime verification flag")
	}
}
