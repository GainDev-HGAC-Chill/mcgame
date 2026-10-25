package achievement

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	domainfeature "mcgame-server/internal/domain/statfeature"
)

func TestPetContractStateReaderUsesPersistedLevels(t *testing.T) {
	repo := characterFeatureStateRepositoryFixture{states: []*domainfeature.CharacterFeatureState{{
		CharacterID: 99,
		FeatureKey:  domainfeature.FeatureContractPet,
		State: map[string]interface{}{
			"hp": 25.0, "hpExp": 90.0,
			"speed": 24.0, "speedExp": 999999.0,
			"physics": 50.0,
			"magic":   49.0,
		},
	}}}
	proofs, err := NewPetContractStateReader(repo).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 8 {
		t.Fatalf("proof count = %d, want 8", len(proofs))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{667, 669, 670, 673} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{668, 671, 672, 674} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
	if byID[671].Progress != 25 || byID[671].Target != 50 {
		t.Fatalf("requirement 671 proof = %#v, want progress 25/50", byID[671])
	}
}

func TestPetContractStateReaderRejectsInvalidOrDuplicateState(t *testing.T) {
	tests := []struct {
		name   string
		states []*domainfeature.CharacterFeatureState
	}{
		{
			name: "fractional level",
			states: []*domainfeature.CharacterFeatureState{{
				CharacterID: 99, FeatureKey: domainfeature.FeatureContractPet,
				State: map[string]interface{}{"hp": 24.5},
			}},
		},
		{
			name: "above catalog maximum",
			states: []*domainfeature.CharacterFeatureState{{
				CharacterID: 99, FeatureKey: domainfeature.FeatureContractPet,
				State: map[string]interface{}{"magic": 51.0},
			}},
		},
		{
			name: "duplicate state",
			states: []*domainfeature.CharacterFeatureState{
				{CharacterID: 99, FeatureKey: domainfeature.FeatureContractPet, State: map[string]interface{}{}},
				{CharacterID: 99, FeatureKey: domainfeature.FeatureContractPet, State: map[string]interface{}{}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewPetContractStateReader(characterFeatureStateRepositoryFixture{states: tt.states})
			if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
				t.Fatal("ReadStateProofs() error = nil, want invalid state error")
			}
		})
	}
}

func TestPetContractLevelRangeMatchesCheckedInCatalog(t *testing.T) {
	path := filepath.Join("..", "..", "..", "docs", "database", "game_data", "TBL_PET_CONTRACT.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var fixture struct {
		Records []struct {
			ID string `json:"id"`
		} `json:"records"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	seen := make(map[int]struct{}, len(fixture.Records))
	for _, record := range fixture.Records {
		id, err := strconv.Atoi(record.ID)
		if err != nil {
			t.Fatalf("parse contract level %q: %v", record.ID, err)
		}
		seen[id] = struct{}{}
	}
	if len(fixture.Records) != maxPetContractLevel {
		t.Fatalf("contract row count = %d, want %d", len(fixture.Records), maxPetContractLevel)
	}
	for level := 1; level <= maxPetContractLevel; level++ {
		if _, ok := seen[level]; !ok {
			t.Fatalf("contract catalog missing level %d", level)
		}
	}
}
