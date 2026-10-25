// Open-sourced by BaoLT

// Fairy state reader proves roster, growth, skill, guard, and skin milestones.
package achievement

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	appfairy "mcgame-server/internal/application/fairy"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"
)

type FairyStateCatalog interface {
	GetAllFairyTempaltes() []*models.FairyTempalteTemplate
}

type fairyIdentityRequirement struct {
	requirementID int64
	templateID    int
	name          string
	requireGuard  bool
}

var fairyIdentityRequirements = [...]fairyIdentityRequirement{
	{500, 1, "Thiên Sứ Poli", true},
	{501, 6, "Yêu Tinh Nicole", true},
	{502, 2, "Rocky", true},
	{503, 5, "Rồng Xanh Abu", true},
	{504, 3, "Băng Nữ Koura", true},
	{509, 4, "Puca", true},
	{524, 7, "Tuần Lộc Miya", true},
	{536, 8, "Thiên Sứ Bụt", true},
	{537, 9, "Minions", true},
	{560, 11, "Thần Chết", false},
	{561, 10, "Casper", false},
}

type fairyAchievementState struct {
	templateID int
	level      int
	growLevel  int
	guarding   bool
	skillCount int
}

type FairyStateReader struct {
	repo    CharacterFeatureStateRepository
	catalog FairyStateCatalog
}

func NewFairyStateReader(repo CharacterFeatureStateRepository, catalog FairyStateCatalog) *FairyStateReader {
	return &FairyStateReader{repo: repo, catalog: catalog}
}

func (r *FairyStateReader) Name() string {
	return "fairy"
}

func (r *FairyStateReader) RequirementIDs() []int64 {
	return []int64{500, 501, 502, 503, 504, 505, 506, 507, 508, 509, 523, 524, 525, 536, 537, 560, 561}
}

func (r *FairyStateReader) ReadStateProofs(ctx context.Context, characterID int64) ([]StateProof, error) {
	if r == nil || r.repo == nil || r.catalog == nil {
		return nil, fmt.Errorf("fairy state dependencies not configured")
	}
	catalog, err := validatedFairyCatalog(r.catalog.GetAllFairyTempaltes())
	if err != nil {
		return nil, err
	}
	states, err := r.repo.ListCharacterFeatureStates(ctx, characterID)
	if err != nil {
		return nil, err
	}
	var roster []*fairyAchievementState
	skinCount := 0
	foundRoster := false
	foundSkin := false
	for _, featureState := range states {
		if featureState == nil || featureState.CharacterID != characterID {
			continue
		}
		switch featureState.FeatureKey {
		case domainfeature.FeatureFairy:
			if foundRoster {
				return nil, fmt.Errorf("duplicate fairy roster state for character %d", characterID)
			}
			foundRoster = true
			roster, err = decodeFairyAchievementRoster(featureState.State, catalog)
			if err != nil {
				return nil, err
			}
		case domainfeature.FeatureFairySkin:
			if foundSkin {
				return nil, fmt.Errorf("duplicate fairy skin state for character %d", characterID)
			}
			foundSkin = true
			skinCount, err = decodeFairyAchievementSkins(featureState.State, catalog)
			if err != nil {
				return nil, err
			}
		}
	}
	owned := make(map[int]bool)
	maxLevel := 0
	maxGrowLevel := 0
	maxSkills := 0
	for _, fairy := range roster {
		if fairy.guarding || !owned[fairy.templateID] {
			owned[fairy.templateID] = fairy.guarding
		}
		if fairy.level > maxLevel {
			maxLevel = fairy.level
		}
		if fairy.growLevel > maxGrowLevel {
			maxGrowLevel = fairy.growLevel
		}
		if fairy.skillCount > maxSkills {
			maxSkills = fairy.skillCount
		}
	}
	proofs := make([]StateProof, 0, 17)
	for _, definition := range fairyIdentityRequirements {
		guarding, exists := owned[definition.templateID]
		progress := 0
		if exists && (!definition.requireGuard || guarding) {
			progress = 1
		}
		proofs = append(proofs, stateProof(definition.requirementID, progress, 1))
	}
	proofs = append(proofs,
		stateProof(505, maxLevel, 10),
		stateProof(506, maxLevel, 20),
		stateProof(507, maxGrowLevel, 21),
		stateProof(508, maxGrowLevel, 41),
		stateProof(523, maxSkills, 6),
		stateProof(525, skinCount, 5),
	)
	return proofs, nil
}

func validatedFairyCatalog(entries []*models.FairyTempalteTemplate) (map[int]*models.FairyTempalteTemplate, error) {
	if len(entries) != 17 {
		return nil, fmt.Errorf("fairy catalog count = %d, want 17", len(entries))
	}
	byID := make(map[int]*models.FairyTempalteTemplate, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.ID <= 0 || entry.ID > 17 {
			return nil, fmt.Errorf("fairy catalog contains invalid entry")
		}
		id := int(entry.ID)
		if _, duplicate := byID[id]; duplicate {
			return nil, fmt.Errorf("fairy catalog repeats entry %d", id)
		}
		byID[id] = entry
	}
	for _, definition := range fairyIdentityRequirements {
		entry := byID[definition.templateID]
		if entry == nil || entry.Name != definition.name {
			return nil, fmt.Errorf("fairy requirement %d identity drift", definition.requirementID)
		}
	}
	return byID, nil
}

