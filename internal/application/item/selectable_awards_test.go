package item

import (
	"context"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func selectableAward(id int64, itemID int, awardID int, awardType int, order int, getNum int) *models.ItemAwardTemplate {
	return &models.ItemAwardTemplate{
		ID:      id,
		ItemID:  float64(itemID),
		AwardID: float64(awardID),
		Type:    float64(awardType),
		Count:   1,
		Payload: map[string]interface{}{
			models.AwardPayloadSelectionKey: map[string]interface{}{
				"order":  float64(order),
				"getNum": float64(getNum),
			},
		},
	}
}

func TestBuildSelectableGiftBoxOrdersConfiguredRows(t *testing.T) {
	first := selectableAward(1, 3376, 1676, 19, 0, 1)
	first.Count = 2
	first.Quality = 8
	first.PreNameType = 3
	first.Payload["bound"] = true
	box, configured, err := buildSelectableGiftBox([]*models.ItemAwardTemplate{
		selectableAward(3, 3376, 1677, 19, 2, 1),
		first,
		selectableAward(2, 3376, 1678, 19, 1, 1),
	})
	if err != nil || !configured {
		t.Fatalf("buildSelectableGiftBox() configured=%v error=%v", configured, err)
	}
	if box.GetNum != 1 || len(box.Choices) != 3 {
		t.Fatalf("gift box = %#v", box)
	}
	want := []int{1676, 1678, 1677}
	for index, choice := range box.Choices {
		if choice.ID != want[index] || choice.AwardRowID != int64(index+1) {
			t.Fatalf("choice %d = %#v, want reward %d", index, choice, want[index])
		}
		if _, exists := choice.Payload[models.AwardPayloadSelectionKey]; exists {
			t.Fatalf("choice %d retained reserved selection payload", index)
		}
	}
	if box.Choices[0].StackNum != 2 || box.Choices[0].Quality != 8 || box.Choices[0].PreNameType != 3 || box.Choices[0].Payload["bound"] != true {
		t.Fatalf("first choice metadata = %#v", box.Choices[0])
	}
}

func TestBuildSelectableGiftBoxRejectsInvalidDefinitions(t *testing.T) {
	tests := map[string][]*models.ItemAwardTemplate{
		"mixed": {
			selectableAward(1, 3376, 1676, 19, 0, 1),
			{ID: 2, ItemID: 3376, AwardID: 1678, Type: 19, Count: 1},
		},
		"duplicate order": {
			selectableAward(1, 3376, 1676, 19, 0, 1),
			selectableAward(2, 3376, 1678, 19, 0, 1),
		},
		"inconsistent count": {
			selectableAward(1, 3376, 1676, 19, 0, 1),
			selectableAward(2, 3376, 1678, 19, 1, 2),
		},
		"unsupported type": {
			selectableAward(1, 3376, 1, 30, 0, 1),
		},
		"malformed metadata": {
			{
				ID: 1, ItemID: 3376, AwardID: 1676, Type: 19, Count: 1,
				Payload: map[string]interface{}{models.AwardPayloadSelectionKey: "invalid"},
			},
		},
	}
	for name, awards := range tests {
		t.Run(name, func(t *testing.T) {
			if _, configured, err := buildSelectableGiftBox(awards); !configured || err == nil {
				t.Fatalf("configured=%v error=%v, want configured failure", configured, err)
			}
		})
	}
}

func TestUseItemConfiguredSelectableAwardsBypassPicker(t *testing.T) {
	logger := zap.NewNop()
	repo := newItemServiceTestRepo()
	service := NewService(repo, logger)
	manager := gamedata.NewManager(nil, logger)
	loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: 3376, Name: "Túi Quà Cánh Xanh Dương", Type: 550, UseType: 1})
	loadAwardTemplate(t, manager, *selectableAward(2, 3376, 1678, 19, 1, 1))
	loadAwardTemplate(t, manager, *selectableAward(1, 3376, 1676, 19, 0, 1))
	loadAwardTemplate(t, manager, *selectableAward(3, 3376, 1677, 19, 2, 1))
	service.SetGameDataManager(manager)

	bag := &domainitem.Item{ID: 9, CharacterID: 7, TemplateID: 3376, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 1}
	repo.items[bag.ID] = bag
	used, result, err := service.UseItem(context.Background(), &domainchar.Character{ID: 7, Level: 80}, bag.ID, nil)
	if err != nil {
		t.Fatalf("UseItem() error = %v", err)
	}
	if used.StackCount != 1 || repo.items[bag.ID] == nil {
		t.Fatal("selectable package was consumed while opening the panel")
	}
	box, ok := result["gift_box"].(*GiftBoxData)
	if !ok || len(box.Choices) != 3 || box.Choices[0].ID != 1676 || box.Choices[1].ID != 1678 || box.Choices[2].ID != 1677 {
		t.Fatalf("gift_box = %#v", result["gift_box"])
	}
}

func TestUseMultiItemRejectsConfiguredSelectableAwards(t *testing.T) {
	logger := zap.NewNop()
	repo := newItemServiceTestRepo()
	service := NewService(repo, logger)
	manager := gamedata.NewManager(nil, logger)
	loadItemTemplate(t, manager, models.ItemTemplateTemplate{ID: 3376, Name: "Túi Quà Cánh Xanh Dương", Type: 550, UseType: 1})
	loadAwardTemplate(t, manager, *selectableAward(1, 3376, 1676, 19, 0, 1))
	service.SetGameDataManager(manager)
	bag := &domainitem.Item{ID: 9, CharacterID: 7, TemplateID: 3376, ItemType: domainitem.ItemTypeConsumable, SlotType: domainitem.SlotTypeBag, StackCount: 2}
	repo.items[bag.ID] = bag

	if _, err := service.UseMultiItem(context.Background(), &domainchar.Character{ID: 7, Level: 80}, bag.ID, 2, 1, -1); err == nil {
		t.Fatal("UseMultiItem() accepted an interactive selectable package")
	}
	if repo.items[bag.ID].StackCount != 2 {
		t.Fatalf("stack = %d, want 2", repo.items[bag.ID].StackCount)
	}
}

func TestGrantSelectablePetAwardFailsWithoutPetService(t *testing.T) {
	service := NewService(newItemServiceTestRepo(), zap.NewNop())
	service.awardHandler = &awardEffectHandler{itemService: service}
	_, err := service.GrantSelectableAward(
		context.Background(),
		&domainchar.Character{ID: 7},
		&GiftBoxChoice{AwardRowID: 1, TableType: 12, ID: 91, StackNum: 1},
	)
	if err == nil {
		t.Fatal("GrantSelectableAward() succeeded without a Pet service")
	}
}
