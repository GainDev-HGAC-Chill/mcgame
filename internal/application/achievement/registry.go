// Open-sourced by BaoLT

// Achievement registry validates the enabled catalog against audited authority classifications.
package achievement

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"

	"mcgame-server/internal/gamedata/models"
)

type RequirementAuthority string

const (
	RequirementAuthorityState       RequirementAuthority = "state"
	RequirementAuthorityEvent       RequirementAuthority = "event"
	RequirementAuthorityUnsupported RequirementAuthority = "unsupported"
)

type RequirementTargetSource string

const (
	RequirementTargetCatalog   RequirementTargetSource = "catalog"
	RequirementTargetAuthority RequirementTargetSource = "authority"
)

type RequirementDefinition struct {
	RequirementID int64
	AchievementID int64
	Type          int
	Target        int
	TargetSource  RequirementTargetSource
	Authority     RequirementAuthority
}

type RequirementRegistry struct {
	definitions map[int64]RequirementDefinition
}

type RequirementCatalog interface {
	GetAllAchievements() []*models.AchievementTemplate
	GetAllAchievementRequires() []*models.AchievementRequireTemplate
	GetAchievementRequiresByAchievementID(achievementID int) []*models.AchievementRequireTemplate
}

type RegistryStats struct {
	EnabledAchievements         int
	EnabledRequirements         int
	AllRequirementTypes         int
	StateRequirements           int
	EventRequirements           int
	UnsupportedRequirements     int
	MultiRequirementParents     int
	MultiRequirementRows        int
	EmptyRequirementParents     int
	AuthorityTargetRequirements int
}

const auditedAchievementCatalogFingerprint = "4e34c010c42554ea93f59f6d3b93ada4a36f25151ec6fc4bcd38e3c18dedb8ff"

func BuildRequirementRegistry(catalog RequirementCatalog) (*RequirementRegistry, RegistryStats, string, error) {
	if catalog == nil {
		return nil, RegistryStats{}, "", fmt.Errorf("achievement catalog not configured")
	}
	achievements := append([]*models.AchievementTemplate(nil), catalog.GetAllAchievements()...)
	requirements := append([]*models.AchievementRequireTemplate(nil), catalog.GetAllAchievementRequires()...)
	sort.Slice(achievements, func(i, j int) bool { return achievements[i].ID < achievements[j].ID })
	sort.Slice(requirements, func(i, j int) bool { return requirements[i].ID < requirements[j].ID })

	stats := RegistryStats{}
	allTypes := make(map[int]struct{})
	enabledParents := make(map[int64]struct{})
	for _, requirement := range requirements {
		if requirement == nil {
			return nil, stats, "", fmt.Errorf("nil achievement requirement")
		}
		typeID, err := exactCatalogInt(requirement.Type, "requirement type", requirement.ID)
		if err != nil {
			return nil, stats, "", err
		}
		allTypes[typeID] = struct{}{}
	}

	for _, achievement := range achievements {
		if achievement == nil {
			return nil, stats, "", fmt.Errorf("nil achievement template")
		}
		if achievement.Enable != 1 {
			continue
		}
		stats.EnabledAchievements++
		enabledParents[achievement.ID] = struct{}{}
	}

	registry := &RequirementRegistry{definitions: make(map[int64]RequirementDefinition)}
	parentCounts := make(map[int64]int)
	for _, requirement := range requirements {
		parentID, err := exactCatalogInt64(requirement.Aid, "requirement parent", requirement.ID)
		if err != nil {
			return nil, stats, "", err
		}
		if _, enabled := enabledParents[parentID]; !enabled {
			continue
		}
		typeID, err := exactCatalogInt(requirement.Type, "requirement type", requirement.ID)
		if err != nil {
			return nil, stats, "", err
		}
		target, err := exactCatalogInt(requirement.Num, "requirement target", requirement.ID)
		if err != nil {
			return nil, stats, "", err
		}
		if target < 0 {
			return nil, stats, "", fmt.Errorf("requirement %d has negative target %d", requirement.ID, target)
		}
		if _, exists := registry.definitions[requirement.ID]; exists {
			return nil, stats, "", fmt.Errorf("duplicate achievement requirement %d", requirement.ID)
		}
		targetSource := RequirementTargetCatalog
		if target == 0 {
			targetSource = RequirementTargetAuthority
			stats.AuthorityTargetRequirements++
		}
		authority := classifyRequirementAuthority(requirement.ID)
		registry.definitions[requirement.ID] = RequirementDefinition{
			RequirementID: requirement.ID,
			AchievementID: parentID,
			Type:          typeID,
			Target:        target,
			TargetSource:  targetSource,
			Authority:     authority,
		}
		parentCounts[parentID]++
		stats.EnabledRequirements++
		switch authority {
		case RequirementAuthorityState:
			stats.StateRequirements++
		case RequirementAuthorityEvent:
			stats.EventRequirements++
		case RequirementAuthorityUnsupported:
			stats.UnsupportedRequirements++
		default:
			return nil, stats, "", fmt.Errorf("requirement %d has invalid authority %q", requirement.ID, authority)
		}
	}

	for parentID := range enabledParents {
		count := parentCounts[parentID]
		if count == 0 {
			stats.EmptyRequirementParents++
		}
		if count > 1 {
			stats.MultiRequirementParents++
			stats.MultiRequirementRows += count
		}
	}
	stats.AllRequirementTypes = len(allTypes)
	return registry, stats, achievementCatalogFingerprint(achievements, requirements, enabledParents), nil
}

