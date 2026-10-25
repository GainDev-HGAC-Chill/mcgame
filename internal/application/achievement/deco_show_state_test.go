// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"
	"time"

	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"
)

type decoShowCatalogFixture map[int]*models.DecoShowTemplate

func (c decoShowCatalogFixture) GetDecoShow(id int) *models.DecoShowTemplate {
	return c[id]
}

func TestDecoShowStateReaderUsesExactPositionAndExpiry(t *testing.T) {
	now := time.UnixMilli(2_000_000)
	repo := characterFeatureStateRepositoryFixture{states: []*domainfeature.CharacterFeatureState{{
		CharacterID: 99,
		FeatureKey:  domainfeature.FeatureDecoHole,
		State: map[string]interface{}{"positions": map[string]interface{}{
			"1": map[string]interface{}{"activeFlag": `{"r":{},"s":{"1":1,"9":2000001,"13":1999999}}`},
			"2": map[string]interface{}{"activeFlag": `{"r":{},"s":{"18":1}}`},
			"3": map[string]interface{}{"activeFlag": `{r:{},s:{}}`},
			"4": map[string]interface{}{"activeFlag": `{"r":{},"s":{}}`},
		}},
	}}}
	reader := NewDecoShowStateReader(repo, canonicalDecoShowCatalogFixture())
	reader.now = func() time.Time { return now }
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != len(decoShowRequirements) {
		t.Fatalf("proof count = %d, want %d", len(proofs), len(decoShowRequirements))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{687, 691, 755} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{688, 741, 754} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

func TestDecoShowStateReaderRejectsInvalidState(t *testing.T) {
	tests := []struct {
		name      string
		positions map[string]interface{}
	}{
		{name: "wrong position", positions: map[string]interface{}{"2": map[string]interface{}{"activeFlag": `{"s":{"1":1}}`}}},
		{name: "fractional expiry", positions: map[string]interface{}{"1": map[string]interface{}{"activeFlag": `{"s":{"1":2.5}}`}}},
		{name: "unknown show", positions: map[string]interface{}{"1": map[string]interface{}{"activeFlag": `{"s":{"21":1}}`}}},
		{name: "malformed flag", positions: map[string]interface{}{"1": map[string]interface{}{"activeFlag": `{"s":`}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := characterFeatureStateRepositoryFixture{states: []*domainfeature.CharacterFeatureState{{
				CharacterID: 99,
				FeatureKey:  domainfeature.FeatureDecoHole,
				State:       map[string]interface{}{"positions": tt.positions},
			}}}
			if _, err := NewDecoShowStateReader(repo, canonicalDecoShowCatalogFixture()).ReadStateProofs(context.Background(), 99); err == nil {
				t.Fatal("ReadStateProofs() error = nil, want invalid state error")
			}
		})
	}
}

func TestDecoShowStateReaderRejectsCatalogDrift(t *testing.T) {
	catalog := canonicalDecoShowCatalogFixture()
	catalog[9].SuitID = 2
	reader := NewDecoShowStateReader(characterFeatureStateRepositoryFixture{}, catalog)
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want catalog drift error")
	}
}

func canonicalDecoShowCatalogFixture() decoShowCatalogFixture {
	catalog := make(decoShowCatalogFixture, len(decoShowRequirements))
	for _, definition := range decoShowRequirements {
		catalog[definition.showID] = &models.DecoShowTemplate{
			ID:       int64(definition.showID),
			Position: float64(definition.position),
			SuitID:   float64(definition.suitID),
		}
	}
	return catalog
}
