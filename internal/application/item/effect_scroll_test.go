package item

import (
	"context"
	"testing"
	"time"

	appbuff "mcgame-server/internal/application/buff"
	domainbuff "mcgame-server/internal/domain/buff"
	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

type summonBookBuffService struct {
	request appbuff.AddRequest
}

func (s *summonBookBuffService) AddOrRefresh(_ context.Context, characterID int64, request appbuff.AddRequest) (*domainbuff.Buff, error) {
	s.request = request
	return &domainbuff.Buff{ID: 1, CharacterID: characterID, BuffID: request.BuffID}, nil
}

func TestSummonBookConfigsMatchFlashBuffs(t *testing.T) {
	tests := []struct {
		templateID int
		buffID     int
		duration   time.Duration
		quality    int
	}{
		{templateID: 2514, buffID: 1709, duration: time.Hour, quality: 3},
		{templateID: 2543, buffID: 1709, duration: 30 * 24 * time.Hour, quality: 3},
		{templateID: 2798, buffID: 1979, duration: 30 * 24 * time.Hour, quality: 2},
	}

	for _, test := range tests {
		config, ok := GetSummonBookConfig(test.templateID)
		if !ok || config.BuffID != test.buffID || config.Duration != test.duration || config.MinQuality != test.quality {
			t.Fatalf("template %d config = %+v/%v", test.templateID, config, ok)
		}
	}
	if _, ok := GetSummonBookConfig(128); ok {
		t.Fatal("secret passage book was classified as summon book")
	}
}

func TestSummonBookEffectGrantsDisplayBuff(t *testing.T) {
	buffs := &summonBookBuffService{}
	service := &Service{buffService: buffs}
	handler := &scrollEffectHandler{service: service}
	tpl := &models.ItemTemplateTemplate{ID: 2514, Type: 500, UseType: 1, Name: "Sách Triệu Hồi"}

	result, err := handler.Apply(context.Background(), &UseContext{TargetType: 1}, &domainchar.Character{ID: 9}, &domainitem.Item{StackCount: 1}, tpl)
	if err != nil {
		t.Fatal(err)
	}
	if buffs.request.BuffID != 1709 || buffs.request.Duration != time.Hour || len(result.GrantedBuffs) != 1 {
		t.Fatalf("request/result = %+v/%+v", buffs.request, result)
	}
}
