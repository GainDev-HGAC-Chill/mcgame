// Gathering service prepares immutable attempts and delegates atomic completion.
package gathering

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	mathrand "math/rand/v2"
	"time"

	appskill "mcgame-server/internal/application/skill"
	domainchar "mcgame-server/internal/domain/character"
	domaingathering "mcgame-server/internal/domain/gathering"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

var (
	ErrResourceUnavailable = errors.New("gathering resource unavailable")
	ErrSkillLevelNotEnough = errors.New("gathering skill level not enough")
	ErrToolNotEquipped     = errors.New("gathering tool not equipped")
	ErrToolInvalid         = errors.New("gathering tool invalid")
	ErrToolBroken          = errors.New("gathering tool broken")
	ErrVigorNotEnough      = errors.New("gathering vigor not enough")
	ErrCatalogInvalid      = errors.New("gathering catalog invalid")
	ErrServiceUnavailable  = errors.New("gathering service unavailable")
)

const (
	fishingToolSlotIndex = 13
	fishingToolType      = 1201
	herbVigorCost        = 2
	herbGatheringDelay   = 5 * time.Second
)

type CharacterReader interface {
	FindByID(ctx context.Context, id int64) (*domainchar.Character, error)
}

type SkillReader interface {
	GetLifeSkillLevel(ctx context.Context, charID int64, skillType int) (int, error)
}

type ItemReader interface {
	GetItemInSlot(ctx context.Context, charID int64, slotType domainitem.SlotType, slotIndex int) (*domainitem.Item, error)
}

type Catalog interface {
	ItemCatalog
	GetNPC(id int) *models.NpcTemplate
	GetEquipment(id int) *models.EquiptTemplateTemplate
}

type RandomSource interface {
	IntN(n int) int
	Float64() float64
}

type StartResult struct {
	Kind          domaingathering.ResourceKind
	NPCID         int
	ResourceLevel int
	ToolItemID    int64
	Delay         time.Duration
}

type PreparedAttempt struct {
	Start      StartResult
	Completion domaingathering.CompletionCommand
}

type Service struct {
	repo       domaingathering.CommandRepository
	characters CharacterReader
	skills     SkillReader
	items      ItemReader
	catalog    Catalog
	random     RandomSource
}

type defaultRandomSource struct{}

func (defaultRandomSource) IntN(n int) int {
	return mathrand.IntN(n)
}

func (defaultRandomSource) Float64() float64 {
	return mathrand.Float64()
}

func NewService(repo domaingathering.CommandRepository, characters CharacterReader, skills SkillReader, items ItemReader, catalog Catalog) *Service {
	return &Service{
		repo:       repo,
		characters: characters,
		skills:     skills,
		items:      items,
		catalog:    catalog,
		random:     defaultRandomSource{},
	}
}

func (s *Service) SetRandomSource(source RandomSource) {
	if source != nil {
		s.random = source
	}
}

func (s *Service) ValidateCatalog() error {
	if s == nil || s.catalog == nil {
		return ErrCatalogInvalid
	}
	if err := ValidateOrdinaryCatalog(s.catalog); err != nil {
		return errors.Join(ErrCatalogInvalid, err)
	}
	return nil
}

