// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"

	domaincharacter "mcgame-server/internal/domain/character"

	"github.com/google/uuid"
)

type finalStatsBonusProviderFixture struct {
	bonuses domaincharacter.EquipmentStatBonuses
}

func (p finalStatsBonusProviderFixture) AggregateEquipmentStats(ctx context.Context, characterID int64) domaincharacter.EquipmentStatBonuses {
	return p.bonuses
}

func TestFinalStatsStateReaderUsesEffectiveCharacterStats(t *testing.T) {
	character := domaincharacter.NewCharacter(uuid.New(), "final-stats", 1, 1)
	character.ID = 99
	baseDefense := character.Defense
	bonuses := domaincharacter.NewEquipmentStatBonuses()
	bonuses.AddFlat(domaincharacter.PropDefense, 40000)
	bonuses.AddFlat(domaincharacter.PropMagicDef, 40000)
	bonuses.AddFlat(domaincharacter.PropSpeed, 15000)
	reader := NewFinalStatsStateReader(
		characterStateRepositoryFixture{character: character},
		finalStatsBonusProviderFixture{bonuses: bonuses},
	)
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	for _, proof := range proofs {
		if !proof.Done {
			t.Fatalf("proof = %#v, want done", proof)
		}
	}
	if character.Defense != baseDefense {
		t.Fatalf("source character defense mutated: got %d want %d", character.Defense, baseDefense)
	}
}