func ValidateRequirementCatalog(catalog RequirementCatalog) error {
	registry, stats, fingerprint, err := BuildRequirementRegistry(catalog)
	if err != nil {
		return err
	}
	if len(registry.definitions) != 697 || stats.EnabledAchievements != 566 || stats.EnabledRequirements != 697 {
		return fmt.Errorf("achievement catalog census drift: achievements=%d requirements=%d", stats.EnabledAchievements, stats.EnabledRequirements)
	}
	if stats.AllRequirementTypes != 60 || stats.StateRequirements != 472 || stats.EventRequirements != 74 || stats.UnsupportedRequirements != 151 {
		return fmt.Errorf("achievement authority census drift: types=%d state=%d event=%d unsupported=%d", stats.AllRequirementTypes, stats.StateRequirements, stats.EventRequirements, stats.UnsupportedRequirements)
	}
	if stats.MultiRequirementParents != 30 || stats.MultiRequirementRows != 163 || stats.EmptyRequirementParents != 2 || stats.AuthorityTargetRequirements != 550 {
		return fmt.Errorf("achievement structure drift: multi=%d rows=%d empty=%d authority_targets=%d", stats.MultiRequirementParents, stats.MultiRequirementRows, stats.EmptyRequirementParents, stats.AuthorityTargetRequirements)
	}
	if fingerprint != auditedAchievementCatalogFingerprint {
		return fmt.Errorf("achievement catalog fingerprint drift: got %s", fingerprint)
	}
	return validateDirectCompletionParents(catalog)
}

func (r *RequirementRegistry) Get(requirementID int64) (RequirementDefinition, bool) {
	if r == nil {
		return RequirementDefinition{}, false
	}
	definition, ok := r.definitions[requirementID]
	return definition, ok
}

func (r *RequirementRegistry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.definitions)
}

func validateDirectCompletionParents(catalog RequirementCatalog) error {
	emptyIDs := make([]int64, 0, 2)
	for _, achievement := range catalog.GetAllAchievements() {
		if achievement == nil || achievement.Enable != 1 {
			continue
		}
		if len(catalog.GetAchievementRequiresByAchievementID(int(achievement.ID))) == 0 {
			emptyIDs = append(emptyIDs, achievement.ID)
		}
	}
	sort.Slice(emptyIDs, func(i, j int) bool { return emptyIDs[i] < emptyIDs[j] })
	if len(emptyIDs) != 2 || emptyIDs[0] != 6438 || emptyIDs[1] != 6439 {
		return fmt.Errorf("achievement direct-completion parents drift: %v", emptyIDs)
	}
	return nil
}

