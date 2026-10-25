// Open-sourced by BaoLT

package achievement

import (
	"context"
	"strconv"
	"testing"

	appmystre "mcgame-server/internal/application/mysterytreasure"
	"mcgame-server/internal/gamedata/models"
)

type mysteryTreasureStateProviderFixture struct {
	state *appmystre.MysteryTreasureState
}

func (p mysteryTreasureStateProviderFixture) Load(ctx context.Context, characterID int64) (*appmystre.MysteryTreasureState, error) {
	return p.state, nil
}

type mysteryTreasureCatalogFixture struct {
	templates []*models.MystreTemplate
}

func (c mysteryTreasureCatalogFixture) MysGetAllMystres() []*models.MystreTemplate {
	return c.templates
}

func TestMysteryTreasureStateReaderMapsCatalogKindsAndTotal(t *testing.T) {
	state := appmystre.DefaultState()
	templates := make([]*models.MystreTemplate, 0, 12)
	for kind := 1; kind <= 11; kind++ {
		id := kind * 10
		templates = append(templates, &models.MystreTemplate{ID: int64(id), Kind: float64(kind)})
		state.MysBook[strconv.Itoa(kind)] = map[string]int{strconv.Itoa(id): id}
	}
	templates = append(templates, &models.MystreTemplate{ID: 11, Kind: 1})
	state.MysBook["1"]["11"] = 999
	reader := NewMysteryTreasureStateReader(
		mysteryTreasureStateProviderFixture{state: state},
		mysteryTreasureCatalogFixture{templates: templates},
	)
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 12 {
		t.Fatalf("proof count = %d, want 12", len(proofs))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	if byID[675].Progress != 11 || byID[675].Target != 12 || byID[675].Done {
		t.Fatalf("total proof = %#v", byID[675])
	}
	if byID[676].Progress != 1 || byID[676].Target != 2 || byID[676].Done {
		t.Fatalf("kind-one proof = %#v", byID[676])
	}
	for requirementID := int64(677); requirementID <= 686; requirementID++ {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
}

func TestMysteryTreasureStateReaderRejectsIncompleteCatalogTopology(t *testing.T) {
	reader := NewMysteryTreasureStateReader(
		mysteryTreasureStateProviderFixture{state: appmystre.DefaultState()},
		mysteryTreasureCatalogFixture{templates: []*models.MystreTemplate{{ID: 1, Kind: 1}}},
	)
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatalf("ReadStateProofs() error = nil, want incomplete catalog error")
	}
}

var _ MysteryTreasureStateProvider = mysteryTreasureStateProviderFixture{}
var _ MysteryTreasureCatalog = mysteryTreasureCatalogFixture{}
