// Atomic warehouse commands carry expected snapshots to detect concurrent changes.
package guild

import (
	"context"

	pkgerrors "mcgame-server/pkg/errors"
)

type WarehouseMoveCommand struct {
	GuildID     int64
	ActorID     int64
	SourceID    int64
	SourceIndex int
	TargetIndex int
	TemplateID  int
	MaxStack    int
	PageCount   int
}

type WarehouseMoveResult struct {
	SourceID    int64
	TargetID    int64
	SourceCount int
	TargetCount int
	Mode        string
}

type WarehouseDropCommand struct {
	GuildID int64
	ActorID int64
	SlotID  int64
}

type WarehouseDropResult struct {
	TemplateID int
	StackCount int
}

type WarehouseCommandRepository interface {
	MoveGuildWarehouseAtomic(context.Context, WarehouseMoveCommand) (*WarehouseMoveResult, error)
	DropGuildWarehouseAtomic(context.Context, WarehouseDropCommand) (*WarehouseDropResult, error)
}

func (s *Service) MoveGuildWarehouseAtomic(ctx context.Context, command WarehouseMoveCommand) (*WarehouseMoveResult, error) {
	if command.GuildID <= 0 || command.ActorID <= 0 || command.SourceID <= 0 || command.TemplateID <= 0 || command.MaxStack <= 0 || command.PageCount < 1 || command.PageCount > 4 || command.SourceIndex < 0 || command.TargetIndex < 0 || command.SourceIndex >= command.PageCount*30 || command.TargetIndex >= command.PageCount*30 || command.SourceIndex == command.TargetIndex {
		return nil, pkgerrors.ErrInvalidInput
	}
	if err := s.AuthorizeAction(ctx, command.GuildID, command.ActorID, "canSlot"); err != nil {
		return nil, err
	}
	commands, ok := s.repo.(WarehouseCommandRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	return commands.MoveGuildWarehouseAtomic(ctx, command)
}

func (s *Service) DropGuildWarehouseAtomic(ctx context.Context, command WarehouseDropCommand) (*WarehouseDropResult, error) {
	if command.GuildID <= 0 || command.ActorID <= 0 || command.SlotID <= 0 {
		return nil, pkgerrors.ErrInvalidInput
	}
	if err := s.AuthorizeAction(ctx, command.GuildID, command.ActorID, "canSlot"); err != nil {
		return nil, err
	}
	commands, ok := s.repo.(WarehouseCommandRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	return commands.DropGuildWarehouseAtomic(ctx, command)
}
