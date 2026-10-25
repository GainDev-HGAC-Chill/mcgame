// Open-sourced by BaoLT

package awaken

import (
	"context"
	"encoding/json"
	"testing"

	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"
)

func TestIsAuthorizedSkillVariantRequiresOwnedBaseAndAllocatedRank(t *testing.T) {
	service := newTestService(t, nil)
	rows := []json.RawMessage{
		json.RawMessage(`{"id":101,"code_name":"codeA","ex_sid1":201,"ex_sid2":202,"ex_sid3":203}`),
		json.RawMessage(`{"id":102,"code_name":"codeB","ex_sid1":301}`),
		json.RawMessage(`{"id":201,"code_name":"codeAEx1"}`),
		json.RawMessage(`{"id":202,"code_name":"codeAEx2"}`),
		json.RawMessage(`{"id":301,"code_name":"codeBEx1"}`),
	}
	if err := service.gameData.GetCache().LoadTable(models.TableSkill, rows); err != nil {
		t.Fatalf("LoadTable(TBL_SKILL): %v", err)
	}
	repo := service.featureRepo.(*fakeFeatureRepo)
	repo.charStates[1] = []*domainfeature.CharacterFeatureState{{
		CharacterID: 1,
		FeatureKey:  featureKey,
		State: map[string]interface{}{
			"pointDict": map[string]interface{}{"codeA": 1, "codeB": 1},
		},
	}}

	tests := []struct {
		name      string
		requested int
		learned   []int
		want      bool
	}{
		{name: "matching variant", requested: 201, learned: []int{101}, want: true},
		{name: "wrong rank", requested: 202, learned: []int{101}},
		{name: "foreign base", requested: 301, learned: []int{101}},
		{name: "matching second base", requested: 301, learned: []int{102}, want: true},
		{name: "missing template", requested: 999, learned: []int{101}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := service.IsAuthorizedSkillVariant(context.Background(), 1, test.requested, test.learned)
			if err != nil {
				t.Fatalf("IsAuthorizedSkillVariant() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("IsAuthorizedSkillVariant() = %v, want %v", got, test.want)
			}
		})
	}
}
