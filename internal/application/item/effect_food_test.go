// Open-sourced by BaoLT

package item

import (
	"context"
	"errors"
	"testing"
	"time"

	appbuff "mcgame-server/internal/application/buff"
	domainbuff "mcgame-server/internal/domain/buff"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

type foodBuffService struct {
	request appbuff.AddRequest
}

func (s *foodBuffService) AddOrRefresh(_ context.Context, characterID int64, request appbuff.AddRequest) (*domainbuff.Buff, error) {
	s.request = request
	return &domainbuff.Buff{ID: 99, CharacterID: characterID, BuffID: request.BuffID, BuffType: request.BuffType}, nil
}

func TestFoodEffectHandlerGrantsCatalogCakeBuffs(t *testing.T) {
	for _, test := range []struct {
		itemID   int
		buffID   int
		duration time.Duration
	}{
		{itemID: 1297, buffID: 772, duration: 15 * time.Minute},
		{itemID: 3916, buffID: 2709, duration: time.Hour},
		{itemID: 4584, buffID: 2858, duration: 30 * time.Minute},
	} {
		t.Run(time.Duration(test.itemID).String(), func(t *testing.T) {
			buffs := &foodBuffService{}
			service := &Service{buffService: buffs}
			handler := &foodEffectHandler{itemService: service}
			char := &domainchar.Character{ID: 7}
			item := &domainitem.Item{ID: 8, CharacterID: 7, TemplateID: test.itemID, StackCount: 1}
			template := &models.ItemTemplateTemplate{ID: int64(test.itemID), Type: 502, UseType: 1, Name: "Bánh"}

			result, err := handler.Apply(context.Background(), &UseContext{TargetType: 1}, char, item, template)
			if err != nil {
				t.Fatalf("Apply() error = %v", err)
			}
			if buffs.request.BuffID != test.buffID || buffs.request.Duration != test.duration {
				t.Fatalf("buff request = %#v", buffs.request)
			}
			if len(result.GrantedBuffs) != 1 || result.GrantedBuffs[0].BuffID != test.buffID {
				t.Fatalf("granted buffs = %#v", result.GrantedBuffs)
			}
			if !result.RefreshCharacterView || result.Message == "" {
				t.Fatalf("result = %#v", result)
			}
		})
	}
}

func TestFoodEffectHandlerRejectsZeroEffectFoodWithoutConsuming(t *testing.T) {
	handler := &foodEffectHandler{itemService: &Service{buffService: &foodBuffService{}}}
	_, err := handler.Apply(
		context.Background(),
		nil,
		&domainchar.Character{ID: 1},
		&domainitem.Item{ID: 2, TemplateID: 1681, StackCount: 1},
		&models.ItemTemplateTemplate{ID: 1681, Type: 502, UseType: 1, Name: "Bánh Bông Gòn"},
	)
	if !errors.Is(err, pkgerrors.ErrCannotUseItem) {
		t.Fatalf("error = %v, want cannot use item", err)
	}
}

func TestFoodBuffCatalogCoversKnownCakeFamilies(t *testing.T) {
	if len(foodBuffConfigs) != 33 {
		t.Fatalf("food buff config count = %d, want 33", len(foodBuffConfigs))
	}
	for _, itemID := range []int{1297, 1301, 1519, 1523, 1676, 1680, 1856, 1860, 2051, 2708, 3912, 3916, 4151, 4153, 4580, 4584} {
		config, ok := foodBuffConfigs[itemID]
		if !ok || config.buffID <= 0 || config.duration <= 0 {
			t.Fatalf("invalid food buff config for item %d: %#v", itemID, config)
		}
	}
}
