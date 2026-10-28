// Atomic guild command contracts return committed character and guild state.
package guild

import (
	"context"

	domainguild "mcgame-server/internal/domain/guild"
)

const GuildCreationCost int64 = 1000000

type CharacterCommitState struct {
	CharacterID        int64
	GuildID            *int64
	Money              int64
	MoneyBind          int64
	Gold               int64
	GuildContrib       int64
	DonateContrib      int64
	GuildRestoreNormal int64
	GuildRestoreDonate int64
}

type CreateGuildCommand struct {
	LeaderID int64
	Name     string
	Cost     int64
	Ranks    []domainguild.RankSetting
}

type CreateGuildResult struct {
	GuildID   int64
	GuildName string
	Leader    CharacterCommitState
}

type ContributeMoneyCommand struct {
	GuildID     int64
	CharacterID int64
	Silver      int64
	Gold        int64
	FundsGain   int64
	DonateGain  int64
}

type ContributeMoneyResult struct {
	CharacterName string
	Character     CharacterCommitState
	GuildFunds    int64
	MemberDonate  int64
	MemberTotal   int64
	MemberWeekly  int64
}

type ContributeMaterialCommand struct {
	GuildID             int64
	CharacterID         int64
	ItemID              int64
	ItemTemplateID      int
	ItemCount           int
	WarehouseTemplateID int
	WarehouseCount      int
	WarehouseMaxStack   int
	WarehousePageCount  int
	DonateGain          int64
}

type ContributeMaterialResult struct {
	CharacterName string
	Character     CharacterCommitState
	ItemID        int64
	ItemRemaining int
	WarehouseSlot *domainguild.GuildWarehouseSlot
	MemberDonate  int64
	MemberTotal   int64
	MemberWeekly  int64
}

type CommandRepository interface {
	CreateGuildAtomic(ctx context.Context, command CreateGuildCommand) (*CreateGuildResult, error)
	ContributeMoneyAtomic(ctx context.Context, command ContributeMoneyCommand) (*ContributeMoneyResult, error)
	ContributeMaterialAtomic(ctx context.Context, command ContributeMaterialCommand) (*ContributeMaterialResult, error)
}

func (s *Service) ContributeMaterialAtomic(ctx context.Context, command ContributeMaterialCommand) (*ContributeMaterialResult, error) {
	if command.GuildID <= 0 || command.CharacterID <= 0 || command.ItemID <= 0 || command.ItemTemplateID <= 0 || command.ItemCount <= 0 || command.WarehouseTemplateID <= 0 || command.WarehouseCount <= 0 || command.WarehouseMaxStack <= 0 || command.WarehouseCount > command.WarehouseMaxStack || command.WarehousePageCount < 1 || command.WarehousePageCount > 4 || command.DonateGain < 0 {
		return nil, ErrInvalidContribution
	}
	commands, ok := s.repo.(CommandRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	return commands.ContributeMaterialAtomic(ctx, command)
}

func (s *Service) CreateGuildAtomic(ctx context.Context, leaderID int64, name string) (*CreateGuildResult, error) {
	commands, ok := s.repo.(CommandRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	return commands.CreateGuildAtomic(ctx, CreateGuildCommand{LeaderID: leaderID, Name: name, Cost: GuildCreationCost, Ranks: DefaultRankSettings()})
}

func (s *Service) ContributeMoneyAtomic(ctx context.Context, guildID, characterID, silver, gold int64) (*ContributeMoneyResult, error) {
	if silver < 0 || gold < 0 || (silver == 0 && gold == 0) || gold > (1<<62)/2000 {
		return nil, ErrInsufficientFunds
	}
	commands, ok := s.repo.(CommandRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	fundsGain := silver + gold*2000
	if fundsGain < 0 {
		return nil, ErrInsufficientFunds
	}
	return commands.ContributeMoneyAtomic(ctx, ContributeMoneyCommand{
		GuildID: guildID, CharacterID: characterID, Silver: silver, Gold: gold,
		FundsGain: fundsGain, DonateGain: fundsGain / 8,
	})
}