func decodeFairyAchievementRoster(raw map[string]interface{}, catalog map[int]*models.FairyTempalteTemplate) ([]*fairyAchievementState, error) {
	rosterRaw := raw
	if wrapped, exists := raw["fairy"]; exists {
		var ok bool
		rosterRaw, ok = wrapped.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid fairy roster wrapper")
		}
	}
	roster := make([]*fairyAchievementState, 0, len(rosterRaw))
	templates := make(map[int]struct{})
	guardCount := 0
	for rawID, rawEntry := range rosterRaw {
		instanceID, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil || instanceID <= 0 {
			return nil, fmt.Errorf("invalid fairy instance id %q", rawID)
		}
		entry, ok := rawEntry.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid fairy instance %d", instanceID)
		}
		if rawPersistedID, exists := entry["id"]; exists {
			persistedID, ok := exactStateInt64(rawPersistedID)
			if !ok || persistedID != instanceID {
				return nil, fmt.Errorf("fairy instance %d identity mismatch", instanceID)
			}
		}
		templateID, ok := stateInt(entry["tid"])
		if !ok || catalog[templateID] == nil {
			return nil, fmt.Errorf("fairy instance %d has invalid template", instanceID)
		}
		if _, duplicate := templates[templateID]; duplicate {
			return nil, fmt.Errorf("fairy template %d is owned more than once", templateID)
		}
		templates[templateID] = struct{}{}
		exp, ok := exactStateInt64(entry["exp"])
		if !ok || exp < 0 {
			return nil, fmt.Errorf("fairy instance %d has invalid experience", instanceID)
		}
		growExp, ok := exactStateInt64(entry["gexp"])
		if !ok || growExp < 0 {
			return nil, fmt.Errorf("fairy instance %d has invalid growth experience", instanceID)
		}
		state, ok := stateInt(entry["state"])
		if !ok || (state != 0 && state != 1) {
			return nil, fmt.Errorf("fairy instance %d has invalid guard state", instanceID)
		}
		if state == 1 {
			guardCount++
			if guardCount > 1 {
				return nil, fmt.Errorf("multiple guarding fairies")
			}
		}
		skillCount, err := decodeFairyAchievementSkills(entry["skillFlag"], instanceID)
		if err != nil {
			return nil, err
		}
		roster = append(roster, &fairyAchievementState{
			templateID: templateID,
			level:      appfairy.LevelFromExperience(exp),
			growLevel:  appfairy.GrowthLevelFromExperience(growExp),
			guarding:   state == 1,
			skillCount: skillCount,
		})
	}
	return roster, nil
}

func decodeFairyAchievementSkills(raw interface{}, instanceID int64) (int, error) {
	if raw == nil {
		return 0, nil
	}
	skills, ok := raw.(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("fairy instance %d has invalid skill state", instanceID)
	}
	seen := make(map[int]struct{}, len(skills))
	for rawSlot, rawSkillID := range skills {
		if len(rawSlot) < 2 || rawSlot[0] != 's' {
			return 0, fmt.Errorf("fairy instance %d has invalid skill slot %q", instanceID, rawSlot)
		}
		slot, err := strconv.Atoi(rawSlot[1:])
		if err != nil || slot < 1 || slot > 20 {
			return 0, fmt.Errorf("fairy instance %d has invalid skill slot %q", instanceID, rawSlot)
		}
		skillID, ok := stateInt(rawSkillID)
		if !ok || !appfairy.IsFairySkill(skillID) {
			return 0, fmt.Errorf("fairy instance %d has invalid skill %d", instanceID, skillID)
		}
		if _, duplicate := seen[skillID]; duplicate {
			return 0, fmt.Errorf("fairy instance %d repeats skill %d", instanceID, skillID)
		}
		seen[skillID] = struct{}{}
	}
	return len(seen), nil
}

func decodeFairyAchievementSkins(raw map[string]interface{}, catalog map[int]*models.FairyTempalteTemplate) (int, error) {
	inner := raw
	if wrapped, exists := raw["fairy_skin"]; exists {
		var ok bool
		inner, ok = wrapped.(map[string]interface{})
		if !ok {
			return 0, fmt.Errorf("invalid fairy skin wrapper")
		}
	}
	list := ""
	if rawList, exists := inner["actFairyList"]; exists {
		var ok bool
		list, ok = rawList.(string)
		if !ok {
			return 0, fmt.Errorf("invalid fairy skin list")
		}
	}
	owned := make(map[int]struct{})
	for _, rawID := range strings.Split(list, "|") {
		rawID = strings.TrimSpace(rawID)
		if rawID == "" {
			continue
		}
		templateID, err := strconv.Atoi(rawID)
		if err != nil || catalog[templateID] == nil {
			return 0, fmt.Errorf("invalid fairy skin id %q", rawID)
		}
		if _, duplicate := owned[templateID]; duplicate {
			return 0, fmt.Errorf("duplicate fairy skin %d", templateID)
		}
		owned[templateID] = struct{}{}
	}
	if rawActive, exists := inner["actFairy"]; exists {
		active, ok := stateInt(rawActive)
		if !ok || active < 0 || (active > 0 && catalog[active] == nil) {
			return 0, fmt.Errorf("invalid active fairy skin")
		}
		if active > 0 {
			if _, exists := owned[active]; !exists {
				return 0, fmt.Errorf("active fairy skin %d is not owned", active)
			}
		}
	}
	return len(owned), nil
}

func exactStateInt64(value interface{}) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		result := int64(typed)
		return result, float64(result) == typed
	default:
		return 0, false
	}
}
