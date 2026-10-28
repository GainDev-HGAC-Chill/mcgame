package guild

import (
	"context"
	"testing"

	domainguild "mcgame-server/internal/domain/guild"
	"mcgame-server/internal/gamedata/models"
)

type buildingRepositoryFixture struct {
	Repository
	states []*domainguild.BuildingState
}

func (r *buildingRepositoryFixture) ListGuildBuildings(context.Context, int64) ([]*domainguild.BuildingState, error) {
	return r.states, nil
}

func (r *buildingRepositoryFixture) StartGuildBuildingAtomic(context.Context, BuildingStartCommand) (*domainguild.BuildingState, error) {
	return nil, nil
}

func (r *buildingRepositoryFixture) FinishGuildBuildingAtomic(context.Context, BuildingFinishCommand) (*domainguild.BuildingState, error) {
	return nil, nil
}

func (r *buildingRepositoryFixture) ResetGuildBuildingAtomic(context.Context, int64, int64, int64) (*domainguild.BuildingState, error) {
	return nil, nil
}

type buildingCatalogFixture struct {
	buildings map[int]*models.BuildingTemplate
	positions []*models.ExtendPositionTemplate
}

func (c buildingCatalogFixture) GetBuilding(id int) *models.BuildingTemplate {
	return c.buildings[id]
}

func (c buildingCatalogFixture) GetAllBuildings() []*models.BuildingTemplate {
	result := make([]*models.BuildingTemplate, 0, len(c.buildings))
	for _, row := range c.buildings {
		result = append(result, row)
	}
	return result
}

func (c buildingCatalogFixture) GetExtendPositionsByMapID(mapID int) []*models.ExtendPositionTemplate {
	if mapID != GuildBuildingMapID {
		return nil
	}
	return c.positions
}

func TestResolveBuildingStartEnforcesExactChainPrerequisitesAndLimits(t *testing.T) {
	catalog := buildingCatalogFixture{
		buildings: map[int]*models.BuildingTemplate{
			2: {ID: 2, CodeName: "HALL", Type: 1, Level: 1, PreBuilding: "1", NumLimit: 1},
			3: {ID: 3, CodeName: "HOUSE", Type: 1, Level: 1, PreBuilding: "2", NumLimit: 2},
			5: {ID: 5, CodeName: "WAREHOUSE", Type: 1, Level: 1, PreBuilding: "2", NumLimit: 1},
			6: {ID: 6, CodeName: "HALL", Type: 1, Level: 2, PreBuilding: "2|3|5", NumLimit: 1, MoneyCost: 10, ExpCost: 20, GenMCost: 3, RareMCost: 4, SPMCost: 5},
			7: {ID: 7, CodeName: "HALL", Type: 1, Level: 3, PreBuilding: "6", NumLimit: 1},
		},
		positions: []*models.ExtendPositionTemplate{{ID: 10, Mid: 49, BuildType: 1, Tid: 1}, {ID: 11, Mid: 49, BuildType: 1, Tid: 1}},
	}
	service := &Service{buildingCatalog: catalog}
	create, err := service.resolveBuildingStart(8, 1, 10, 2, BuildingCreateMode, nil)
	if err != nil || create.ExtendID != 10 || create.TargetTemplateID != 2 {
		t.Fatalf("create = %#v, error = %v", create, err)
	}
	states := []*domainguild.BuildingState{
		{ID: 100, GuildID: 8, ExtendID: 10, TemplateID: 2, Revision: 4},
		{ID: 101, GuildID: 8, ExtendID: 11, TemplateID: 3, Revision: 1},
		{ID: 102, GuildID: 8, ExtendID: 12, TemplateID: 5, Revision: 1},
	}
	upgrade, err := service.resolveBuildingStart(8, 1, 100, 6, BuildingUpgradeMode, states)
	if err != nil || upgrade.ExtendID != 10 || upgrade.CurrentTemplateID != 2 || upgrade.ExpectedRevision != 4 {
		t.Fatalf("upgrade = %#v, error = %v", upgrade, err)
	}
	for _, test := range []struct {
		name   string
		source int64
		target int
		mode   int
		states []*domainguild.BuildingState
	}{
		{"skip level", 100, 7, BuildingUpgradeMode, states},
		{"missing prerequisite", 100, 6, BuildingUpgradeMode, states[:2]},
		{"occupied extension", 10, 2, BuildingCreateMode, states},
		{"wrong extension", 99, 2, BuildingCreateMode, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.resolveBuildingStart(8, 1, test.source, test.target, test.mode, test.states); err == nil {
				t.Fatal("invalid building transition was accepted")
			}
		})
	}
}

