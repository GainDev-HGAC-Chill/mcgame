package gathering

import (
	"context"
	"errors"
	"testing"
	"time"

	domainchar "mcgame-server/internal/domain/character"
	domaingathering "mcgame-server/internal/domain/gathering"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

type serviceCatalog struct {
	items     map[int]*models.ItemTemplateTemplate
	npcs      map[int]*models.NpcTemplate
	equipment map[int]*models.EquiptTemplateTemplate
}

func (c *serviceCatalog) GetItem(id int) *models.ItemTemplateTemplate {
	return c.items[id]
}

func (c *serviceCatalog) GetNPC(id int) *models.NpcTemplate {
	return c.npcs[id]
}

func (c *serviceCatalog) GetEquipment(id int) *models.EquiptTemplateTemplate {
	return c.equipment[id]
}

type serviceCharacterReader struct {
	character *domainchar.Character
}

func (r serviceCharacterReader) FindByID(context.Context, int64) (*domainchar.Character, error) {
	return r.character, nil
}

type serviceSkillReader struct {
	level int
	err   error
}

func (r serviceSkillReader) GetLifeSkillLevel(context.Context, int64, int) (int, error) {
	return r.level, r.err
}

type serviceItemReader struct {
	item *domainitem.Item
	err  error
}

func (r serviceItemReader) GetItemInSlot(context.Context, int64, domainitem.SlotType, int) (*domainitem.Item, error) {
	return r.item, r.err
}

type serviceRandom struct {
	index int
	roll  float64
}

func (r serviceRandom) IntN(int) int {
	return r.index
}

func (r serviceRandom) Float64() float64 {
	return r.roll
}

type serviceCommandRepository struct {
	command domaingathering.CompletionCommand
	result  *domaingathering.CompletionResult
}

func (r *serviceCommandRepository) CompleteGathering(_ context.Context, command domaingathering.CompletionCommand) (*domaingathering.CompletionResult, error) {
	r.command = command
	return r.result, nil
}

func TestPrepareFishingBuildsImmutableSuccessfulAttempt(t *testing.T) {
	durability := 100
	catalog := &serviceCatalog{
		npcs: map[int]*models.NpcTemplate{1596: {ID: 1596, Type: 21, Lv: 7}},
		equipment: map[int]*models.EquiptTemplateTemplate{
			1438: {ID: 1438, Type: 1201, ProTime: 20, SuccRate: 75},
		},
	}
	tool := &domainitem.Item{ID: 91, CharacterID: 7, TemplateID: 1438, SlotType: domainitem.SlotTypeEquipped, SlotIndex: fishingToolSlotIndex, Durability: &durability}
	service := NewService(nil, nil, serviceSkillReader{level: 7}, serviceItemReader{item: tool}, catalog)
	service.SetRandomSource(serviceRandom{index: 1, roll: 0.74})

	attempt, err := service.PrepareFishing(context.Background(), 7, 1596)
	if err != nil {
		t.Fatalf("PrepareFishing() error = %v", err)
	}
	if attempt.Start.Delay != 20*time.Second || attempt.Start.ToolItemID != 91 {
		t.Fatalf("unexpected start result: %+v", attempt.Start)
	}
	if !attempt.Completion.Succeeded || attempt.Completion.RewardTemplateID != 2592 || attempt.Completion.MasteryGain != 2 {
		t.Fatalf("unexpected completion command: %+v", attempt.Completion)
	}
}

func TestPrepareFishingFailureHasNoMastery(t *testing.T) {
	durability := 1
	catalog := &serviceCatalog{
		npcs: map[int]*models.NpcTemplate{1189: {ID: 1189, Type: 21, Lv: 1}},
		equipment: map[int]*models.EquiptTemplateTemplate{
			1437: {ID: 1437, Type: 1201, ProTime: 20, SuccRate: 70},
		},
	}
	tool := &domainitem.Item{ID: 11, TemplateID: 1437, SlotType: domainitem.SlotTypeEquipped, SlotIndex: fishingToolSlotIndex, Durability: &durability}
	service := NewService(nil, nil, serviceSkillReader{level: 2}, serviceItemReader{item: tool}, catalog)
	service.SetRandomSource(serviceRandom{index: 0, roll: 0.70})

	attempt, err := service.PrepareFishing(context.Background(), 3, 1189)
	if err != nil {
		t.Fatalf("PrepareFishing() error = %v", err)
	}
	if attempt.Completion.Succeeded || attempt.Completion.MasteryGain != 0 {
		t.Fatalf("failed roll mutated mastery command: %+v", attempt.Completion)
	}
}

func TestPrepareFishingRejectsUnsupportedSkillAndToolStates(t *testing.T) {
	durability := 0
	catalog := &serviceCatalog{
		npcs: map[int]*models.NpcTemplate{1189: {ID: 1189, Type: 21, Lv: 1}},
		equipment: map[int]*models.EquiptTemplateTemplate{
			1437: {ID: 1437, Type: 1201, ProTime: 20, SuccRate: 70},
		},
	}

	tests := []struct {
		name  string
		npcID int
		level int
		item  *domainitem.Item
		want  error
	}{
		{name: "contest node", npcID: 1573, level: 10, want: ErrResourceUnavailable},
		{name: "low skill", npcID: 1189, level: 0, want: ErrSkillLevelNotEnough},
		{name: "missing rod", npcID: 1189, level: 1, want: ErrToolNotEquipped},
		{name: "broken rod", npcID: 1189, level: 1, item: &domainitem.Item{TemplateID: 1437, SlotType: domainitem.SlotTypeEquipped, SlotIndex: fishingToolSlotIndex, Durability: &durability}, want: ErrToolBroken},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(nil, nil, serviceSkillReader{level: test.level}, serviceItemReader{item: test.item}, catalog)
			_, err := service.PrepareFishing(context.Background(), 3, test.npcID)
			if !errors.Is(err, test.want) {
				t.Fatalf("PrepareFishing() error = %v; want %v", err, test.want)
			}
		})
	}
}

