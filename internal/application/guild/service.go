// Open-sourced by BaoLT

// Guild application service with business logic
package guild

import (
	"context"
	"errors"
	"fmt"

	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/guild"

	"go.uber.org/zap"
)

var (
	ErrGuildNotFound       = errors.New("guild not found")
	ErrGuildNameTaken      = errors.New("guild name already taken")
	ErrAlreadyInGuild      = errors.New("character already in a guild")
	ErrNotInGuild          = errors.New("character not in a guild")
	ErrNotGuildLeader      = errors.New("not the guild leader")
	ErrCannotKickLeader    = errors.New("cannot kick the guild leader")
	ErrGuildFull           = errors.New("guild is full")
	ErrInsufficientFunds   = errors.New("insufficient guild funds")
	ErrInsufficientLevel   = errors.New("insufficient character level")
	ErrUnsupportedAction   = errors.New("guild action has no authoritative cost")
	ErrInvalidContribution = errors.New("invalid guild contribution")
)

type Repository interface {
	Create(ctx context.Context, g *guild.Guild) error
	GetByID(ctx context.Context, guildID int64) (*guild.Guild, error)
	GetByName(ctx context.Context, name string) (*guild.Guild, error)
	GetByMemberID(ctx context.Context, characterID int64) (*guild.Guild, error)
	ListGuilds(ctx context.Context) ([]*guild.Guild, error)
	GetMembers(ctx context.Context, guildID int64) ([]*guild.GuildMember, error)
	GetMemberByID(ctx context.Context, memberID int64) (*guild.GuildMember, error)
	AddMember(ctx context.Context, guildID, characterID int64, rank int) error
	RemoveMember(ctx context.Context, guildID, characterID int64) error
	UpdateMemberRank(ctx context.Context, guildID, characterID int64, rank int) error
	AddMemberContribution(ctx context.Context, guildID, characterID int64, normal, donate, total int64) error
	SetMemberContribution(ctx context.Context, guildID, characterID int64, normal, donate, total int64) error
	UpdateLeader(ctx context.Context, guildID, oldLeaderID, newLeaderID int64) error
	CreateApplication(ctx context.Context, application *guild.GuildApplication) error
	GetApplicationByID(ctx context.Context, applicationID int64) (*guild.GuildApplication, error)
	GetApplicationByCharacterID(ctx context.Context, characterID int64) (*guild.GuildApplication, error)
	ListApplications(ctx context.Context, guildID int64) ([]*guild.GuildApplication, error)
	DeleteApplication(ctx context.Context, applicationID int64) error
	SaveRankSettings(ctx context.Context, guildID, actorID int64, settings []guild.RankSetting) error
	LoadRankSettings(ctx context.Context, guildID int64) ([]guild.RankSetting, error)
	SaveBankPageCount(ctx context.Context, guildID, actorID int64, pageCount int) error
	LoadBankPageCount(ctx context.Context, guildID int64) (int, error)
	ListWarehouseSlots(ctx context.Context, guildID int64) ([]*guild.GuildWarehouseSlot, error)
	UpdateWarehouseSlot(ctx context.Context, guildID, slotID int64, slotIndex int, stackCount int) error
	DeleteWarehouseSlot(ctx context.Context, guildID int64, slotID int64) error
	GetGuildSkillIDs(ctx context.Context, guildID int64) ([]int, error)
	GetGuildSkillLevels(ctx context.Context, guildID int64) (map[int]int, error)
	UpsertGuildSkill(ctx context.Context, guildID int64, skillID int) error
	Update(ctx context.Context, g *guild.Guild) error
	Delete(ctx context.Context, guildID int64) error
}

type CharacterService interface {
	GetByID(ctx context.Context, characterID int64) (*character.Character, error)
	Update(ctx context.Context, char *character.Character) error
}

type QuestService interface {
	AbandonGuildQuests(ctx context.Context, charID int64) ([]int, error)
}

type Service struct {
	repo            Repository
	charSvc         CharacterService
	questSvc        QuestService
	logger          *zap.Logger
	skillCatalog    GuildSkillLookup
	buildingCatalog GuildBuildingCatalog
}

func NewService(repo Repository, charSvc CharacterService, logger *zap.Logger) *Service {
	return &Service{
		repo:    repo,
		charSvc: charSvc,
		logger:  logger,
	}
}

func (s *Service) SetQuestService(questSvc QuestService) {
	s.questSvc = questSvc
}

func (s *Service) SetSkillCatalog(catalog GuildSkillLookup) {
	s.skillCatalog = catalog
}

func (s *Service) SetBuildingCatalog(catalog GuildBuildingCatalog) {
	s.buildingCatalog = catalog
}

// CreateGuild creates a new guild with the character as leader
func (s *Service) CreateGuild(ctx context.Context, leaderID int64, name string) (*guild.Guild, error) {
	return nil, ErrUnsupportedAction
}

