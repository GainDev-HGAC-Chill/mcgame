// Guild building commands validate catalog facts before durable mutations.
package guild

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	domainguild "mcgame-server/internal/domain/guild"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

const (
	BuildingCreateMode  = 2
	BuildingUpgradeMode = 1
	GuildBuildingMapID  = 49
)

type GuildBuildingCatalog interface {
	GetBuilding(id int) *models.BuildingTemplate
	GetAllBuildings() []*models.BuildingTemplate
	GetExtendPositionsByMapID(mapID int) []*models.ExtendPositionTemplate
}

type BuildingStartCommand struct {
	GuildID           int64
	ActorID           int64
	SourceID          int64
	ExtendID          int
	TargetTemplateID  int
	CurrentTemplateID int
	Mode              int
	ExpectedRevision  int64
	PrerequisiteIDs   []int
	FamilyIDs         []int
	NumLimit          int
}

type BuildingFinishCommand struct {
	GuildID          int64
	ActorID          int64
	BuildingID       int64
	TargetTemplateID int
	ExpectedRevision int64
	MoneyCost        int64
	ExperienceCost   int64
	GeneralCost      int
	RareCost         int
	SpecialCost      int
	HallLevel        int
}

type BuildingCommandRepository interface {
	ListGuildBuildings(context.Context, int64) ([]*domainguild.BuildingState, error)
	StartGuildBuildingAtomic(context.Context, BuildingStartCommand) (*domainguild.BuildingState, error)
	FinishGuildBuildingAtomic(context.Context, BuildingFinishCommand) (*domainguild.BuildingState, error)
	ResetGuildBuildingAtomic(context.Context, int64, int64, int64) (*domainguild.BuildingState, error)
}

type BuildingMutationResult struct {
	Building *domainguild.BuildingState
	Guild    *domainguild.Guild
}

