package item

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/gamedata/models"
)

func silverVoucherService(t *testing.T, stack int) (*Service, *itemServiceTestRepo) {
	t.Helper()
	svc, repo := capacityTestService(t, 1132, stack)
	raw, err := json.Marshal(models.ItemTemplateTemplate{ID: 1132, Type: 550, UseType: 1, ReqLevel: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.gameDataRec.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	return svc, repo
}

func TestSilverVoucherCurrencyFollowsInstanceBinding(t *testing.T) {
	for _, bound := range []bool{false, true} {
		svc, repo := silverVoucherService(t, 2)
		repo.items[88].IsBound = bound
		char := &domainchar.Character{ID: 99, Level: 1, Money: 7, MoneyBind: 9, Gold: 11}
		_, result, err := svc.UseItem(context.Background(), char, 88, &UseContext{TargetType: 1})
		if err != nil {
			t.Fatal(err)
		}
		key := "money"
		if bound {
			key = "moneyBind"
			if char.Money != 7 || char.MoneyBind != 100009 {
				t.Fatal("bound voucher credited wrong wallet")
			}
		} else if char.Money != 100007 || char.MoneyBind != 9 {
			t.Fatal("unbound voucher credited wrong wallet")
		}
		grants := result["granted_currencies"].([]GrantedCurrencyResult)
		if len(grants) != 1 || grants[0].ClientKey != key || grants[0].Delta != 100000 || repo.items[88].StackCount != 1 || char.Gold != 11 {
			t.Fatalf("wrong voucher result: %v", result)
		}
	}
}

func TestSilverVoucherMultiUseAndFailures(t *testing.T) {
	svc, repo := silverVoucherService(t, 3)
	char := &domainchar.Character{ID: 99, Level: 1}
	if _, err := svc.UseMultiItem(context.Background(), char, 88, 3, 1, -1); err != nil {
		t.Fatal(err)
	}
	if char.Money != 300000 || repo.items[88] != nil {
		t.Fatal("multi-use did not consume/credit exact quantity")
	}
	for _, kind := range []string{"target", "level", "overflow", "item-write"} {
		svc, repo := silverVoucherService(t, 1)
		char := &domainchar.Character{ID: 99, Level: 1}
		uctx := &UseContext{TargetType: 1}
		switch kind {
		case "target":
			uctx.TargetType = 2
		case "level":
			char.Level = 0
		case "overflow":
			char.Money = math.MaxInt64
		case "item-write":
			svc.itemRepo = capacityWriteFailureRepo{Repository: repo, err: errors.New("write failure")}
		}
		before := char.Money
		if _, _, err := svc.UseItem(context.Background(), char, 88, uctx); err == nil {
			t.Fatalf("%s unexpectedly succeeded", kind)
		}
		if char.Money != before || char.MoneyBind != 0 || repo.items[88] == nil || repo.items[88].StackCount != 1 {
			t.Fatalf("%s changed wallet/item on failure", kind)
		}
	}
}
