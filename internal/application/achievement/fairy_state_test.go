// Open-sourced by BaoLT

package achievement

import (
	"context"
	"strconv"
	"testing"

	appfairy "mcgame-server/internal/application/fairy"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"
)

type fairyStateCatalogFixture struct {
	entries []*models.FairyTempalteTemplate
}

func (c fairyStateCatalogFixture) GetAllFairyTempaltes() []*models.FairyTempalteTemplate {
	return c.entries
}

func TestFairyStateReaderUsesRosterGrowthSkillsGuardAndSkins(t *testing.T) {
	skills := map[string]interface{}{}
	for index, skillID := range []int{5917, 5920, 5923, 5926, 5929, 5932} {
		skills["s"+strconv.Itoa(index+1)] = skillID
	}
	repo := characterFeatureStateRepositoryFixture{states: []*domainfeature.CharacterFeatureState{
		{
			CharacterID: 99,
			FeatureKey:  domainfeature.FeatureFairy,
			State: map[string]interface{}{"fairy": map[string]interface{}{
				"1": map[string]interface{}{"id": int64(1), "tid": 1, "exp": int64(0), "gexp": int64(0), "state": 1, "skillFlag": map[string]interface{}{}},
				"2": map[string]interface{}{"id": int64(2), "tid": 11, "exp": appfairy.FairyLevelExp[19], "gexp": appfairy.FairyGrowExp[40], "state": 0, "skillFlag": skills},
			}},
		},
		{
			CharacterID: 99,
			FeatureKey:  domainfeature.FeatureFairySkin,
			State: map[string]interface{}{"fairy_skin": map[string]interface{}{
				"actFairy": 1, "actFairyList": "1|2|3|4|5",
			}},
		},
	}}
	proofs, err := NewFairyStateReader(repo, canonicalFairyStateCatalogFixture()).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 17 {
		t.Fatalf("proof count = %d, want 17", len(proofs))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{500, 505, 506, 507, 508, 523, 525, 560} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{501, 509, 561} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

func TestFairyStateReaderRejectsInvalidPersistedState(t *testing.T) {
	tests := []struct {
		name   string
		states []*domainfeature.CharacterFeatureState
	}{
		{
			name: "multiple guards",
			states: []*domainfeature.CharacterFeatureState{{CharacterID: 99, FeatureKey: domainfeature.FeatureFairy, State: map[string]interface{}{"fairy": map[string]interface{}{
				"1": map[string]interface{}{"tid": 1, "exp": 0, "gexp": 0, "state": 1},
				"2": map[string]interface{}{"tid": 2, "exp": 0, "gexp": 0, "state": 1},
			}}}},
		},
		{
			name: "invalid skill",
			states: []*domainfeature.CharacterFeatureState{{CharacterID: 99, FeatureKey: domainfeature.FeatureFairy, State: map[string]interface{}{"fairy": map[string]interface{}{
				"1": map[string]interface{}{"tid": 1, "exp": 0, "gexp": 0, "state": 0, "skillFlag": map[string]interface{}{"s1": 999}},
			}}}},
		},
		{
			name: "unknown skin",
			states: []*domainfeature.CharacterFeatureState{{CharacterID: 99, FeatureKey: domainfeature.FeatureFairySkin, State: map[string]interface{}{
				"fairy_skin": map[string]interface{}{"actFairy": 0, "actFairyList": "99"},
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewFairyStateReader(characterFeatureStateRepositoryFixture{states: tt.states}, canonicalFairyStateCatalogFixture())
			if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
				t.Fatal("ReadStateProofs() error = nil, want invalid state error")
			}
		})
	}
}

func TestFairyStateReaderRejectsCatalogIdentityDrift(t *testing.T) {
	catalog := canonicalFairyStateCatalogFixture()
	catalog.entries[0].Name = "drift"
	reader := NewFairyStateReader(characterFeatureStateRepositoryFixture{}, catalog)
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want catalog drift error")
	}
}

func canonicalFairyStateCatalogFixture() fairyStateCatalogFixture {
	entries := make([]*models.FairyTempalteTemplate, 17)
	for id := 1; id <= 17; id++ {
		entries[id-1] = &models.FairyTempalteTemplate{ID: int64(id), Name: "fairy-" + strconv.Itoa(id)}
	}
	for _, definition := range fairyIdentityRequirements {
		entries[definition.templateID-1].Name = definition.name
	}
	return fairyStateCatalogFixture{entries: entries}
}