func (s *Service) ListGuildBuildings(ctx context.Context, guildID int64) ([]*domainguild.BuildingState, error) {
	repo, ok := s.repo.(BuildingCommandRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	return repo.ListGuildBuildings(ctx, guildID)
}

func (s *Service) BuildingFunctionAvailable(ctx context.Context, guildID int64, functionName string) (bool, error) {
	if guildID <= 0 || functionName == "" || s.buildingCatalog == nil {
		return false, pkgerrors.ErrInvalidInput
	}
	repo, ok := s.repo.(BuildingCommandRepository)
	if !ok {
		return false, ErrUnsupportedAction
	}
	states, err := repo.ListGuildBuildings(ctx, guildID)
	if err != nil {
		return false, fmt.Errorf("failed to list guild buildings: %w", err)
	}
	for _, state := range states {
		if state == nil || state.Pending() || state.TemplateID <= 0 {
			continue
		}
		template := s.buildingCatalog.GetBuilding(state.TemplateID)
		if template != nil && buildingTemplateExposesFunction(template, functionName) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) StartGuildBuilding(ctx context.Context, guildID, actorID, sourceID int64, targetID, mode, mapID int) (*domainguild.BuildingState, error) {
	if guildID <= 0 || actorID <= 0 || sourceID <= 0 || targetID <= 0 || mapID != GuildBuildingMapID || (mode != BuildingCreateMode && mode != BuildingUpgradeMode) {
		return nil, pkgerrors.ErrInvalidInput
	}
	if err := s.AuthorizeLeader(ctx, guildID, actorID); err != nil {
		return nil, err
	}
	repo, ok := s.repo.(BuildingCommandRepository)
	if !ok || s.buildingCatalog == nil {
		return nil, ErrUnsupportedAction
	}
	states, err := repo.ListGuildBuildings(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to list guild buildings: %w", err)
	}
	command, err := s.resolveBuildingStart(guildID, actorID, sourceID, targetID, mode, states)
	if err != nil {
		return nil, err
	}
	return repo.StartGuildBuildingAtomic(ctx, command)
}

func (s *Service) FinishGuildBuilding(ctx context.Context, guildID, actorID, buildingID int64, expectedMode int) (*BuildingMutationResult, error) {
	if guildID <= 0 || actorID <= 0 || buildingID <= 0 || (expectedMode != BuildingCreateMode && expectedMode != BuildingUpgradeMode) {
		return nil, pkgerrors.ErrInvalidInput
	}
	if err := s.AuthorizeLeader(ctx, guildID, actorID); err != nil {
		return nil, err
	}
	repo, ok := s.repo.(BuildingCommandRepository)
	if !ok || s.buildingCatalog == nil {
		return nil, ErrUnsupportedAction
	}
	states, err := repo.ListGuildBuildings(ctx, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to list guild buildings: %w", err)
	}
	state := findBuildingByID(states, buildingID)
	if state == nil || !state.Pending() || state.Mode != expectedMode {
		return nil, pkgerrors.ErrNotFound
	}
	template := s.buildingCatalog.GetBuilding(state.PendingTemplateID)
	command, err := buildingFinishCommand(guildID, actorID, state, template)
	if err != nil {
		return nil, err
	}
	committed, err := repo.FinishGuildBuildingAtomic(ctx, command)
	if err != nil {
		return nil, err
	}
	guildItem, err := s.GetGuildForClient(ctx, guildID)
	if err != nil {
		return nil, err
	}
	return &BuildingMutationResult{Building: committed, Guild: guildItem}, nil
}

func (s *Service) ResetGuildBuilding(ctx context.Context, guildID, actorID, buildingID int64) (*domainguild.BuildingState, error) {
	if guildID <= 0 || actorID <= 0 || buildingID <= 0 {
		return nil, pkgerrors.ErrInvalidInput
	}
	if err := s.AuthorizeLeader(ctx, guildID, actorID); err != nil {
		return nil, err
	}
	repo, ok := s.repo.(BuildingCommandRepository)
	if !ok {
		return nil, ErrUnsupportedAction
	}
	return repo.ResetGuildBuildingAtomic(ctx, guildID, actorID, buildingID)
}

func (s *Service) resolveBuildingStart(guildID, actorID, sourceID int64, targetID, mode int, states []*domainguild.BuildingState) (BuildingStartCommand, error) {
	target := s.buildingCatalog.GetBuilding(targetID)
	if !validBuildingTemplate(target) || target.Type != 1 {
		return BuildingStartCommand{}, pkgerrors.ErrInvalidInput
	}
	command := BuildingStartCommand{GuildID: guildID, ActorID: actorID, SourceID: sourceID, TargetTemplateID: targetID, Mode: mode}
	if mode == BuildingCreateMode {
		if target.Level != 1 || sourceID > math.MaxInt {
			return BuildingStartCommand{}, pkgerrors.ErrInvalidInput
		}
		command.ExtendID = int(sourceID)
		if !s.validGuildExtension(command.ExtendID) || findBuildingByExtend(states, command.ExtendID) != nil {
			return BuildingStartCommand{}, pkgerrors.ErrInvalidInput
		}
	} else {
		current := findBuildingByID(states, sourceID)
		if current == nil || current.Pending() {
			return BuildingStartCommand{}, pkgerrors.ErrInvalidInput
		}
		currentTemplate := s.buildingCatalog.GetBuilding(current.TemplateID)
		if !validBuildingTemplate(currentTemplate) || currentTemplate.CodeName != target.CodeName || target.Level != currentTemplate.Level+1 {
			return BuildingStartCommand{}, pkgerrors.ErrInvalidInput
		}
		command.ExtendID = current.ExtendID
		command.CurrentTemplateID = current.TemplateID
		command.ExpectedRevision = current.Revision
	}
	prerequisites, valid := parseBuildingIDs(target.PreBuilding)
	if !valid {
		return BuildingStartCommand{}, ErrUnsupportedAction
	}
	command.PrerequisiteIDs = prerequisites
	command.FamilyIDs = s.buildingFamilyIDs(target.CodeName)
	limit, ok := exactNonnegativeInt(target.NumLimit)
	if !ok || limit == 0 {
		return BuildingStartCommand{}, ErrUnsupportedAction
	}
	command.NumLimit = limit
	if !buildingPrerequisitesMet(command.PrerequisiteIDs, command.ExtendID, states) || !buildingCountAllowed(command, states) {
		return BuildingStartCommand{}, ErrUnsupportedAction
	}
	if _, err := buildingFinishCommand(guildID, actorID, &domainguild.BuildingState{ID: sourceID, PendingTemplateID: targetID}, target); err != nil {
		return BuildingStartCommand{}, err
	}
	return command, nil
}

func (s *Service) validGuildExtension(extendID int) bool {
	for _, position := range s.buildingCatalog.GetExtendPositionsByMapID(GuildBuildingMapID) {
		if position != nil && int(position.ID) == extendID && position.Mid == GuildBuildingMapID && position.BuildType == 1 && position.Tid == 1 {
			return true
		}
	}
	return false
}

func (s *Service) buildingFamilyIDs(codeName string) []int {
	ids := make([]int, 0, 6)
	for _, row := range s.buildingCatalog.GetAllBuildings() {
		if validBuildingTemplate(row) && row.CodeName == codeName && row.Type == 1 {
			ids = append(ids, int(row.ID))
		}
	}
	return ids
}

func buildingFinishCommand(guildID, actorID int64, state *domainguild.BuildingState, template *models.BuildingTemplate) (BuildingFinishCommand, error) {
	if !validBuildingTemplate(template) || template.ID != int64(state.PendingTemplateID) || template.GoldCost != 0 {
		return BuildingFinishCommand{}, ErrUnsupportedAction
	}
	money, ok := exactNonnegativeInt64(template.MoneyCost)
	if !ok {
		return BuildingFinishCommand{}, ErrUnsupportedAction
	}
	experience, ok := exactNonnegativeInt64(template.ExpCost)
	if !ok {
		return BuildingFinishCommand{}, ErrUnsupportedAction
	}
	general, ok := exactNonnegativeInt(template.GenMCost)
	if !ok {
		return BuildingFinishCommand{}, ErrUnsupportedAction
	}
	rare, ok := exactNonnegativeInt(template.RareMCost)
	if !ok {
		return BuildingFinishCommand{}, ErrUnsupportedAction
	}
	special, ok := exactNonnegativeInt(template.SPMCost)
	if !ok {
		return BuildingFinishCommand{}, ErrUnsupportedAction
	}
	hallLevel := 0
	if template.CodeName == "BUILD100001" {
		level, levelOK := exactNonnegativeInt(template.Level)
		property, propertyOK := exactNonnegativeInt(template.Prop1)
		propertyValue, propertyValueOK := exactNonnegativeInt(template.Prop1Value)
		if !levelOK || !propertyOK || !propertyValueOK || level <= 0 || property != 4 || propertyValue != level {
			return BuildingFinishCommand{}, ErrUnsupportedAction
		}
		hallLevel = level
	}
	return BuildingFinishCommand{GuildID: guildID, ActorID: actorID, BuildingID: state.ID, TargetTemplateID: state.PendingTemplateID, ExpectedRevision: state.Revision, MoneyCost: money, ExperienceCost: experience, GeneralCost: general, RareCost: rare, SpecialCost: special, HallLevel: hallLevel}, nil
}

func validBuildingTemplate(template *models.BuildingTemplate) bool {
	return template != nil && template.ID > 0 && template.Level >= 0 && template.Level == math.Trunc(template.Level) && template.Type == math.Trunc(template.Type) && template.CodeName != ""
}

func buildingTemplateExposesFunction(template *models.BuildingTemplate, functionName string) bool {
	if template == nil {
		return false
	}
	switch functionName {
	case "showGuildWarehouse", "showDonatePanel", "showGuildSkillPanel":
		return strings.Contains(template.FuncScript, functionName)
	default:
		return false
	}
}

func parseBuildingIDs(value string) ([]int, bool) {
	if strings.TrimSpace(value) == "" {
		return nil, true
	}
	parts := strings.Split(value, "|")
	ids := make([]int, 0, len(parts))
	seen := make(map[int]struct{}, len(parts))
	for _, part := range parts {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			return nil, false
		}
		if _, exists := seen[id]; exists {
			return nil, false
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, true
}

func buildingPrerequisitesMet(required []int, _ int, states []*domainguild.BuildingState) bool {
	for _, id := range required {
		if id == 1 || id == 10 {
			continue
		}
		found := false
		for _, state := range states {
			if state != nil && state.TemplateID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func buildingCountAllowed(command BuildingStartCommand, states []*domainguild.BuildingState) bool {
	count := 0
	for _, state := range states {
		if state == nil || state.ExtendID == command.ExtendID {
			continue
		}
		for _, id := range command.FamilyIDs {
			if state.TemplateID == id || state.PendingTemplateID == id {
				count++
				break
			}
		}
	}
	return count < command.NumLimit
}

func findBuildingByID(states []*domainguild.BuildingState, id int64) *domainguild.BuildingState {
	for _, state := range states {
		if state != nil && state.ID == id {
			return state
		}
	}
	return nil
}

func findBuildingByExtend(states []*domainguild.BuildingState, extendID int) *domainguild.BuildingState {
	for _, state := range states {
		if state != nil && state.ExtendID == extendID {
			return state
		}
	}
	return nil
}

func exactNonnegativeInt64(value float64) (int64, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value >= float64(math.MaxInt64) || value != math.Trunc(value) {
		return 0, false
	}
	return int64(value), true
}

func exactNonnegativeInt(value float64) (int, bool) {
	if value > float64(math.MaxInt) {
		return 0, false
	}
	result, ok := exactNonnegativeInt64(value)
	return int(result), ok
}
