// Open-sourced by BaoLT

package utils

import (
	"testing"

	domainchar "mcgame-server/internal/domain/character"
)

type expTestSender struct {
	calls []expTestCall
}

func TestSendExpAndLevelUpCallbacks_OnlySendsSupportedCallbacks(t *testing.T) {
	for _, leveledUp := range []bool{false, true} {
		for _, equipment := range []bool{false, true} {
			char := &domainchar.Character{ID: 17, Level: 2, Experience: 12, AttrPoints: 5}
			char.RecalculateStats()
			sender := &expTestSender{}
			if equipment {
				bonuses := domainchar.NewEquipmentStatBonuses()
				SendExpAndLevelUpCallbacksWithEquipment(sender, char, 12, leveledUp, &bonuses)
			} else {
				SendExpAndLevelUpCallbacks(sender, char, 12, leveledUp)
			}
			want := []string{"onUPP"}
			if leveledUp {
				want = append(want, "onPlayerLevelUp")
			}
			if len(sender.calls) != len(want) {
				t.Fatalf("leveledUp=%v equipment=%v: got %d callbacks, want %d", leveledUp, equipment, len(sender.calls), len(want))
			}
			for i, call := range sender.calls {
				if call.method != want[i] || len(call.args) != 1 {
					t.Fatalf("callback %d: method=%s args=%d", i, call.method, len(call.args))
				}
			}
			if leveledUp {
				payload, ok := sender.calls[1].args[0].(map[string]interface{})
				if !ok || payload["id"] != "17" || payload["level"] != 2 || payload["lp"] != "5" {
					t.Fatalf("level-up payload = %#v", sender.calls[1].args[0])
				}
			}
		}
	}
}

type expTestCall struct {
	method string
	args   []interface{}
}

func (s *expTestSender) SendCallback(method string, args ...interface{}) error {
	copiedArgs := make([]interface{}, len(args))
	copy(copiedArgs, args)
	s.calls = append(s.calls, expTestCall{method: method, args: copiedArgs})
	return nil
}

func (s *expTestSender) AddExpSkill(amount int64) int64 {
	return amount
}

func TestSendExpAndLevelUpCallbacksWithEquipment_UsesEffectiveMaxStats(t *testing.T) {
	char := &domainchar.Character{
		Level:        10,
		Strength:     30,
		Agility:      18,
		Stamina:      20,
		Intelligence: 15,
		Spirit:       12,
	}
	char.RecalculateStats()
	effectiveMaxHP := char.MaxHP + 1000
	char.CurrentHP = effectiveMaxHP

	bonuses := domainchar.NewEquipmentStatBonuses()
	bonuses.AddFlat(domainchar.PropMaxHP, 1000)

	sender := &expTestSender{}
	SendExpAndLevelUpCallbacksWithEquipment(sender, char, 0, false, &bonuses)

	if len(sender.calls) == 0 {
		t.Fatal("no callbacks sent")
	}

	payload, ok := sender.calls[0].args[0].(map[string]interface{})
	if !ok {
		t.Fatalf("callback payload = %T, want map[string]interface{}", sender.calls[0].args[0])
	}
	if got := payload["currentHp"]; got != effectiveMaxHP {
		t.Fatalf("currentHp = %v, want %d", got, effectiveMaxHP)
	}

	property, ok := payload["property"].(map[string]interface{})
	if !ok {
		t.Fatalf("property payload = %T, want map[string]interface{}", payload["property"])
	}
	if got := property["finalHp"]; got != effectiveMaxHP {
		t.Fatalf("property.finalHp = %v, want %d", got, effectiveMaxHP)
	}
	if got := property["hpMax"]; got != effectiveMaxHP {
		t.Fatalf("property.hpMax = %v, want %d", got, effectiveMaxHP)
	}
}
