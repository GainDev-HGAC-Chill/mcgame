package achievement

import (
	"context"
	"strconv"
	"testing"

	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"
)

type petHandbookCatalogFixture struct {
	entries []*models.CreatureHandbookTemplate
}

func (c petHandbookCatalogFixture) GetAllCreatureHandbooks() []*models.CreatureHandbookTemplate {
	return c.entries
}

func TestPetHandbookStateReaderCountsActivatedEntriesByClass(t *testing.T) {
	catalog := canonicalPetHandbookCatalogFixture()
	state := map[string]interface{}{}
	for _, entry := range catalog.entries {
		classID := int(entry.ClassID)
		if classID == 1 || (classID == 2 && entry.ID%petHandbookClassSize == 1) {
			state[stringHandbookID(entry.ID)] = map[string]interface{}{"actived": true}
		}
	}
	repo := characterFeatureStateRepositoryFixture{states: []*domainfeature.CharacterFeatureState{{
		CharacterID: 99, FeatureKey: domainfeature.FeatureActivePet, State: state,
	}}}
	proofs, err := NewPetHandbookStateReader(repo, catalog).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 16 {
		t.Fatalf("proof count = %d, want 16", len(proofs))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{539, 547, 548} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{540, 541, 549} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

func TestPetHandbookStateReaderRejectsCatalogDrift(t *testing.T) {
	catalog := canonicalPetHandbookCatalogFixture()
	catalog.entries = catalog.entries[:len(catalog.entries)-1]
	reader := NewPetHandbookStateReader(characterFeatureStateRepositoryFixture{}, catalog)
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want catalog drift error")
	}
}

func canonicalPetHandbookCatalogFixture() petHandbookCatalogFixture {
	entries := make([]*models.CreatureHandbookTemplate, 0, 80)
	for classID := 1; classID <= 8; classID++ {
		for offset := 1; offset <= petHandbookClassSize; offset++ {
			id := int64((classID * petHandbookClassSize) + offset)
			entries = append(entries, &models.CreatureHandbookTemplate{ID: id, ClassID: float64(classID)})
		}
	}
	return petHandbookCatalogFixture{entries: entries}
}

func stringHandbookID(id int64) string {
	return strconv.FormatInt(id, 10)
}
