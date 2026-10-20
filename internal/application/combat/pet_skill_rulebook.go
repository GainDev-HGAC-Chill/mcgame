// Shared data-backed pet skill fallback. Exact audited plans always take precedence.
package combat

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"

	domaincombat "mcgame-server/internal/domain/combat"
	"mcgame-server/internal/gamedata/models"
)

//go:embed pet_skill_rules.json
var petSkillRulesJSON []byte

type petSkillRulebook struct {
	Version int                               `json:"version"`
	Plans   map[string]domaincombat.SkillPlan `json:"plans"`
}

var petSkillRulesOnce sync.Once
var petSkillRules map[int]domaincombat.SkillPlan
var petSkillRulesError error

// readPetSkillRules is pure so bad or ambiguous data is rejected in tests.
func readPetSkillRules(data []byte) (map[int]domaincombat.SkillPlan, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var book petSkillRulebook
	if err := decoder.Decode(&book); err != nil {
		return nil, fmt.Errorf("invalid pet skill rulebook: %w", err)
	}
	if book.Version != 1 {
		return nil, fmt.Errorf("unsupported pet skill rulebook version %d", book.Version)
	}
	plans := make(map[int]domaincombat.SkillPlan, len(book.Plans))
	for key, plan := range book.Plans {
		id, err := strconv.Atoi(key)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid pet skill ID %q", key)
		}
		if err := validatePetSkillPlan(id, plan); err != nil {
			return nil, err
		}
		plans[id] = plan
	}
	return plans, nil
}

func validatePetSkillPlan(id int, plan domaincombat.SkillPlan) error {
	if strings.TrimSpace(plan.Family) == "" || plan.Level <= 0 || plan.Level > 3 {
		return fmt.Errorf("pet skill %d: Family/Level invalid", id)
	}
	if plan.Target < domaincombat.TargetSelf || plan.Target > domaincombat.TargetSelectedDeadTeam {
		return fmt.Errorf("pet skill %d: Target invalid", id)
	}
	if plan.TargetCount < 1 || plan.TargetCount > 20 || len(plan.Effects) == 0 {
		return fmt.Errorf("pet skill %d: TargetCount/Effects invalid", id)
	}
	if plan.RepeatChance < 0 || plan.RepeatChance > 1 {
		return fmt.Errorf("pet skill %d: RepeatChance invalid", id)
	}
	for index, effect := range plan.Effects {
		if effect.Kind < domaincombat.EffectPhysicalDamage || effect.Kind > domaincombat.EffectTransfer {
			return fmt.Errorf("pet skill %d effect %d: unsupported Kind", id, index)
		}
		if effect.Kind == domaincombat.EffectPhysicalDamage || effect.Kind == domaincombat.EffectMagicDamage {
			if effect.Coefficient <= 0 && effect.CasterMaxHPRatio <= 0 && effect.Fixed <= 0 {
				return fmt.Errorf("pet skill %d effect %d: missing damage coefficient", id, index)
			}
		}
		if effect.BuffID < 0 || effect.Duration < 0 {
			return fmt.Errorf("pet skill %d effect %d: invalid buff ID/duration", id, index)
		}
	}
	return nil
}

func configuredPetSkillPlan(id int) *domaincombat.SkillPlan {
	petSkillRulesOnce.Do(func() {
		petSkillRules, petSkillRulesError = readPetSkillRules(petSkillRulesJSON)
		if petSkillRulesError != nil {
			log.Printf("PET_SKILL_RULEBOOK_INVALID: %v; all configured pet skill rules are disabled", petSkillRulesError)
		}
	})
	if petSkillRulesError != nil {
		return nil
	}
	plan, ok := petSkillRules[id]
	if !ok {
		return nil
	}
	cloned := plan
	cloned.Effects = append([]domaincombat.SkillEffect(nil), plan.Effects...)
	cloned.Costs = append([]domaincombat.SkillCost(nil), plan.Costs...)
	return &cloned
}