// GetGuild retrieves a guild by ID with members
func (s *Service) GetGuild(ctx context.Context, guildID int64) (*guild.Guild, error) {
	g, err := s.repo.GetByID(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild: %w", err)
	}
	if g == nil {
		return nil, ErrGuildNotFound
	}

	// Load members
	members, err := s.repo.GetMembers(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	g.Members = members

	return g, nil
}

// GetGuildByMember retrieves the guild that a character belongs to
func (s *Service) GetGuildByMember(ctx context.Context, characterID int64) (*guild.Guild, error) {
	g, err := s.repo.GetByMemberID(ctx, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild by member: %w", err)
	}
	if g == nil {
		return nil, nil // Not in a guild
	}

	// Load members
	members, err := s.repo.GetMembers(ctx, g.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get members: %w", err)
	}
	g.Members = members

	return g, nil
}

// InviteMember adds a character to the guild
func (s *Service) InviteMember(ctx context.Context, guildID, inviterID, inviteeID int64) error {
	return ErrUnsupportedAction
}

// KickMember removes a character from the guild
func (s *Service) KickMember(ctx context.Context, guildID, kickerID, targetID int64) error {
	commands, ok := s.repo.(LifecycleCommandRepository)
	if !ok {
		return ErrUnsupportedAction
	}
	_, err := commands.KickGuildMemberAtomic(ctx, guildID, kickerID, targetID)
	if err == nil {
		s.cleanupGuildDeparture(ctx, targetID)
	}
	return err
}

// LeaveMember allows a member to leave the guild
func (s *Service) LeaveMember(ctx context.Context, guildID, characterID int64) error {
	commands, ok := s.repo.(LifecycleCommandRepository)
	if !ok {
		return ErrUnsupportedAction
	}
	_, err := commands.LeaveGuildAtomic(ctx, guildID, characterID)
	if err == nil {
		s.cleanupGuildDeparture(ctx, characterID)
	}
	return err
}

// PromoteMember promotes a member to a higher rank
func (s *Service) PromoteMember(ctx context.Context, guildID, promoterID, targetID int64, newRank int) error {
	if newRank < guild.RankViceLeader || newRank > guild.RankMember {
		return fmt.Errorf("invalid guild rank: %d", newRank)
	}
	commands, ok := s.repo.(LifecycleCommandRepository)
	if !ok {
		return ErrUnsupportedAction
	}
	return commands.SetGuildMemberRankAtomic(ctx, guildID, promoterID, targetID, newRank)
}

// DisbandGuild deletes the guild
func (s *Service) DisbandGuild(ctx context.Context, guildID, characterID int64) error {
	commands, ok := s.repo.(LifecycleCommandRepository)
	if !ok {
		return ErrUnsupportedAction
	}
	members, err := s.repo.GetMembers(ctx, guildID)
	if err != nil {
		return err
	}
	memberIDs := make([]int64, 0, len(members))
	for _, member := range members {
		if member != nil && member.CharacterID > 0 {
			memberIDs = append(memberIDs, member.CharacterID)
		}
	}
	result, err := commands.DisbandGuildAtomic(ctx, DisbandGuildCommand{GuildID: guildID, LeaderID: characterID, ExpectedMemberIDs: memberIDs})
	if err != nil {
		return err
	}
	for _, committed := range result.Characters {
		s.cleanupGuildDeparture(ctx, committed.CharacterID)
	}
	return nil
}

// UpdateGuildInfo updates guild information
func (s *Service) UpdateGuildInfo(ctx context.Context, guildID, updaterID int64, announcement, description string) error {
	// Get guild
	g, err := s.repo.GetByID(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild: %w", err)
	}
	if g == nil {
		return ErrGuildNotFound
	}

	members, err := s.repo.GetMembers(ctx, guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild members: %w", err)
	}
	var updater *guild.GuildMember
	for _, member := range members {
		if member != nil && member.CharacterID == updaterID {
			updater = member
			break
		}
	}
	if err := s.authorizeMember(ctx, g, updater, guild.PermissionInfo, false); err != nil {
		return err
	}

	// Update fields
	g.Announcement = announcement
	if description != "" {
		g.Description = description
	} else if announcement != "" {
		g.Description = announcement
	}

	commands, ok := s.repo.(GuildMetadataRepository)
	if !ok {
		return ErrUnsupportedAction
	}
	if err := commands.SetGuildNotice(ctx, guildID, updaterID, g.Announcement, g.Description); err != nil {
		return fmt.Errorf("failed to update guild: %w", err)
	}

	s.logger.Info("Guild info updated",
		zap.Int64("guild_id", guildID),
		zap.Int64("updater_id", updaterID))

	return nil
}