func achievementCatalogFingerprint(achievements []*models.AchievementTemplate, requirements []*models.AchievementRequireTemplate, enabledParents map[int64]struct{}) string {
	hash := sha256.New()
	for _, achievement := range achievements {
		if achievement == nil || achievement.Enable != 1 {
			continue
		}
		fmt.Fprintf(hash, "a:%d:%s:%s\n", achievement.ID, catalogNumber(achievement.Type), catalogNumber(achievement.Kind))
	}
	for _, requirement := range requirements {
		if requirement == nil {
			continue
		}
		parentID := int64(requirement.Aid)
		if _, enabled := enabledParents[parentID]; !enabled {
			continue
		}
		fmt.Fprintf(hash, "r:%d:%s:%s:%s\n", requirement.ID, catalogNumber(requirement.Aid), catalogNumber(requirement.Type), catalogNumber(requirement.Num))
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func catalogNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func exactCatalogInt(value float64, field string, requirementID int64) (int, error) {
	result := int(value)
	if float64(result) != value {
		return 0, fmt.Errorf("requirement %d has fractional %s %s", requirementID, field, catalogNumber(value))
	}
	return result, nil
}

func exactCatalogInt64(value float64, field string, requirementID int64) (int64, error) {
	result := int64(value)
	if float64(result) != value {
		return 0, fmt.Errorf("requirement %d has fractional %s %s", requirementID, field, catalogNumber(value))
	}
	return result, nil
}

func classifyRequirementAuthority(requirementID int64) RequirementAuthority {
	if isUnsupportedRequirement(requirementID) {
		return RequirementAuthorityUnsupported
	}
	if isEventRequirement(requirementID) {
		return RequirementAuthorityEvent
	}
	return RequirementAuthorityState
}

func between(value, first, last int64) bool {
	return value >= first && value <= last
}

func isEventRequirement(id int64) bool {
	switch {
	case between(id, 53, 56), id == 83, between(id, 102, 109), between(id, 119, 127):
		return true
	case between(id, 150, 157), between(id, 185, 211), between(id, 221, 223):
		return true
	case id == 226, between(id, 231, 251), between(id, 254, 258):
		return true
	case between(id, 270, 275), between(id, 277, 280), between(id, 287, 288), between(id, 290, 294):
		return true
	case id == 326, id == 338, between(id, 348, 353):
		return true
	case id == 376, between(id, 379, 382), between(id, 404, 405), between(id, 423, 424):
		return true
	case id == 514, id == 518, between(id, 526, 534), between(id, 570, 575):
		return true
	case between(id, 745, 750), id == 753:
		return true
	default:
		return false
	}
}

func isUnsupportedRequirement(id int64) bool {
	switch {
	case id == 21, between(id, 132, 133), id == 289, between(id, 7499, 7504):
		return true
	case id == 83, between(id, 102, 109), between(id, 119, 127), id == 150:
		return true
	case between(id, 221, 223), between(id, 244, 247), between(id, 255, 258):
		return true
	case between(id, 270, 275), between(id, 277, 280), between(id, 287, 288), between(id, 290, 294):
		return true
	case id == 338, between(id, 348, 353), id == 376, between(id, 379, 382):
		return true
	case between(id, 404, 405), id == 518, between(id, 745, 750), id == 753:
		return true
	case between(id, 138, 141), between(id, 161, 162), between(id, 164, 165), between(id, 213, 219):
		return true
	case between(id, 333, 334), between(id, 389, 391), between(id, 396, 397):
		return true
	case id == 517, between(id, 519, 522):
		return true
	case id == 538, between(id, 562, 568), between(id, 576, 581), between(id, 585, 591):
		return true
	case id == 462, between(id, 659, 666), between(id, 726, 733), between(id, 737, 740):
		return true
	case between(id, 751, 752), id == 760:
		return true
	default:
		return false
	}
}