func TestBuildingFinishCostsFailClosed(t *testing.T) {
	state := &domainguild.BuildingState{ID: 1, PendingTemplateID: 6, Revision: 2}
	template := &models.BuildingTemplate{ID: 6, CodeName: "HALL", Type: 1, Level: 2, MoneyCost: 10, ExpCost: 20, GenMCost: 3, RareMCost: 4, SPMCost: 5}
	command, err := buildingFinishCommand(8, 1, state, template)
	if err != nil || command.MoneyCost != 10 || command.ExperienceCost != 20 || command.GeneralCost != 3 || command.RareCost != 4 || command.SpecialCost != 5 {
		t.Fatalf("command = %#v, error = %v", command, err)
	}
	template.GoldCost = 1
	if _, err := buildingFinishCommand(8, 1, state, template); err == nil {
		t.Fatal("unresolved gold payer must fail closed")
	}
	template.GoldCost = 0
	template.MoneyCost = 1.5
	if _, err := buildingFinishCommand(8, 1, state, template); err == nil {
		t.Fatal("fractional catalog cost must fail closed")
	}
}

func TestBuildingFinishCommandDerivesAuthoritativeHallLevel(t *testing.T) {
	state := &domainguild.BuildingState{ID: 1, PendingTemplateID: 6, Revision: 2}
	template := &models.BuildingTemplate{ID: 6, CodeName: "BUILD100001", Type: 1, Level: 2, Prop1: 4, Prop1Value: 2}
	command, err := buildingFinishCommand(8, 1, state, template)
	if err != nil || command.HallLevel != 2 {
		t.Fatalf("command = %#v, error = %v", command, err)
	}
	template.Prop1Value = 3
	if _, err := buildingFinishCommand(8, 1, state, template); err == nil {
		t.Fatal("Hall property drift was accepted")
	}
}

func TestBuildingFunctionAvailableRequiresCompletedFunctionalBuilding(t *testing.T) {
	catalog := buildingCatalogFixture{buildings: map[int]*models.BuildingTemplate{
		4: {ID: 4, CodeName: "ACADEMY", Type: 1, Level: 1, FuncScript: `func:"showGuildSkillPanel"`},
		5: {ID: 5, CodeName: "WAREHOUSE", Type: 1, Level: 1, FuncScript: `func:"showGuildWarehouse";func:"showDonatePanel"`},
	}}
	repo := &buildingRepositoryFixture{states: []*domainguild.BuildingState{
		{ID: 10, GuildID: 8, ExtendID: 10, PendingTemplateID: 5, Mode: BuildingCreateMode},
		{ID: 11, GuildID: 8, ExtendID: 11, TemplateID: 4},
	}}
	service := &Service{repo: repo, buildingCatalog: catalog}

	if available, err := service.BuildingFunctionAvailable(context.Background(), 8, "showGuildWarehouse"); err != nil || available {
		t.Fatalf("pending warehouse availability = %v, error = %v", available, err)
	}
	if available, err := service.BuildingFunctionAvailable(context.Background(), 8, "showGuildSkillPanel"); err != nil || !available {
		t.Fatalf("completed academy availability = %v, error = %v", available, err)
	}
	if available, err := service.BuildingFunctionAvailable(context.Background(), 8, "unknownFunction"); err != nil || available {
		t.Fatalf("unknown function availability = %v, error = %v", available, err)
	}

	repo.states[0] = &domainguild.BuildingState{ID: 10, GuildID: 8, ExtendID: 10, TemplateID: 5}
	for _, functionName := range []string{"showGuildWarehouse", "showDonatePanel"} {
		if available, err := service.BuildingFunctionAvailable(context.Background(), 8, functionName); err != nil || !available {
			t.Fatalf("completed warehouse %s availability = %v, error = %v", functionName, available, err)
		}
	}
}
