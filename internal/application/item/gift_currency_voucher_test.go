package item

import (
	"context"
	"encoding/json"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/gamedata/models"
)

func giftCurrencyVoucherService(t *testing.T, templateID, stack, reqLevel int) (*Service, *itemServiceTestRepo) {
	t.Helper()
	svc, repo := capacityTestService(t, templateID, stack)
	raw, err := json.Marshal(models.ItemTemplateTemplate{ID: int64(templateID), Type: 550, UseType: 1, ReqLevel: float64(reqLevel)})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.gameDataRec.GetCache().LoadTable(models.TableItemTemplate, []json.RawMessage{raw}); err != nil {
		t.Fatal(err)
	}
	return svc, repo
}

func TestGiftCurrencyVouchersFollowDescriptions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		templateID int
		level      int
		clientKey  string
		amount     int64
	}{
		{"silver gift", silverGiftVoucherTemplateID, 1, "moneyBind", openSilverGiftAmount},
		{"gold gift", goldGiftVoucherTemplateID, 50, "goldBind", goldGiftAmount},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := giftCurrencyVoucherService(t, tc.templateID, 2, tc.level)
			char := &domainchar.Character{ID: 99, Level: tc.level, MoneyBind: 7, GoldBind: 11}
			_, result, err := svc.UseItem(context.Background(), char, 88, &UseContext{TargetType: 1})
			if err != nil {
				t.Fatal(err)
			}
			grants := result["granted_currencies"].([]GrantedCurrencyResult)
			if len(grants) != 1 || grants[0].ClientKey != tc.clientKey || grants[0].Delta != tc.amount {
				t.Fatalf("grants = %#v", grants)
			}
			if repo.items[88].StackCount != 1 {
				t.Fatalf("remaining stack = %d", repo.items[88].StackCount)
			}
			if tc.clientKey == "moneyBind" && (char.MoneyBind != 7+tc.amount || char.GoldBind != 11) {
				t.Fatalf("wallet = moneyBind %d goldBind %d", char.MoneyBind, char.GoldBind)
			}
			if tc.clientKey == "goldBind" && (char.GoldBind != 11+tc.amount || char.MoneyBind != 7) {
				t.Fatalf("wallet = moneyBind %d goldBind %d", char.MoneyBind, char.GoldBind)
			}
		})
	}
}

func TestGoldCardAwardsOnlyAdvertisedAmounts(t *testing.T) {
	svc, repo := giftCurrencyVoucherService(t, goldCardTemplateID, 20, 1)
	char := &domainchar.Character{ID: 99, Level: 1}
	allowed := make(map[int64]bool, len(goldCardAmounts))
	for _, amount := range goldCardAmounts {
		allowed[amount] = true
	}
	for i := 0; i < 20; i++ {
		before := char.GoldBind
		_, result, err := svc.UseItem(context.Background(), char, 88, &UseContext{TargetType: 1})
		if err != nil {
			t.Fatal(err)
		}
		grant := result["granted_currencies"].([]GrantedCurrencyResult)[0]
		if grant.ClientKey != "goldBind" || !allowed[grant.Delta] || char.GoldBind != before+grant.Delta {
			t.Fatalf("grant = %#v wallet = %d", grant, char.GoldBind)
		}
	}
	if repo.items[88] != nil {
		t.Fatalf("exhausted card stack still exists: %#v", repo.items[88])
	}
}

func TestGiftCurrencyVoucherRejectsWrongTargetWithoutConsumption(t *testing.T) {
	svc, repo := giftCurrencyVoucherService(t, goldGiftVoucherTemplateID, 1, 50)
	char := &domainchar.Character{ID: 99, Level: 50}
	if _, _, err := svc.UseItem(context.Background(), char, 88, &UseContext{TargetType: 2}); err == nil {
		t.Fatal("expected target rejection")
	}
	if repo.items[88] == nil || repo.items[88].StackCount != 1 || char.GoldBind != 0 {
		t.Fatalf("state changed: item=%#v goldBind=%d", repo.items[88], char.GoldBind)
	}
}

func TestGiftCurrencyVoucherMultiUseCreditsWholeCount(t *testing.T) {
	svc, repo := giftCurrencyVoucherService(t, goldGiftVoucherTemplateID, 3, 50)
	char := &domainchar.Character{ID: 99, Level: 50, GoldBind: 9}
	result, err := svc.UseMultiItem(context.Background(), char, 88, 2, 1, -1)
	if err != nil {
		t.Fatal(err)
	}
	grant := result["granted_currencies"].([]GrantedCurrencyResult)[0]
	if grant.ClientKey != "goldBind" || grant.Delta != 100 || char.GoldBind != 109 {
		t.Fatalf("grant = %#v wallet = %d", grant, char.GoldBind)
	}
	if repo.items[88].StackCount != 1 {
		t.Fatalf("remaining stack = %d", repo.items[88].StackCount)
	}
}