func (s *Service) PrepareFishing(ctx context.Context, characterID int64, npcID int) (*PreparedAttempt, error) {
	if s == nil || s.skills == nil || s.items == nil || s.catalog == nil || s.random == nil {
		return nil, ErrServiceUnavailable
	}
	resourceLevel, ok := OrdinaryFishingNodeLevel(npcID)
	if !ok || !s.validNPC(npcID, 21, resourceLevel) {
		return nil, ErrResourceUnavailable
	}
	skillLevel, err := s.skills.GetLifeSkillLevel(ctx, characterID, appskill.FishSkillType)
	if err != nil {
		return nil, err
	}
	if skillLevel < resourceLevel {
		return nil, ErrSkillLevelNotEnough
	}
	tool, err := s.items.GetItemInSlot(ctx, characterID, domainitem.SlotTypeEquipped, fishingToolSlotIndex)
	if err != nil {
		return nil, err
	}
	if tool == nil {
		return nil, ErrToolNotEquipped
	}
	equipment := s.catalog.GetEquipment(tool.TemplateID)
	if equipment == nil || int(equipment.Type) != fishingToolType || tool.SlotType != domainitem.SlotTypeEquipped || tool.SlotIndex != fishingToolSlotIndex {
		return nil, ErrToolInvalid
	}
	if tool.Durability == nil || *tool.Durability <= 0 {
		return nil, ErrToolBroken
	}
	if equipment.ProTime <= 0 || equipment.SuccRate < 0 || equipment.SuccRate > 100 {
		return nil, ErrCatalogInvalid
	}
	rewardTemplateID, err := s.selectReward(domaingathering.ResourceKindFishing, resourceLevel)
	if err != nil {
		return nil, err
	}
	succeeded := s.random.Float64()*100 < equipment.SuccRate
	masteryGain := 0
	if succeeded {
		masteryGain = MasteryGain(skillLevel, resourceLevel)
	}
	attemptID, err := newAttemptID()
	if err != nil {
		return nil, err
	}
	return &PreparedAttempt{
		Start: StartResult{
			Kind:          domaingathering.ResourceKindFishing,
			NPCID:         npcID,
			ResourceLevel: resourceLevel,
			ToolItemID:    tool.ID,
			Delay:         time.Duration(equipment.ProTime * float64(time.Second)),
		},
		Completion: domaingathering.CompletionCommand{
			AttemptID:        attemptID,
			CharacterID:      characterID,
			Kind:             domaingathering.ResourceKindFishing,
			NPCID:            npcID,
			ResourceLevel:    resourceLevel,
			RewardTemplateID: rewardTemplateID,
			ToolItemID:       tool.ID,
			Succeeded:        succeeded,
			MasteryGain:      masteryGain,
		},
	}, nil
}

func (s *Service) PrepareHerb(ctx context.Context, characterID int64, npcID int, placementAuthorized bool) (*PreparedAttempt, error) {
	if s == nil || s.characters == nil || s.skills == nil || s.catalog == nil || s.random == nil {
		return nil, ErrServiceUnavailable
	}
	resourceLevel, ok := HerbTemplateLevel(npcID)
	if !placementAuthorized || !ok || !s.validNPC(npcID, 23, resourceLevel) {
		return nil, ErrResourceUnavailable
	}
	skillLevel, err := s.skills.GetLifeSkillLevel(ctx, characterID, appskill.HerbSkillType)
	if err != nil {
		return nil, err
	}
	if skillLevel < resourceLevel {
		return nil, ErrSkillLevelNotEnough
	}
	character, err := s.characters.FindByID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if character == nil || character.CurrentSP < herbVigorCost {
		return nil, ErrVigorNotEnough
	}
	rewardTemplateID, err := s.selectReward(domaingathering.ResourceKindHerb, resourceLevel)
	if err != nil {
		return nil, err
	}
	attemptID, err := newAttemptID()
	if err != nil {
		return nil, err
	}
	return &PreparedAttempt{
		Start: StartResult{
			Kind:          domaingathering.ResourceKindHerb,
			NPCID:         npcID,
			ResourceLevel: resourceLevel,
			Delay:         herbGatheringDelay,
		},
		Completion: domaingathering.CompletionCommand{
			AttemptID:        attemptID,
			CharacterID:      characterID,
			Kind:             domaingathering.ResourceKindHerb,
			NPCID:            npcID,
			ResourceLevel:    resourceLevel,
			RewardTemplateID: rewardTemplateID,
			Succeeded:        true,
			VigorCost:        herbVigorCost,
			MasteryGain:      MasteryGain(skillLevel, resourceLevel),
		},
	}, nil
}

func (s *Service) Complete(ctx context.Context, command domaingathering.CompletionCommand) (*domaingathering.CompletionResult, error) {
	if s == nil || s.repo == nil {
		return nil, ErrServiceUnavailable
	}
	return s.repo.CompleteGathering(ctx, command)
}

func (s *Service) validNPC(npcID int, expectedType int, expectedLevel int) bool {
	npc := s.catalog.GetNPC(npcID)
	return npc != nil && int(npc.Type) == expectedType && int(npc.Lv) == expectedLevel
}

func (s *Service) selectReward(kind domaingathering.ResourceKind, level int) (int, error) {
	rewards, ok := OrdinaryRewards(kind, level)
	if !ok {
		return 0, ErrResourceUnavailable
	}
	return rewards[s.random.IntN(len(rewards))], nil
}

func newAttemptID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := cryptorand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
