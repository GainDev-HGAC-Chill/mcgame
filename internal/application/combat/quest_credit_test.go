package combat

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	domaincombat "mcgame-server/internal/domain/combat"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

func questCreditBattle(characterIDs ...int64) *domaincombat.Battle {
	battle := domaincombat.NewBattle(domaincombat.BattleTypePVE, 3)
	for _, id := range characterIDs {
		participant := domaincombat.NewParticipant(strconv.FormatInt(id, 10), "player", domaincombat.SidePlayer, false)
		participant.EntityID = id
		participant.EntityType = domaincombat.ParticipantTypeCharacter
		battle.AddParticipant(participant)
	}
	battle.End(&domaincombat.BattleResult{WinnerSide: domaincombat.SidePlayer})
	return battle
}

func TestClaimQuestKillCredit_ConcurrentOncePerPartyCharacter(t *testing.T) {
	service := NewService(newTestBattleRepo(), nil, nil, nil, nil, nil, zap.NewNop())
	battle := questCreditBattle(1, 2)
	service.cacheEndedBattle(battle)
	var claims atomic.Int64
	var workers sync.WaitGroup
	for range 100 {
		workers.Go(func() {
			if claimedBattle, claimed, err := service.ClaimQuestKillCredit(battle.ID, 1); claimed {
				if err != nil {
					t.Error(err)
				}
				if claimedBattle != battle {
					t.Error("claim did not return authoritative cached battle")
				}
				claims.Add(1)
			}
		})
	}
	workers.Wait()
	if got := claims.Load(); got != 1 {
		t.Fatalf("claims = %d, want 1", got)
	}
	service.cacheEndedBattle(battle)
	if _, claimed, err := service.ClaimQuestKillCredit(battle.ID, 1); claimed || !errors.Is(err, ErrQuestKillCreditInflight) {
		t.Fatal("recaching reset the claim")
	}
	service.AckBattleCleanup(battle.ID, 1)
	if _, pending := service.endedBattles[battle.ID].pendingCleanups["1"]; !pending {
		t.Fatal("inflight cleanup was acknowledged")
	}
	service.CompleteQuestKillCredit(battle.ID, 1)
	service.AckBattleCleanup(battle.ID, 1)
	if _, err := service.GetBattleForCleanup(battle.ID); err != nil {
		t.Fatalf("party cleanup unavailable after first ack: %v", err)
	}
	if _, claimed, err := service.ClaimQuestKillCredit(battle.ID, 1); claimed || err != nil {
		t.Fatal("acknowledged character received credit")
	}
	if _, claimed, err := service.ClaimQuestKillCredit(battle.ID, 2); !claimed || err != nil {
		t.Fatal("second party member could not claim")
	}
	service.CompleteQuestKillCredit(battle.ID, 2)
	service.AckBattleCleanup(battle.ID, 2)
	if len(service.endedBattles) != 0 {
		t.Fatal("cleanup retained battle/claim state")
	}
	if _, claimed, err := service.ClaimQuestKillCredit(battle.ID, 2); claimed || err != nil {
		t.Fatal("removed battle received credit")
	}
}

func TestClaimQuestKillCredit_RejectsInvalidBattlesAndMembers(t *testing.T) {
	for _, name := range []string{"foreign", "pet", "enemy", "wrong_entity", "defeat", "missing_result", "active", "uncached", "already_acked"} {
		t.Run(name, func(t *testing.T) {
			service := NewService(newTestBattleRepo(), nil, nil, nil, nil, nil, zap.NewNop())
			battle := questCreditBattle(1, 2)
			characterID := int64(1)
			switch name {
			case "foreign":
				characterID = 99
			case "pet":
				battle.GetParticipant("1").EntityType = domaincombat.ParticipantTypePet
			case "enemy":
				battle.GetParticipant("1").Side = domaincombat.SideEnemy
			case "wrong_entity":
				battle.GetParticipant("1").EntityID = 99
			case "defeat":
				battle.Result.WinnerSide = domaincombat.SideEnemy
			case "missing_result":
				battle.Result = nil
			case "active":
				battle.Start()
			}
			if name != "uncached" {
				service.cacheEndedBattle(battle)
			}
			if name == "already_acked" {
				service.AckBattleCleanup(battle.ID, 1)
			}
			if claimedBattle, claimed, err := service.ClaimQuestKillCredit(battle.ID, characterID); claimed || claimedBattle != nil || err != nil {
				t.Fatal("invalid credit claim accepted")
			}
		})
	}
}

func TestQuestKillCredit_ReleaseAllowsRetryButNotCompletedReplay(t *testing.T) {
	service := NewService(newTestBattleRepo(), nil, nil, nil, nil, nil, zap.NewNop())
	battle := questCreditBattle(1)
	service.cacheEndedBattle(battle)
	if _, claimed, err := service.ClaimQuestKillCredit(battle.ID, 1); !claimed || err != nil {
		t.Fatalf("initial claim = %v, %v", claimed, err)
	}
	service.AckBattleCleanup(battle.ID, 1)
	service.ReleaseQuestKillCredit(battle.ID, 1)
	if _, claimed, err := service.ClaimQuestKillCredit(battle.ID, 1); !claimed || err != nil {
		t.Fatalf("retry claim = %v, %v", claimed, err)
	}
	service.CompleteQuestKillCredit(battle.ID, 1)
	service.ReleaseQuestKillCredit(battle.ID, 1)
	if _, claimed, err := service.ClaimQuestKillCredit(battle.ID, 1); claimed || err != nil {
		t.Fatalf("completed replay = %v, %v", claimed, err)
	}
	service.AckBattleCleanup(battle.ID, 1)
	if len(service.endedBattles) != 0 {
		t.Fatal("completed cleanup retained claim state")
	}
}

func TestGetCharacterBattleForCleanup_ValidatesMembership(t *testing.T) {
	repo := newTestBattleRepo()
	service := NewService(repo, nil, nil, nil, nil, nil, zap.NewNop())
	battle := questCreditBattle(1)
	if err := repo.Store(battle); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetCharacterBattleForCleanup(battle.ID, 99); !errors.Is(err, pkgerrors.ErrUnauthorized) {
		t.Fatalf("foreign character error = %v, want unauthorized", err)
	}
	if got, err := service.GetCharacterBattleForCleanup(battle.ID, 1); err != nil || got != battle {
		t.Fatalf("member cleanup = %v, %v", got, err)
	}
	service.cacheEndedBattle(battle)
	if err := repo.Remove(battle.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := service.GetCharacterBattleForCleanup(battle.ID, 1); err != nil || got != battle {
		t.Fatalf("cached member cleanup = %v, %v", got, err)
	}
}
