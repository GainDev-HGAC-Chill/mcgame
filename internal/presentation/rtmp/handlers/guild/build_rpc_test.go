// Guild building clicks resolve persisted state before selecting Flash callbacks.
package guild

import (
	"errors"
	"testing"

	appguild "mcgame-server/internal/application/guild"
	domainguild "mcgame-server/internal/domain/guild"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

func TestResolveGuildBuildClickPendingUsesProgressCallback(t *testing.T) {
	buildings := []*domainguild.BuildingState{{ID: 9, ExtendID: 10, PendingTemplateID: 2, Mode: 2}}

	callback, templateID, err := resolveGuildBuildClick(9, guildBuildingType, buildings)
	if err != nil {
		t.Fatalf("resolveGuildBuildClick() error = %v", err)
	}
	if callback != "onShowBuildProgress" || templateID != 2 {
		t.Fatalf("result = (%q, %d), want (onShowBuildProgress, 2)", callback, templateID)
	}
}

func TestResolveGuildBuildClickCompletedUsesPersistedTemplate(t *testing.T) {
	buildings := []*domainguild.BuildingState{{ID: 9, ExtendID: 10, TemplateID: 5}}

	callback, templateID, err := resolveGuildBuildClick(9, guildBuildingType, buildings)
	if err != nil {
		t.Fatalf("resolveGuildBuildClick() error = %v", err)
	}
	if callback != "" || templateID != 5 {
		t.Fatalf("result = (%q, %d), want empty callback and template 5", callback, templateID)
	}
}

func TestResolveGuildBuildClickEmptyExtensionUsesExtensionTemplate(t *testing.T) {
	callback, templateID, err := resolveGuildBuildClick(10, guildExtendBuildType, nil)
	if err != nil {
		t.Fatalf("resolveGuildBuildClick() error = %v", err)
	}
	if callback != "" || templateID != guildExtendTemplateID {
		t.Fatalf("result = (%q, %d), want empty callback and template %d", callback, templateID, guildExtendTemplateID)
	}
}

func TestResolveGuildBuildClickRejectsUnownedBuilding(t *testing.T) {
	_, _, err := resolveGuildBuildClick(9, guildBuildingType, nil)
	if !errors.Is(err, pkgerrors.ErrNotFound) {
		t.Fatalf("resolveGuildBuildClick() error = %v, want not found", err)
	}
}

func TestResetGuildBuildingCallbackCreateRestoresEmptyExtension(t *testing.T) {
	previous := &domainguild.BuildingState{ID: 9, ExtendID: 10, PendingTemplateID: 2, Mode: appguild.BuildingCreateMode}
	committed := &domainguild.BuildingState{ID: 9, ExtendID: 10}
	position := &models.ExtendPositionTemplate{ID: 10, Mid: 49, Tid: 1, BuildType: 1, PosX: 245, PosY: 407}

	callback, payload, err := resetGuildBuildingCallback(previous, committed, position)
	if err != nil {
		t.Fatalf("resetGuildBuildingCallback() error = %v", err)
	}
	if callback != "onCancelCreate" || payload["bid"] != int64(9) {
		t.Fatalf("callback = %q, payload = %#v", callback, payload)
	}
	extension, ok := payload["ext"].(map[string]interface{})
	if !ok || extension["id"] != "10" || extension["tid"] != "1" || extension["mid"] != "49" {
		t.Fatalf("extension payload = %#v", payload["ext"])
	}
}

func TestResetGuildBuildingCallbackUpgradeKeepsCompletedBuilding(t *testing.T) {
	previous := &domainguild.BuildingState{ID: 9, ExtendID: 10, TemplateID: 2, PendingTemplateID: 6, Mode: appguild.BuildingUpgradeMode}
	committed := &domainguild.BuildingState{ID: 9, ExtendID: 10, TemplateID: 2}

	callback, payload, err := resetGuildBuildingCallback(previous, committed, nil)
	if err != nil {
		t.Fatalf("resetGuildBuildingCallback() error = %v", err)
	}
	if callback != "onCancelUpgrade" || payload["bid"] != int64(9) {
		t.Fatalf("callback = %q, payload = %#v", callback, payload)
	}
}