func TestPrepareHerbRequiresPlacementSkillAndVigor(t *testing.T) {
	catalog := &serviceCatalog{npcs: map[int]*models.NpcTemplate{1243: {ID: 1243, Type: 23, Lv: 3}}}
	service := NewService(nil, serviceCharacterReader{character: &domainchar.Character{CurrentSP: 2}}, serviceSkillReader{level: 4}, nil, catalog)
	service.SetRandomSource(serviceRandom{index: 1})

	if _, err := service.PrepareHerb(context.Background(), 8, 1243, false); !errors.Is(err, ErrResourceUnavailable) {
		t.Fatalf("unplaced PrepareHerb() error = %v", err)
	}
	attempt, err := service.PrepareHerb(context.Background(), 8, 1243, true)
	if err != nil {
		t.Fatalf("PrepareHerb() error = %v", err)
	}
	if attempt.Start.Delay != 5*time.Second || attempt.Completion.VigorCost != 2 || attempt.Completion.MasteryGain != 1 || attempt.Completion.RewardTemplateID != 2496 {
		t.Fatalf("unexpected herb attempt: %+v", attempt)
	}

	lowVigor := NewService(nil, serviceCharacterReader{character: &domainchar.Character{CurrentSP: 1}}, serviceSkillReader{level: 3}, nil, catalog)
	if _, err := lowVigor.PrepareHerb(context.Background(), 8, 1243, true); !errors.Is(err, ErrVigorNotEnough) {
		t.Fatalf("low-vigor PrepareHerb() error = %v", err)
	}
}

func TestCompleteDelegatesPreparedCommand(t *testing.T) {
	repository := &serviceCommandRepository{result: &domaingathering.CompletionResult{Succeeded: true}}
	service := NewService(repository, nil, nil, nil, nil)
	command := domaingathering.CompletionCommand{CharacterID: 5, Kind: domaingathering.ResourceKindFishing, NPCID: 1189, Succeeded: true}

	result, err := service.Complete(context.Background(), command)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if result != repository.result || repository.command != command {
		t.Fatalf("Complete() did not delegate command: %+v", repository.command)
	}
}
