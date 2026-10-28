package cache

import (
	"context"
	"testing"

	appguild "mcgame-server/internal/application/guild"
	domainchar "mcgame-server/internal/domain/character"
)

type guildCommandBarrierFixture struct {
	appguild.Repository
	entered chan struct{}
	release chan struct{}
}

func (r *guildCommandBarrierFixture) CreateGuildAtomic(context.Context, appguild.CreateGuildCommand) (*appguild.CreateGuildResult, error) {
	return nil, appguild.ErrUnsupportedAction
}

func (r *guildCommandBarrierFixture) ContributeMoneyAtomic(_ context.Context, command appguild.ContributeMoneyCommand) (*appguild.ContributeMoneyResult, error) {
	close(r.entered)
	<-r.release
	guildID := command.GuildID
	return &appguild.ContributeMoneyResult{Character: appguild.CharacterCommitState{CharacterID: command.CharacterID, GuildID: &guildID, Money: 70, Gold: 4, GuildContrib: 9, DonateContrib: 10}}, nil
}

func (r *guildCommandBarrierFixture) ContributeMaterialAtomic(context.Context, appguild.ContributeMaterialCommand) (*appguild.ContributeMaterialResult, error) {
	return nil, appguild.ErrUnsupportedAction
}

func TestGuildCommandBarrierRejectsPreCommandCharacterSnapshot(t *testing.T) {
	data := NewPlayerData(7)
	data.SetCharacter(&domainchar.Character{ID: 7, Money: 100, Gold: 5})
	data.ClearDirty()
	stale := data.GetCharacter()
	stale.Name = "stale-write"

	cache := NewPlayerCache(nil, nil, nil, nil, nil, nil, nil)
	cache.activeSessions[7] = data
	commands := &guildCommandBarrierFixture{entered: make(chan struct{}), release: make(chan struct{})}
	repo := NewCachedGuildRepository(cache, commands)

	commandDone := make(chan error, 1)
	go func() {
		_, err := repo.ContributeMoneyAtomic(context.Background(), appguild.ContributeMoneyCommand{GuildID: 3, CharacterID: 7})
		commandDone <- err
	}()
	<-commands.entered

	staleResult := make(chan bool, 1)
	go func() {
		staleResult <- data.SetCharacterAtRevision(stale)
	}()
	select {
	case <-staleResult:
		t.Fatal("cached mutation crossed the command barrier")
	default:
	}
	close(commands.release)
	if err := <-commandDone; err != nil {
		t.Fatal(err)
	}
	if <-staleResult {
		t.Fatal("pre-command character snapshot was accepted after reconciliation")
	}
	committed := data.GetCharacter()
	if committed.Name == "stale-write" || committed.Money != 70 || committed.Gold != 4 || committed.GuildContrib != 9 || committed.DonateContrib != 10 {
		t.Fatalf("committed character = %#v", committed)
	}
	if data.IsDirty() {
		t.Fatal("committed command state became write-behind dirty")
	}
}
