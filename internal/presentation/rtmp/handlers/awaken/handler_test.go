// Open-sourced by BaoLT

package awaken

import (
	"context"
	"testing"

	domainchar "mcgame-server/internal/domain/character"

	"go.uber.org/zap"
)

type testCharacterLoader struct {
	char *domainchar.Character
}

func (l *testCharacterLoader) FindByID(_ context.Context, _ int64) (*domainchar.Character, error) {
	return l.char, nil
}

type testBonusProvider struct {
	bonuses domainchar.EquipmentStatBonuses
}

func (p *testBonusProvider) AggregateEquipmentStats(_ context.Context, _ int64) domainchar.EquipmentStatBonuses {
	return p.bonuses
}

type testCallbackSender struct {
	method  string
	payload map[string]interface{}
}

func (s *testCallbackSender) SendCallback(method string, args ...interface{}) error {
	s.method = method
	if len(args) > 0 {
		s.payload, _ = args[0].(map[string]interface{})
	}
	return nil
}

func TestAwakeningSucceededOnlyAcceptsSuccessfulResult(t *testing.T) {
	if !awakeningSucceeded(map[string]interface{}{"success": true}) {
		t.Fatal("successful result was not detected")
	}
	if awakeningSucceeded(map[string]interface{}{"success": false}) || awakeningSucceeded(map[string]interface{}{"needGold": 10}) {
		t.Fatal("failed or needGold result must not trigger stat refresh")
	}
}

func TestSendStatRefreshBuildsEquipmentAwareUPPPayload(t *testing.T) {
	char := &domainchar.Character{ID: 7, ClassID: 1, Strength: 10, Agility: 10, Stamina: 10, Intelligence: 10, Spirit: 10}
	char.RecalculateStats()
	bonuses := domainchar.NewEquipmentStatBonuses()
	bonuses.AddFlat(domainchar.PropMaxHP, 250)
	handler := &Handler{
		characterRepo: &testCharacterLoader{char: char},
		bonusProvider: &testBonusProvider{bonuses: bonuses},
		logger:        zap.NewNop(),
	}
	sender := &testCallbackSender{}

	handler.sendStatRefresh(context.Background(), sender, char.ID)

	if sender.method != "onUPP" {
		t.Fatalf("callback method = %q, want onUPP", sender.method)
	}
	if got, want := sender.payload["hpMax"], char.MaxHP+250; got != want {
		t.Fatalf("hpMax = %v, want %d", got, want)
	}
}
