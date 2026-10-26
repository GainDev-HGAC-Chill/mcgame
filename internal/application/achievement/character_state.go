// Open-sourced by BaoLT

// Character state reader proves achievement conditions from persisted character state.
package achievement

import (
	"context"
	"fmt"

	domaincharacter "mcgame-server/internal/domain/character"
)

type CharacterStateRepository interface {
	FindByID(ctx context.Context, id int64) (*domaincharacter.Character, error)
}

type CharacterStateReader struct {
	repo CharacterStateRepository
}

func NewCharacterStateReader(repo CharacterStateRepository) *CharacterStateReader {
	return &CharacterStateReader{repo: repo}
}

func (r *CharacterStateReader) Name() string {
	return "character"
}

func (r *CharacterStateReader) RequirementIDs() []int64 {
	return []int64{1, 2, 3, 4, 17, 18, 19, 20, 22, 57, 58, 100, 101, 134, 135, 136, 137, 142, 143, 144, 145, 146, 147, 148, 149, 386, 387, 392, 393, 394, 395, 510, 535, 652, 653, 654, 655, 656, 657, 658}
}

func (r *CharacterStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil {
		return nil, fmt.Errorf("character state repository not configured")
	}
	character, err := r.repo.FindByID(ctx, characterID)
	if err != nil {
		return nil, err
	}
	if character == nil || character.ID != characterID {
		return nil, fmt.Errorf("character %d not found", characterID)
	}

	proofs := []StateProof{
		stateProof(1, character.Level, 10),
		stateProof(2, character.Level, 50),
		stateProof(3, character.Level, 100),
		stateProof(4, character.Level, 140),
		stateProof(17, boundedStateProgress(character.Money), 8888),
		stateProof(18, boundedStateProgress(character.Money), 888888),
		stateProof(19, boundedStateProgress(character.Money), 888888888),
		stateProof(20, boundedStateProgress(character.Gold), 28888),
		stateProof(57, character.BagSlotNum, 1),
		stateProof(58, character.BankSlotNum, 1),
		stateProof(100, character.PetMaxNum, 1),
		stateProof(101, character.PetMaxNum, 30),
		stateProof(134, character.ArenaPoints, 10000),
		stateProof(135, character.ArenaPoints, 30000),
		stateProof(136, character.ArenaPoints, 100000),
		stateProof(137, character.ArenaPoints, 200000),
		stateProof(142, boundedStateProgress(character.Pop), 500),
		stateProof(143, boundedStateProgress(character.Pop), 2000),
		stateProof(144, boundedStateProgress(character.Pop), 5000),
		stateProof(145, boundedStateProgress(character.Pop), 10000),
		stateProof(146, boundedStateProgress(character.Reputation), 500),
		stateProof(147, boundedStateProgress(character.Reputation), 2000),
		stateProof(148, boundedStateProgress(character.Reputation), 5000),
		stateProof(149, boundedStateProgress(character.Reputation), 10000),
		stateProof(386, character.TempBagSlots, 1),
		stateProof(387, character.TempBagSlots, 5),
		stateProof(392, negativeThresholdProgress(character.Pop, -500), 1),
		stateProof(393, negativeThresholdProgress(character.Pop, -5000), 1),
		stateProof(394, negativeThresholdProgress(character.Pop, -10000), 1),
		stateProof(395, character.PetMaxNum, 60),
		stateProof(510, character.Level, 150),
		stateProof(535, character.RebirthLvl, 1),
		stateProof(652, character.AwakenLevel, 11),
		stateProof(653, character.AwakenLevel, 22),
		stateProof(654, character.AwakenLevel, 33),
		stateProof(655, character.AwakenLevel, 44),
		stateProof(656, character.AwakenLevel, 55),
		stateProof(657, character.AwakenLevel, 66),
		stateProof(658, character.AwakenLevel, 77),
	}
	if character.Level >= 60 && character.Money == 0 && character.MoneyBind == 0 && character.Gold == 0 && character.GoldBind == 0 {
		proofs = append(proofs, stateProof(22, 1, 1))
	}
	return proofs, nil
}

func negativeThresholdProgress(value, threshold int64) int {
	if value <= threshold {
		return 1
	}
	return 0
}

func stateProof(requirementID int64, progress, target int) StateProof {
	if progress < 0 {
		progress = 0
	}
	return StateProof{
		RequirementID: requirementID,
		Progress:      progress,
		Target:        target,
		Done:          progress >= target,
	}
}

func boundedStateProgress(value int64) int {
	if value <= 0 {
		return 0
	}
	maxInt := int(^uint(0) >> 1)
	if value > int64(maxInt) {
		return maxInt
	}
	return int(value)
}
