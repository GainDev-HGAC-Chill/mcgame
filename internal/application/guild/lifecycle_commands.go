// Guild lifecycle commands own membership, contribution, and restore transitions atomically.
package guild

import (
	"context"

	domainguild "mcgame-server/internal/domain/guild"
)

type ApproveApplicationResult struct {
	Application *domainguild.GuildApplication
	Member      *domainguild.GuildMember
	Character   CharacterCommitState
}

type ExitGuildResult struct {
	Member    *domainguild.GuildMember
	Character CharacterCommitState
}

type TransferLeadershipResult struct {
	GuildID       int64
	OldLeaderID   int64
	NewLeaderID   int64
	NewLeaderName string
}

type DisbandGuildResult struct {
	GuildID    int64
	Characters []CharacterCommitState
}

type DisbandGuildCommand struct {
	GuildID           int64
	LeaderID          int64
	ExpectedMemberIDs []int64
}

type LifecycleCommandRepository interface {
	ApplyGuildAtomic(context.Context, int64, int64) (*domainguild.GuildApplication, error)
	ApproveGuildApplicationAtomic(context.Context, int64, int64) (*ApproveApplicationResult, error)
	RefuseGuildApplicationAtomic(context.Context, int64, int64) (*domainguild.GuildApplication, error)
	CancelGuildApplicationAtomic(context.Context, int64, int64) (*domainguild.GuildApplication, error)
	LeaveGuildAtomic(context.Context, int64, int64) (*ExitGuildResult, error)
	KickGuildMemberAtomic(context.Context, int64, int64, int64) (*ExitGuildResult, error)
	TransferGuildLeadershipAtomic(context.Context, int64, int64, int64) (*TransferLeadershipResult, error)
	SetGuildMemberRankAtomic(context.Context, int64, int64, int64, int) error
	DisbandGuildAtomic(context.Context, DisbandGuildCommand) (*DisbandGuildResult, error)
}
