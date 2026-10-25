package achievement

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata/models"
)

type petRosterRepositoryFixture struct {
	pets []*domainpet.Pet
}

func (r petRosterRepositoryFixture) FindByCharacterID(context.Context, int64) ([]*domainpet.Pet, error) {
	return r.pets, nil
}

type petRosterCatalogFixture map[int]*models.CreatureTemplate

func (c petRosterCatalogFixture) GetCreature(id int) *models.CreatureTemplate {
	return c[id]
}

func TestPetRosterStateReaderUsesDistinctTypesQualityAndPersistedSkills(t *testing.T) {
	catalog := requiredPetCatalogFixture()
	pet := &domainpet.Pet{
		ID: 1, CharacterID: 99, TemplateID: 71, Level: 80, GrowRate: 2.1,
		Property: map[string]interface{}{},
	}
	for slot := 1; slot <= 10; slot++ {
		pet.Property[fmt.Sprintf("skill%d", slot)] = 1000 + slot
	}
	reader := NewPetRosterStateReader(petRosterRepositoryFixture{pets: []*domainpet.Pet{pet}}, catalog)
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
	for _, requirementID := range []int64{72, 73, 76, 89, 90, 97, 98, 307} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{59, 66, 71, 74, 81, 82, 91, 99, 308, 343, 765, 766, 767, 768} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

func TestPetRosterStateReaderRejectsCollectionCatalogDrift(t *testing.T) {
	catalog := requiredPetCatalogFixture()
	catalog[71].Name = "drift"
	reader := NewPetRosterStateReader(petRosterRepositoryFixture{}, catalog)
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want catalog drift error")
	}
}

func TestPetRosterStateReaderRequiresOrangeGuardianQuality(t *testing.T) {
	catalog := requiredPetCatalogFixture()
	reader := NewPetRosterStateReader(petRosterRepositoryFixture{pets: []*domainpet.Pet{
		{ID: 1, CharacterID: 99, TemplateID: 2290, GrowRate: 2.1, Property: map[string]interface{}{}},
		{ID: 2, CharacterID: 99, TemplateID: 2281, GrowRate: 1.9, Property: map[string]interface{}{}},
	}}, catalog)
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	if !byID[765].Done {
		t.Fatalf("requirement 765 proof = %#v, want done", byID[765])
	}
	for _, requirementID := range []int64{766, 767, 768} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

func TestPetCollectionRequirementsMatchCheckedInCatalog(t *testing.T) {
	path := filepath.Join("..", "..", "..", "docs", "database", "game_data", "TBL_CREATURE.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	fixture := achievementCatalogFixture{}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	catalog := petRosterCatalogFixture{}
	for _, record := range fixture.Records {
		input := struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{}
		if err := json.Unmarshal(record, &input); err != nil {
			t.Fatalf("decode creature record: %v", err)
		}
		id, err := strconv.Atoi(input.ID)
		if err != nil {
			t.Fatalf("parse creature id %q: %v", input.ID, err)
		}
		if id == 2 {
			catalog[id] = &models.CreatureTemplate{ID: int64(id), Name: input.Name}
		}
		for _, definition := range petCollectionRequirements {
			if id == definition.templateID {
				catalog[id] = &models.CreatureTemplate{ID: int64(id), Name: input.Name}
			}
		}
		for _, definition := range petOrangeGuardianRequirements {
			if id == definition.templateID {
				catalog[id] = &models.CreatureTemplate{ID: int64(id), Name: input.Name}
			}
		}
	}
	if err := validatePetCollectionCatalog(catalog); err != nil {
		t.Fatalf("checked-in pet collection catalog: %v", err)
	}
}

func requiredPetCatalogFixture() petRosterCatalogFixture {
	catalog := petRosterCatalogFixture{
		2: {ID: 2, Name: "Thất Vị Kê", ClassID: 2, ClassIds: 2},
	}
	for _, definition := range petCollectionRequirements {
		catalog[definition.templateID] = &models.CreatureTemplate{
			ID: int64(definition.templateID), Name: definition.name, ClassID: 1, ClassIds: 10,
		}
	}
	for _, definition := range petOrangeGuardianRequirements {
		catalog[definition.templateID] = &models.CreatureTemplate{
			ID: int64(definition.templateID), Name: definition.name, ClassID: 1, ClassIds: 10,
		}
	}
	return catalog
}
