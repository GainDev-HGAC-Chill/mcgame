package achievement

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	domaincharacter "mcgame-server/internal/domain/character"
	"mcgame-server/internal/gamedata/models"
)

type dressBookCatalogFixture struct {
	entries []*models.DressTemplate
}

func (c dressBookCatalogFixture) GetAllDresses() []*models.DressTemplate {
	return c.entries
}

func TestDressBookStateReaderUsesActivatedBookAndCatalogType(t *testing.T) {
	catalog := canonicalDressBookCatalogFixture()
	book := map[string]int{"1002": 1, "1018": 2, "2011": 1}
	for id := int64(3000); id < 3018; id++ {
		book[fmt.Sprint(id)] = 1
	}
	character := &domaincharacter.Character{ID: 99, DressInfo: dressBookJSON(t, book)}
	proofs, err := NewDressBookStateReader(characterStateRepositoryFixture{character: character}, catalog).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if len(proofs) != 32 {
		t.Fatalf("proof count = %d, want 32", len(proofs))
	}
	byID := make(map[int64]StateProof, len(proofs))
	for _, proof := range proofs {
		byID[proof.RequirementID] = proof
	}
	for _, requirementID := range []int64{611, 612, 613, 614, 617, 618, 630} {
		if !byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want done", requirementID, byID[requirementID])
		}
	}
	for _, requirementID := range []int64{615, 619, 631, 712, 714} {
		if byID[requirementID].Done {
			t.Fatalf("requirement %d proof = %#v, want incomplete", requirementID, byID[requirementID])
		}
	}
}

func TestDressBookStateReaderRejectsUnknownActivatedDress(t *testing.T) {
	character := &domaincharacter.Character{ID: 99, DressInfo: dressBookJSON(t, map[string]int{"999999": 1})}
	reader := NewDressBookStateReader(characterStateRepositoryFixture{character: character}, canonicalDressBookCatalogFixture())
	if _, err := reader.ReadStateProofs(context.Background(), 99); err == nil {
		t.Fatal("ReadStateProofs() error = nil, want unknown dress error")
	}
}

func TestDressBookRequirementsMatchCheckedInCatalog(t *testing.T) {
	path := filepath.Join("..", "..", "..", "docs", "database", "game_data", "TBL_DRESS.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var fixture struct {
		Records []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			EquiptID string `json:"equiptId"`
		} `json:"records"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	entries := make([]*models.DressTemplate, 0, len(fixture.Records))
	for _, record := range fixture.Records {
		id, idErr := strconv.ParseInt(record.ID, 10, 64)
		dressType, typeErr := strconv.Atoi(record.Type)
		equipmentID, equipmentErr := strconv.Atoi(record.EquiptID)
		if idErr != nil || typeErr != nil || equipmentErr != nil {
			t.Fatalf("invalid checked-in dress record %#v", record)
		}
		entries = append(entries, &models.DressTemplate{ID: id, Type: float64(dressType), EquiptID: float64(equipmentID)})
	}
	if _, err := validatedDressBookCatalog(entries); err != nil {
		t.Fatalf("checked-in dress catalog: %v", err)
	}
}

func canonicalDressBookCatalogFixture() dressBookCatalogFixture {
	byID := make(map[int64]*models.DressTemplate)
	for _, identity := range dressBookIdentityRequirements {
		byID[identity.dressID] = &models.DressTemplate{
			ID: identity.dressID, Type: float64(identity.dressType), EquiptID: float64(identity.equipmentID),
		}
	}
	fashionCount := 0
	flyerCount := 0
	for _, entry := range byID {
		if entry.Type == 1 {
			fashionCount++
		} else {
			flyerCount++
		}
	}
	for id := int64(3000); fashionCount < 55; id++ {
		if byID[id] == nil {
			byID[id] = &models.DressTemplate{ID: id, Type: 1}
			fashionCount++
		}
	}
	for id := int64(4000); flyerCount < 20; id++ {
		if byID[id] == nil {
			byID[id] = &models.DressTemplate{ID: id, Type: 2}
			flyerCount++
		}
	}
	entries := make([]*models.DressTemplate, 0, len(byID))
	for _, entry := range byID {
		entries = append(entries, entry)
	}
	return dressBookCatalogFixture{entries: entries}
}

func dressBookJSON(t *testing.T, book map[string]int) string {
	t.Helper()
	parts := ""
	for id, state := range book {
		if parts != "" {
			parts += ","
		}
		parts += fmt.Sprintf("%q:%d", id, state)
	}
	return `{"book":{` + parts + `},"recipe":{},"bag":{"crystal":0,"jewel":0}}`
}
