// Open-sourced by BaoLT

package achievement

import (
	"context"
	"testing"

	domainitem "mcgame-server/internal/domain/item"
	domainmarriage "mcgame-server/internal/domain/marriage"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

type coupleWingMarriageRepositoryFixture struct {
	record *domainmarriage.MarriageRecord
	err    error
}

func (r coupleWingMarriageRepositoryFixture) FindByPartner(ctx context.Context, characterID int64) (*domainmarriage.MarriageRecord, error) {
	return r.record, r.err
}

type coupleWingItemRepositoryFixture map[int64][]*domainitem.Item

func (r coupleWingItemRepositoryFixture) FindEquipped(ctx context.Context, characterID int64) ([]*domainitem.Item, error) {
	return r[characterID], nil
}

func TestCoupleWingStateReaderRequiresMatchingTemplateAndColor(t *testing.T) {
	marriage := &domainmarriage.MarriageRecord{ID: 1, Partner1ID: 99, Partner2ID: 100}
	catalog := characterLoadoutCatalogFixture{1676: {ID: 1676, Position: 22, Kind: 13}}
	items := coupleWingItemRepositoryFixture{
		99:  {equippedWingFixture(1, 99, 1676, 4)},
		100: {equippedWingFixture(2, 100, 1676, 4)},
	}
	proofs, err := NewCoupleWingStateReader(coupleWingMarriageRepositoryFixture{record: marriage}, items, catalog).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	if proofs[0].Done || !proofs[1].Done || proofs[2].Done {
		t.Fatalf("proofs = %#v, want only purple matching wing complete", proofs)
	}
	items[100][0].TemplateID = 1677
	catalog[1677] = &models.EquiptTemplateTemplate{ID: 1677, Position: 22, Kind: 13}
	proofs, err = NewCoupleWingStateReader(coupleWingMarriageRepositoryFixture{record: marriage}, items, catalog).ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() mismatch error = %v", err)
	}
	for _, proof := range proofs {
		if proof.Done {
			t.Fatalf("mismatched wing proof = %#v, want incomplete", proof)
		}
	}
}

func TestCoupleWingStateReaderTreatsNoMarriageAsIncomplete(t *testing.T) {
	reader := NewCoupleWingStateReader(
		coupleWingMarriageRepositoryFixture{err: pkgerrors.ErrNotFound},
		coupleWingItemRepositoryFixture{},
		characterLoadoutCatalogFixture{},
	)
	proofs, err := reader.ReadStateProofs(context.Background(), 99)
	if err != nil {
		t.Fatalf("ReadStateProofs() error = %v", err)
	}
	for _, proof := range proofs {
		if proof.Done {
			t.Fatalf("proof = %#v, want incomplete", proof)
		}
	}
}

func equippedWingFixture(id, characterID int64, templateID, color int) *domainitem.Item {
	return &domainitem.Item{
		ID: id, CharacterID: characterID, TemplateID: templateID,
		ItemType: domainitem.ItemTypeEquipment, SlotType: domainitem.SlotTypeEquipped,
		SlotIndex: 21, ColorCode: color,
	}
}
