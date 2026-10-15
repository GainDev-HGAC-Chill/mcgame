package combat

import (
	"testing"

	domcombat "mcgame-server/internal/domain/combat"
)

func TestDetectBossKillsUsesExplicitScheduleContext(t *testing.T) {
	service := &Service{}
	battle := domcombat.NewBattle(domcombat.BattleTypePVE, 3)
	battle.BossCtx = &domcombat.BossContext{BossNID: 706, Source: "schedule", Tier: "normal"}
	battle.Participants = append(battle.Participants, &domcombat.Participant{
		Side:       domcombat.SideEnemy,
		EntityType: domcombat.ParticipantTypeCreature,
		EntityID:   348,
		IsAlive:    false,
	})

	kills := service.detectBossKills(battle)
	if len(kills) != 1 {
		t.Fatalf("kills = %+v, want one explicit schedule kill", kills)
	}
	if kills[0].BossNID != 706 || kills[0].Source != "schedule" || kills[0].Tier != "normal" {
		t.Fatalf("kill = %+v", kills[0])
	}
}
