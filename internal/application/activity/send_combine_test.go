package activity

import (
	"context"
	"testing"

	domainactivity "mcgame-server/internal/domain/activity"
	domainitem "mcgame-server/internal/domain/item"
)

type sendCombineRepositoryFixture struct {
	command domainactivity.SendCombinePurchaseCommand
}

func (r *sendCombineRepositoryFixture) PurchaseSendCombine(ctx context.Context, command domainactivity.SendCombinePurchaseCommand) (*domainactivity.SendCombinePurchaseResult, error) {
	r.command = command
	return &domainactivity.SendCombinePurchaseResult{BuyCount: 1, Gold: 10}, nil
}

func (r *sendCombineRepositoryFixture) ListSendCombineBuyCounts(ctx context.Context, characterID int64, eventEndMs int64) (map[string]int, error) {
	return map[string]int{"0": 2}, nil
}

func TestSendCombineServiceValidatesAndDelegates(t *testing.T) {
	repository := &sendCombineRepositoryFixture{}
	service := NewSendCombineService(repository)
	command := domainactivity.SendCombinePurchaseCommand{CharacterID: 3, ActivityID: "0", EventEndMs: 10, Limit: 5, Cost: 7, Awards: []domainactivity.SendCombineAward{{TemplateID: 6861, ItemType: domainitem.ItemTypeConsumable, Count: 10}}}
	result, err := service.Purchase(context.Background(), command)
	if err != nil {
		t.Fatalf("Purchase() error = %v", err)
	}
	if result.BuyCount != 1 || repository.command.ActivityID != "0" {
		t.Fatalf("unexpected result/command: result=%+v command=%+v", result, repository.command)
	}
	command.Awards[0].Count = 0
	if _, err := service.Purchase(context.Background(), command); err == nil {
		t.Fatal("Purchase() accepted invalid award")
	}
	counts, err := service.BuyCounts(context.Background(), 3, 10)
	if err != nil || counts["0"] != 2 {
		t.Fatalf("BuyCounts() = %v, %v", counts, err)
	}
}
