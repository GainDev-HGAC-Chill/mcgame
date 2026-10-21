package postgres

import (
	"fmt"
	"testing"

	domainpet "mcgame-server/internal/domain/pet"
)

type petAttributePointRow struct{}

func (petAttributePointRow) Scan(dest ...any) error {
	if len(dest) != 34 {
		return fmt.Errorf("destination count = %d", len(dest))
	}
	*(dest[30].(*int)) = 0
	*(dest[31].(*int)) = 440
	*(dest[32].(*int)) = 430
	*(dest[33].(*int)) = 9800
	return nil
}

func TestScanPetRowMapsPhysicalAttributePointOrder(t *testing.T) {
	p := &domainpet.Pet{}
	if err := scanPetRow(petAttributePointRow{}, p); err != nil {
		t.Fatal(err)
	}
	if p.AttrPoints != 0 || p.DistributedAttrPoints != 440 || p.MaxAttrPoints != 430 || p.Life != 9800 {
		t.Fatalf("attribute point mapping is wrong: attr=%d distributed=%d max=%d life=%d", p.AttrPoints, p.DistributedAttrPoints, p.MaxAttrPoints, p.Life)
	}
}
