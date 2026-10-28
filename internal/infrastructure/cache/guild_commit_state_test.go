package cache

import (
	"context"
	"errors"
	"testing"

	appguild "mcgame-server/internal/application/guild"
	domainchar "mcgame-server/internal/domain/character"
)

func TestSyncCommittedGuildStateDoesNotCreateDirtyWriteBehind(t *testing.T) {
	data := NewPlayerData(7)
	data.SetCharacter(&domainchar.Character{ID: 7, Money: 100, Gold: 20})
	data.ClearDirty()
	guildID := int64(3)
	data.SyncCommittedGuildState(&guildID, 50, 40, 10, 8, 9, 0, 0)
	character := data.GetCharacter()
	if character.GuildID == nil || *character.GuildID != guildID || character.Money != 50 || character.MoneyBind != 40 || character.Gold != 10 || character.GuildContrib != 8 || character.DonateContrib != 9 {
		t.Fatalf("character = %#v", character)
	}
	if data.IsDirty() {
		t.Fatal("committed state must not become write-behind dirty")
	}
}

func TestGuildCommandsRemainClosedWithoutSharedMutationBarrier(t *testing.T) {
	repo := NewCachedGuildRepository(nil, nil)
	if _, err := repo.CreateGuildAtomic(context.Background(), appguild.CreateGuildCommand{}); !errors.Is(err, appguild.ErrUnsupportedAction) {
		t.Fatalf("create error = %v", err)
	}
	if _, err := repo.ContributeMoneyAtomic(context.Background(), appguild.ContributeMoneyCommand{}); !errors.Is(err, appguild.ErrUnsupportedAction) {
		t.Fatalf("money contribution error = %v", err)
	}
	if _, err := repo.ContributeMaterialAtomic(context.Background(), appguild.ContributeMaterialCommand{}); !errors.Is(err, appguild.ErrUnsupportedAction) {
		t.Fatalf("material contribution error = %v", err)
	}
}

func TestSyncCommittedGuildStatePreservesConcurrentDirtyCharacter(t *testing.T) {
	data := NewPlayerData(7)
	data.SetCharacter(&domainchar.Character{ID: 7, Name: "updated", Money: 100})
	guildID := int64(3)
	data.SyncCommittedGuildState(&guildID, 50, 0, 0, 0, 0, 0, 0)
	if !data.IsDirty() {
		t.Fatal("concurrent dirty state must remain dirty")
	}
	character := data.GetCharacter()
	if character.Name != "updated" || character.Money != 50 {
		t.Fatalf("character = %#v", character)
	}
}
