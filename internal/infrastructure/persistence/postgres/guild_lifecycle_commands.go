// PostgreSQL atomic guild membership lifecycle commands.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appguild "mcgame-server/internal/application/guild"
	domainguild "mcgame-server/internal/domain/guild"
)

type lifecycleApplicationDTO struct {
	ID             int64      `json:"id"`
	GuildID        int64      `json:"guild_id"`
	CharacterID    int64      `json:"character_id"`
	CharacterName  string     `json:"character_name"`
	CharacterClass int        `json:"character_class"`
	CharacterLevel int        `json:"character_level"`
	CharacterExp   int64      `json:"character_exp"`
	Message        string     `json:"message"`
	Status         int        `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	ProcessedAt    *time.Time `json:"processed_at"`
	ProcessedBy    *int64     `json:"processed_by"`
}

type lifecycleMemberDTO struct {
	ID                 int64      `json:"id"`
	GuildID            int64      `json:"guild_id"`
	CharacterID        int64      `json:"character_id"`
	CharacterName      string     `json:"character_name"`
	CharacterLevel     int        `json:"character_level"`
	CharacterClass     int        `json:"character_class"`
	CharacterExp       int64      `json:"character_exp"`
	Rank               int        `json:"rank"`
	Duty               string     `json:"duty"`
	ContributionNormal int64      `json:"contribution_normal"`
	ContributionDonate int64      `json:"contribution_donate"`
	ContributionTotal  int64      `json:"contribution_total"`
	ContributionWeekly int64      `json:"contribution_weekly"`
	JoinedAt           time.Time  `json:"joined_at"`
	LastOnline         *time.Time `json:"last_online"`
}

type lifecycleCharacterDTO struct {
	CharacterID        int64  `json:"character_id"`
	GuildID            *int64 `json:"guild_id"`
	Money              int64  `json:"money"`
	MoneyBind          int64  `json:"money_bind"`
	Gold               int64  `json:"gold"`
	GuildContrib       int64  `json:"guild_contrib"`
	DonateContrib      int64  `json:"donate_contrib"`
	GuildRestoreNormal int64  `json:"guild_restore_normal"`
	GuildRestoreDonate int64  `json:"guild_restore_donate"`
}

type approveLifecycleDTO struct {
	Application lifecycleApplicationDTO `json:"application"`
	Member      lifecycleMemberDTO      `json:"member"`
	Character   lifecycleCharacterDTO   `json:"character"`
}

type exitLifecycleDTO struct {
	Member    lifecycleMemberDTO    `json:"member"`
	Character lifecycleCharacterDTO `json:"character"`
}

type disbandLifecycleDTO struct {
	GuildID    int64                   `json:"guild_id"`
	Characters []lifecycleCharacterDTO `json:"characters"`
}

func (r *GuildRepository) ApplyGuildAtomic(ctx context.Context, guildID, characterID int64) (*domainguild.GuildApplication, error) {
	var raw []byte
	if err := r.db.pool.QueryRow(ctx, "select player.apply_guild_atomic($1,$2)", guildID, characterID).Scan(&raw); err != nil {
		return nil, mapGuildCommandError("apply to guild", err)
	}
	var dto lifecycleApplicationDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return nil, fmt.Errorf("failed to decode guild application: %w", err)
	}
	return dto.application(), nil
}

func (r *GuildRepository) ApproveGuildApplicationAtomic(ctx context.Context, applicationID, approverID int64) (*appguild.ApproveApplicationResult, error) {
	var raw []byte
	if err := r.db.pool.QueryRow(ctx, "select player.approve_guild_application_atomic($1,$2)", applicationID, approverID).Scan(&raw); err != nil {
		return nil, mapGuildCommandError("approve guild application", err)
	}
	var dto approveLifecycleDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return nil, fmt.Errorf("failed to decode approved guild application: %w", err)
	}
	return &appguild.ApproveApplicationResult{Application: dto.Application.application(), Member: dto.Member.member(), Character: dto.Character.state()}, nil
}

func (r *GuildRepository) RefuseGuildApplicationAtomic(ctx context.Context, applicationID, approverID int64) (*domainguild.GuildApplication, error) {
	return r.deleteGuildApplicationAtomic(ctx, "refuse_guild_application_atomic", applicationID, approverID)
}

func (r *GuildRepository) CancelGuildApplicationAtomic(ctx context.Context, applicationID, characterID int64) (*domainguild.GuildApplication, error) {
	return r.deleteGuildApplicationAtomic(ctx, "cancel_guild_application_atomic", applicationID, characterID)
}

func (r *GuildRepository) deleteGuildApplicationAtomic(ctx context.Context, function string, applicationID, actorID int64) (*domainguild.GuildApplication, error) {
	query := "select player." + function + "($1,$2)"
	var raw []byte
	if err := r.db.pool.QueryRow(ctx, query, applicationID, actorID).Scan(&raw); err != nil {
		return nil, mapGuildCommandError("delete guild application", err)
	}
	var dto lifecycleApplicationDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return nil, fmt.Errorf("failed to decode deleted guild application: %w", err)
	}
	return dto.application(), nil
}

func (r *GuildRepository) LeaveGuildAtomic(ctx context.Context, guildID, characterID int64) (*appguild.ExitGuildResult, error) {
	return r.exitGuildAtomic(ctx, guildID, characterID, characterID, false)
}

func (r *GuildRepository) KickGuildMemberAtomic(ctx context.Context, guildID, actorID, targetID int64) (*appguild.ExitGuildResult, error) {
	return r.exitGuildAtomic(ctx, guildID, actorID, targetID, true)
}

func (r *GuildRepository) exitGuildAtomic(ctx context.Context, guildID, actorID, targetID int64, kick bool) (*appguild.ExitGuildResult, error) {
	var raw []byte
	if err := r.db.pool.QueryRow(ctx, "select player.exit_guild_member_atomic($1,$2,$3,$4)", guildID, actorID, targetID, kick).Scan(&raw); err != nil {
		return nil, mapGuildCommandError("exit guild", err)
	}
	var dto exitLifecycleDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return nil, fmt.Errorf("failed to decode guild exit: %w", err)
	}
	return &appguild.ExitGuildResult{Member: dto.Member.member(), Character: dto.Character.state()}, nil
}

func (r *GuildRepository) TransferGuildLeadershipAtomic(ctx context.Context, guildID, oldLeaderID, newLeaderID int64) (*appguild.TransferLeadershipResult, error) {
	result := &appguild.TransferLeadershipResult{}
	if err := r.db.pool.QueryRow(ctx, "select * from player.transfer_guild_leadership_atomic($1,$2,$3)", guildID, oldLeaderID, newLeaderID).
		Scan(&result.GuildID, &result.OldLeaderID, &result.NewLeaderID, &result.NewLeaderName); err != nil {
		return nil, mapGuildCommandError("transfer guild leadership", err)
	}
	return result, nil
}

func (r *GuildRepository) SetGuildMemberRankAtomic(ctx context.Context, guildID, actorID, targetID int64, rank int) error {
	var updated bool
	if err := r.db.pool.QueryRow(ctx, "select player.set_guild_member_rank_atomic($1,$2,$3,$4)", guildID, actorID, targetID, rank).Scan(&updated); err != nil {
		return fmt.Errorf("failed to set guild member rank atomically: %w", err)
	}
	if !updated {
		return appguild.ErrUnsupportedAction
	}
	return nil
}

func (r *GuildRepository) DisbandGuildAtomic(ctx context.Context, command appguild.DisbandGuildCommand) (*appguild.DisbandGuildResult, error) {
	var raw []byte
	if err := r.db.pool.QueryRow(ctx, "select player.disband_guild_atomic($1,$2,$3)", command.GuildID, command.LeaderID, command.ExpectedMemberIDs).Scan(&raw); err != nil {
		return nil, mapGuildCommandError("disband guild", err)
	}
	var dto disbandLifecycleDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return nil, fmt.Errorf("failed to decode guild disband: %w", err)
	}
	result := &appguild.DisbandGuildResult{GuildID: dto.GuildID, Characters: make([]appguild.CharacterCommitState, 0, len(dto.Characters))}
	for _, characterState := range dto.Characters {
		result.Characters = append(result.Characters, characterState.state())
	}
	return result, nil
}

func (d lifecycleApplicationDTO) application() *domainguild.GuildApplication {
	return &domainguild.GuildApplication{ID: d.ID, GuildID: d.GuildID, CharacterID: d.CharacterID, CharacterName: d.CharacterName, CharacterClass: d.CharacterClass, CharacterLevel: d.CharacterLevel, CharacterExp: d.CharacterExp, Message: d.Message, Status: d.Status, CreatedAt: d.CreatedAt, ProcessedAt: d.ProcessedAt, ProcessedBy: d.ProcessedBy}
}

func (d lifecycleMemberDTO) member() *domainguild.GuildMember {
	return &domainguild.GuildMember{ID: d.ID, GuildID: d.GuildID, CharacterID: d.CharacterID, CharacterName: d.CharacterName, CharacterLevel: d.CharacterLevel, CharacterClass: d.CharacterClass, CharacterExp: d.CharacterExp, Rank: d.Rank, Duty: d.Duty, ContributionNormal: d.ContributionNormal, ContributionDonate: d.ContributionDonate, ContributionTotal: d.ContributionTotal, ContributionWeekly: d.ContributionWeekly, JoinedAt: d.JoinedAt, LastOnline: d.LastOnline}
}

func (d lifecycleCharacterDTO) state() appguild.CharacterCommitState {
	return appguild.CharacterCommitState{CharacterID: d.CharacterID, GuildID: d.GuildID, Money: d.Money, MoneyBind: d.MoneyBind, Gold: d.Gold, GuildContrib: d.GuildContrib, DonateContrib: d.DonateContrib, GuildRestoreNormal: d.GuildRestoreNormal, GuildRestoreDonate: d.GuildRestoreDonate}
}

var _ appguild.LifecycleCommandRepository = (*GuildRepository)(nil)
