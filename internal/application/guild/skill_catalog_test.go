package guild

import (
	"testing"

	"mcgame-server/internal/gamedata/models"
)

type skillCatalogFixture map[int]*models.SkillTemplate

func (c skillCatalogFixture) GetSkill(id int) *models.SkillTemplate { return c[id] }

func (c skillCatalogFixture) GetSkillIDsByUseEnv(env int) []int {
	ids := []int{}
	for id, skill := range c {
		if skill.UseEnv == float64(env) {
			ids = append(ids, id)
		}
	}
	return ids
}

func TestGuildSkillDevelopmentExactChainAndCosts(t *testing.T) {
	catalog := skillCatalogFixture{
		1: {ID: 1, CodeName: "A", UseEnv: 4, Level: 1, GuildDevExp: 10, GuildDevMoney: 20},
		2: {ID: 2, CodeName: "A", UseEnv: 4, Level: 2, GuildDevExp: 30, GuildDevMoney: 40},
		3: {ID: 3, CodeName: "A", UseEnv: 4, Level: 3},
		4: {ID: 4, CodeName: "A", UseEnv: 4, Level: 4},
		5: {ID: 5, CodeName: "A", UseEnv: 4, Level: 5},
		6: {ID: 6, CodeName: "A", UseEnv: 4, Level: 6},
		7: {ID: 7, CodeName: "ANO", UseEnv: 4, Level: 7},
		8: {ID: 8, CodeName: "B", UseEnv: 4, Level: 1, CostGuildContrib: 1},
		9: {ID: 9, CodeName: "A", UseEnv: 1, Level: 1},
	}
	result, err := ResolveGuildSkillDevelopment(catalog, 2, []int{1})
	if err != nil || result.Previous != 1 || result.ExpCost != 30 || result.MoneyCost != 40 {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
	for _, test := range []struct {
		name    string
		id      int
		learned []int
	}{
		{"skip", 3, []int{1}}, {"relearn", 1, []int{1}}, {"level7", 7, nil},
		{"payer unknown", 8, nil}, {"wrong environment", 9, nil}, {"duplicate family", 3, []int{1, 2}},
		{"missing persisted skill", 1, []int{99}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ResolveGuildSkillDevelopment(catalog, test.id, test.learned); err == nil {
				t.Fatal("invalid development was accepted")
			}
		})
	}
}