var simplePetDamage = regexp.MustCompile(`(?i)gây(?:\s+ra)?\s+(\d+(?:[.,]\d+)?)\s*%\s*(?:sát\s+thương|(?:lực\s+)?tấn\s+công)(?:\s+(vật\s+l[ýí]|ma\s+pháp|phép\s+thuật))?`)
var petMultipleTargets = regexp.MustCompile(`(?i)\b[2-9]\d*\s+mục\s+tiêu`)

// These words indicate unmodelled secondary effects. Do not silently drop them.
var complexPetDescriptionWords = []string{
	"tỷ lệ", "tỉ lệ", "xác suất", "đồng thời", "kết liễu", "mỗi lượt",
	"sát thương thật", "bỏ qua", "xuyên", "buff", "debuff", "choáng",
	"trúng độc", "khống chế", "giải trừ", "hồi phục", "hút máu",
	"giới hạn hp", "máu tối đa", "sinh mệnh", "phản đòn", "phản xạ",
	"giảm phòng", "giảm thủ", "tăng công", "đánh thêm", "tái thi triển",
	"ngẫu nhiên", "tất cả", "hàng ngang", "hàng dọc", "hàng dọc",
}

func isPetActiveSkill(tpl *models.SkillTemplate) bool {
	return tpl != nil && int(tpl.Kind) == 1 && strings.Contains(tpl.ReqClass, "|100|")
}

// petSkillDataPlan covers only one-target direct damage backed by an explicit
// percentage. All complex abilities must use the central JSON rulebook.
// Unhandled scripted pet skills need a reviewed rule instead of made-up damage.
// Log each ID only once when the skill is actually requested in combat.
var missingPetSkillPlans sync.Map

func warnMissingPetSkillPlan(tpl *models.SkillTemplate) {
	if !isPetActiveSkill(tpl) {
		return
	}
	if _, loaded := missingPetSkillPlans.LoadOrStore(tpl.ID, struct{}{}); loaded {
		return
	}
	log.Printf("PET_SKILL_PLAN_MISSING id=%d name=%q code=%q: add an audited SkillPlan to pet_skill_rules.json", tpl.ID, strings.TrimSpace(tpl.Name), tpl.CodeName)
}

func petSkillDataPlan(tpl *models.SkillTemplate) *domaincombat.SkillPlan {
	if !isPetActiveSkill(tpl) {
		return nil
	}
	if plan := configuredPetSkillPlan(int(tpl.ID)); plan != nil {
		return plan
	}
	if tpl.TargetNum != 1 || tpl.BuffID > 0 || tpl.BuffRate > 0 || tpl.AreaAttack > 0 {
		return nil
	}
	if int(tpl.TargetType) != 4 {
		return nil
	} // canonical single enemy
	switch int(tpl.Type) {
	case 1, 2, 3, 4:
	default:
		return nil
	}
	desc := strings.TrimSpace(tpl.Description)
	lowered := strings.ToLower(desc)
	if petMultipleTargets.MatchString(lowered) {
		return nil
	}
	for _, word := range complexPetDescriptionWords {
		if strings.Contains(lowered, word) {
			return nil
		}
	}
	matches := simplePetDamage.FindAllStringSubmatch(desc, -1)
	if len(matches) != 1 || strings.Count(desc, "%") != 1 {
		return nil
	}
	coefficientText := strings.ReplaceAll(matches[0][1], ",", ".")
	value, err := strconv.ParseFloat(coefficientText, 64)
	if err != nil || value <= 0 || value > 1000 {
		return nil
	}
	physical := int(tpl.Type) == 1 || int(tpl.Type) == 2
	damageDescriptor := strings.ToLower(matches[0][2])
	if strings.Contains(damageDescriptor, "ma pháp") || strings.Contains(damageDescriptor, "phép") {
		physical = false
	} else if strings.Contains(damageDescriptor, "vật l") {
		physical = true
	}
	kind := domaincombat.EffectPhysicalDamage
	if !physical {
		kind = domaincombat.EffectMagicDamage
	}
	return &domaincombat.SkillPlan{
		Family:      strings.TrimSpace(tpl.Name),
		Level:       int(tpl.Level),
		Target:      domaincombat.TargetEnemy,
		TargetCount: 1,
		Effects:     []domaincombat.SkillEffect{{Kind: kind, Coefficient: value / 100.0}},
	}
}
