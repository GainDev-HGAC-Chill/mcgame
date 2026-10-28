// Guild material contribution wire and conversion contracts.
package guild

import (
	"testing"

	"mcgame-server/internal/gamedata/models"
)

func TestGuildMaterialContributionUsesPersistedItemColor(t *testing.T) {
	template := &models.ItemTemplateTemplate{ID: 16, Kind: itemKindMaterial, Color: -1}

	warehouseTemplateID, warehouseCount, contribution, err := guildMaterialContribution(template, 3, 1)
	if err != nil {
		t.Fatalf("guildMaterialContribution() error = %v", err)
	}
	if warehouseTemplateID != guildMaterialNormalTemplate || warehouseCount != 30 || contribution != 1518 {
		t.Fatalf("result = (%d, %d, %d), want (%d, 30, 1518)", warehouseTemplateID, warehouseCount, contribution, guildMaterialNormalTemplate)
	}
}

func TestGuildMaterialContributionRejectsLowPersistedItemColor(t *testing.T) {
	template := &models.ItemTemplateTemplate{ID: 16, Kind: itemKindMaterial, Color: 5}

	if _, _, _, err := guildMaterialContribution(template, 1, 1); err == nil {
		t.Fatal("guildMaterialContribution() error = nil, want low instance color rejection")
	}
}
