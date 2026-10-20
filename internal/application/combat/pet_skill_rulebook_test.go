package combat

import (
	domaincombat "mcgame-server/internal/domain/combat"
	"mcgame-server/internal/gamedata/models"
	"strconv"
	"strings"
	"testing"
)

func petTestSkill(id int64, desc string) *models.SkillTemplate {
	return &models.SkillTemplate{ID: id, Kind: 1, Level: 1, Name: "Kiếm Ảnh", ReqClass: "|100|", TargetNum: 1, TargetType: 4, Type: 1, BuffID: -1, BuffRate: -1, AreaAttack: -1, Description: desc}
}

func TestPetGenericPhysicalDamage(t *testing.T) {
	plan := petSkillDataPlan(petTestSkill(6195, "Gây 240% tấn công vật lý lên mục tiêu"))
	if plan == nil || plan.Effects[0].Kind != domaincombat.EffectPhysicalDamage || plan.Effects[0].Coefficient != 2.4 {
		t.Fatalf("physical plan = %#v, want x2.4", plan)
	}
}

func TestPetGenericMagicDamage(t *testing.T) {
	tpl := petTestSkill(6199, "Gây 240% tấn công ma pháp lên mục tiêu")
	tpl.Type = 3
	plan := petSkillDataPlan(tpl)
	if plan == nil || plan.Effects[0].Kind != domaincombat.EffectMagicDamage || plan.Effects[0].Coefficient != 2.4 {
		t.Fatalf("magic plan = %#v, want x2.4", plan)
	}
}

func TestPetGenericRejectsIncompleteSecondaryEffects(t *testing.T) {
	for _, desc := range []string{
		"Gây 240% sát thương vật lý, có 30% tỷ lệ choáng",
		"Gây 240% sát thương vật lý và 15% giới hạn HP",
		"Gây 240% sát thương vật lý lên 3 mục tiêu",
		"Gây 200% sát thương và tăng công",
	} {
		if plan := petSkillDataPlan(petTestSkill(6195, desc)); plan != nil {
			t.Fatalf("complex skill %q must require explicit rule: %#v", desc, plan)
		}
	}
}

func TestPetGenericRejectsNonPetClassAndNamedBuffs(t *testing.T) {
	tpl := petTestSkill(6195, "Gây 240% sát thương vật lý lên mục tiêu")
	tpl.ReqClass = "|6|"
	if petSkillDataPlan(tpl) != nil {
		t.Fatal("modified a player class skill")
	}
	tpl.ReqClass = "|100|"
	tpl.BuffID = 14
	if petSkillDataPlan(tpl) != nil {
		t.Fatal("silent removal of original buff")
	}
}

func TestPetRulebookStrictSchemaAndCompositeEffects(t *testing.T) {
	data := []byte(`{"version":1,"plans":{"90001":{"Family":"Ví dụ đa hiệu ứng","Level":1,"Target":1,"TargetCount":1,"Effects":[{"Kind":1,"Coefficient":1.25},{"Kind":8,"BuffID":14,"Duration":2,"Chance":0.5}]}}}`)
	rules, err := readPetSkillRules(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := rules[90001]; len(got.Effects) != 2 || got.Effects[1].BuffID != 14 {
		t.Fatalf("bad rulebook plan %#v", got)
	}
	if _, err := readPetSkillRules([]byte(`{"version":1,"plans":{"90001":{"Family":"a","Level":1,"Target":1,"TargetCount":1,"Effects":[{"Kind":1,"Coefficient":0}]}}}`)); err == nil {
		t.Fatal("zero damage rule not rejected")
	}
	if _, err := readPetSkillRules([]byte(`{"version":1,"plans":{},"unknown":true}`)); err == nil {
		t.Fatal("unknown top-level field not rejected")
	}
}

func TestAuditedHuyenMinhPlanPreserved(t *testing.T) {
	for _, id := range []int{7002, 7003, 7004, 7005, 7006, 7007, 7008, 7009, 7010} {
		if petActiveSkillPlanByTemplateID(id) == nil {
			t.Fatalf("existing Huyền Minh Linh Vũ plan %d missing", id)
		}
	}
}

// This is a report, not a hard-coded assertion on the entire evolving seed.
// Run: go test ./internal/application/combat -run TestPetSkillEngineCoverageReport -v -count=1
func TestPetSkillEngineCoverageReport(t *testing.T) {
	rows := readCatalogFixture(t, "TBL_SKILL.json")
	active := 0
	exact := 0
	family := 0
	automatic := 0
	unknown := 0
	examples := make([]string, 0, 12)
	number := func(row map[string]string, key string) float64 {
		v, _ := strconv.ParseFloat(row[key], 64)
		return v
	}
	for _, row := range rows.Records {
		if row["kind"] != "1" || !strings.Contains(row["reqClass"], "|100|") {
			continue
		}
		active++
		id := int(number(row, "id"))
		level := int(number(row, "level"))
		name := strings.TrimSpace(row["name"])
		if petActiveSkillPlanByTemplateID(id) != nil {
			exact++
			continue
		}
		if petActiveSkillPlan(name, level) != nil {
			family++
			continue
		}
		tpl := &models.SkillTemplate{
			ID: int64(id), Name: name, Kind: 1, Level: float64(level),
			ReqClass: row["reqClass"], Description: row["description"],
			TargetType: number(row, "targetType"), TargetNum: number(row, "targetNum"),
			AreaAttack: number(row, "areaAttack"), Type: number(row, "type"),
			BuffID: number(row, "buffId"), BuffRate: number(row, "buffRate"),
		}
		if petSkillDataPlan(tpl) != nil {
			automatic++
			continue
		}
		unknown++
		if len(examples) < 12 {
			examples = append(examples, strconv.Itoa(id)+" "+name)
		}
	}
	t.Logf("Pet active skills total=%d exact=%d family=%d automatic=%d pending_review=%d", active, exact, family, automatic, unknown)
	for _, entry := range examples {
		t.Logf("PENDING_RULE %s", entry)
	}
	if active == 0 {
		t.Log("No pet ReqClass markers in this version of fixture; coverage report skipped")
	}
}
