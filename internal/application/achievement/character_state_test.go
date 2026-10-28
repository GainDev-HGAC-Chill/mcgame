// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"

	domaincharacter "mcgame-server/internal/domain/character"
)

type characterStateRepositoryFixture struct {
	character *domaincharacter.Character
	err       error
}

func (r characterStateRepositoryFixture) FindByID(ctx context.Context, id int64) (*domaincharacter.Character, error) {
	return r.character, r.err
}

func TestCharacterStateReaderBuildsPersistedProofs(t *testing.T) {
	character := &domaincharacter.Character{
		ID:           99,
		Level:        150,
		RebirthLvl:   2,
		Money:        900000000,
		Gold:         30000,
		BagSlotNum:   2,
		BankSlotNum:  1,
		TempBagSlots: 5,
		PetMaxNum:    60,
		ArenaPoints:  200000,
		Honor:        200,
		Pop:          10000,
		Reputation:   10000,
		AwakenLevel:  77,
	}
	reader := NewCharacterStateReader(characterStateRepositoryFixture{character: character})
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 43 {
		t.Fatalf("proof count = %d, want 43", len(proofs))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{1, 4, 17, 20, 57, 58, 100, 101, 134, 137, 138, 139, 140, 141, 142, 149, 386, 387, 395, 510, 535, 652, 658} {
		proof, ok := byID[requirementID]
		if !ok || !proof.Done {
			t.Fatalf("requirement %d proof = %#v, found=%t", requirementID, proof, ok)
		}
	}
	if _, exists := byID[22]; exists {
		t.Fatalf("zero-wallet requirement should be absent for non-zero wallet")
	}
}

func TestCharacterStateReaderProvesNegativePopularityMilestones(t *testing.T) {
	character := &domaincharacter.Character{ID: 99, Pop: -5000}
	proofs, err := NewCharacterStateReader(characterStateRepositoryFixture{character: character}).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	if !byID[392].Done || !byID[393].Done || byID[394].Done {
		t.Fatalf("negative popularity proofs = %#v/%#v/%#v", byID[392], byID[393], byID[394])
	}
}

func TestCharacterStateReaderProvesPKPointMilestones(t *testing.T) {
	character := &domaincharacter.Character{ID: 99, Honor: 25}
	proofs, err := NewCharacterStateReader(characterStateRepositoryFixture{character: character}).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	if !byID[138].Done || !byID[139].Done || byID[140].Done || byID[141].Done {
		t.Fatalf("PK proofs = %#v/%#v/%#v/%#v", byID[138], byID[139], byID[140], byID[141])
	}
}

func TestCharacterStateReaderProvesZeroWalletOnlyAtRequiredLevel(t *testing.T) {
	character := &domaincharacter.Character{ID: 99, Level: 60, BagSlotNum: 1, BankSlotNum: 1, PetMaxNum: 6}
	reader := NewCharacterStateReader(characterStateRepositoryFixture{character: character})
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	found := false
	for _, proof := range proofs {
		if proof.RequirementID == 22 {
			found = proof.Done && proof.Progress == 1 && proof.Target == 1
		}
	}
	if !found {
		t.Fatalf("zero-wallet proof not found: %#v", proofs)
	}

	character.Level = 59
	proofs, err = reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() second error = %v", err)
	}
	for _, proof := range proofs {
		if proof.RequirementID == 22 {
			t.Fatalf("zero-wallet proof emitted below level 60")
		}
	}
}

var _ CharacterStateRepository = characterStateRepositoryFixture{}
