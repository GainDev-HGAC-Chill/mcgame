// Open-sourced by BaoLT

// Fairy skill / state / lifecycle operations.
//
//	onSkill            teach a skill from a Bí Kíp Tinh Linh item -> {flag,id,sindex,skill}
//	                   stores the level-1 TBL_SKILL id (via SkillItemToTblSkill map)
//	upFairySkill       upgrade a learned skill 1->2->3 via TBL_SKILL chain;
//	                   consumes ITEM_FAIRY_SKILL_UP_ITEM=3583 (1 book lv1->2, 3 books lv2->3)
//	fairySkillConfigChange store the active skill config (fire-and-forget, no push)
//	changeFairyState   toggle guard(1)/rest(0) -> push onFairyOn/onFairyOff (+onUpdateFairy)
//	delFairy           release a fairy -> push onDelFairy
//
// skillIndex (int, 1-based = tab*4 + N where N=button 1..4) maps to the persisted
// skillFlag key as sindex = "s"+skillIndex (range s1..s20). The integer->sN mapping
// comes from FairySkillCanvas.as: upSkill(N) with skillIndex = selectedTabIndex*4 + N.
// nextSkillSlot iterates s1..s20 to find the first empty slot.
package fairy

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"

	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

const (
	ItemFairySkillUpItem = 3583
	itemTypeFairySkill   = 521

	fairyStateRest  = 0
	fairyStateGuard = 1

	skillMaxLevel = 3
)

var ErrSkillMaxLevel = errors.New("tinh linh: kỹ năng đã đạt cấp tối đa")

type SkillResult struct {
	Flag   bool
	ID     int64
	Sindex string
	Skill  int
	Code   int
}

type StateChange struct {
	CharID         int64
	FairyID        int64
	State          int
	ResCode        int
	RestedFairyIDs []int64
}

type DeleteResult struct {
	FairyID     int64
	WasGuarding bool
}

func skillIndexToSindex(skillIndex int) string {
	if skillIndex < 1 {
		skillIndex = 1
	}
	return "s" + strconv.Itoa(skillIndex)
}

func (s *Service) OnSkill(ctx context.Context, charID, fairyID, skillBookSlotID int64) (*SkillResult, error) {
	unlock := s.lockCharacterMutation(charID)
	defer unlock()

	state, err := s.loadState(ctx, charID)
	if err != nil {
		return nil, err
	}
	f, ok := state.fairy(fairyID)
	if !ok {
		return &SkillResult{Flag: false, Code: 2}, ErrFairyMissing
	}
	item, err := s.items.GetItemByID(ctx, charID, skillBookSlotID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return &SkillResult{Flag: false, Code: 2}, ErrItemNotFound
	}
	if s.itemData == nil || s.skillData == nil {
		return &SkillResult{Flag: false, Code: 2}, ErrSkillFailed
	}
	book := s.itemData.GetItem(item.TemplateID)
	skillID, mapped := SkillItemToTblSkill[item.TemplateID]
	bookType, validBookType := exactCatalogInt(bookValue(book, func(templateType *models.ItemTemplateTemplate) float64 { return templateType.Type }))
	propType, validPropType := exactCatalogInt(bookValue(book, func(templateType *models.ItemTemplateTemplate) float64 { return templateType.PropType }))
	reqLevel, validReqLevel := exactCatalogInt(bookValue(book, func(templateType *models.ItemTemplateTemplate) float64 { return templateType.ReqLevel }))
	if !mapped || book == nil || !validBookType || bookType != itemTypeFairySkill || !validPropType || propType != skillID || !validReqLevel || reqLevel < 0 {
		return &SkillResult{Flag: false, Code: 2}, ErrItemNotFound
	}
	skill := s.skillData.GetSkill(skillID)
	_, targetLevel, canonical := fairySkillFamily(skillID)
	skillLevel, validSkillLevel := skillTemplateLevel(skill)
	if skill == nil || !canonical || targetLevel != 1 || int(skill.ID) != skillID || !validSkillLevel || skillLevel != 1 {
		return &SkillResult{Flag: false, Code: 2}, ErrSkillFailed
	}
	if gexpToLv(f.Gexp) < reqLevel {
		return &SkillResult{Flag: false, Code: 2}, ErrSkillFailed
	}
	targetFamily, _, _ := fairySkillFamily(skillID)
	for _, learned := range f.SkillFlag {
		learnedFamily, _, ok := fairySkillFamily(learned)
		if ok && learnedFamily == targetFamily {
			return &SkillResult{Flag: false, Code: 1}, ErrSkillDuplicate
		}
	}
	sindex := s.nextSkillSlot(f)
	if sindex == "" {
		return &SkillResult{Flag: false, Code: 2}, ErrSkillFailed
	}
	if _, _, err := s.items.ConsumeItemStackByID(ctx, charID, skillBookSlotID, 1); err != nil {
		return &SkillResult{Flag: false, Code: 2}, err
	}
	f.SkillFlag[sindex] = skillID
	if err := s.saveState(ctx, charID, state); err != nil {
		if refundErr := s.refundBook(ctx, charID, item.TemplateID, item.ItemType, 1, item.IsBound); refundErr != nil {
			return nil, fmt.Errorf("persist learned fairy skill: %w; refund book: %v", err, refundErr)
		}
		return nil, err
	}
	return &SkillResult{Flag: true, ID: fairyID, Sindex: sindex, Skill: skillID}, nil
}

func (s *Service) UpSkill(ctx context.Context, charID, fairyID int64, skillIndex int) (*SkillResult, error) {
	unlock := s.lockCharacterMutation(charID)
	defer unlock()

	state, err := s.loadState(ctx, charID)
	if err != nil {
		return nil, err
	}
	f, ok := state.fairy(fairyID)
	if !ok {
		return &SkillResult{Flag: false, Code: 2}, ErrFairyMissing
	}
	if skillIndex < 1 || skillIndex > 20 {
		return &SkillResult{Flag: false, Code: 2}, ErrSkillFailed
	}
	sindex := skillIndexToSindex(skillIndex)
	currentID, exists := f.SkillFlag[sindex]
	if !exists {
		return &SkillResult{Flag: false, Code: 2}, ErrSkillFailed
	}
	if s.skillData == nil {
		return &SkillResult{Flag: false, Code: 2}, ErrSkillFailed
	}
	cur := s.skillData.GetSkill(currentID)
	_, currentLevel, canonical := fairySkillFamily(currentID)
	catalogCurrentLevel, validCurrentLevel := skillTemplateLevel(cur)
	if cur == nil || !canonical || int(cur.ID) != currentID || !validCurrentLevel || catalogCurrentLevel != currentLevel {
		return &SkillResult{Flag: false, Code: 2}, ErrSkillFailed
	}
	if currentLevel >= skillMaxLevel {
		return &SkillResult{Flag: false, Code: 3}, ErrSkillMaxLevel
	}
	newID, ok := nextFairySkill(currentID)
	if !ok {
		return &SkillResult{Flag: false, Code: 2}, ErrSkillFailed
	}
	next := s.skillData.GetSkill(newID)
	catalogNextLevel, validNextLevel := skillTemplateLevel(next)
	if next == nil || int(next.ID) != newID || !validNextLevel || catalogNextLevel != currentLevel+1 {
		return &SkillResult{Flag: false, Code: 2}, ErrSkillFailed
	}
	bookCount := 1
	if currentLevel == 2 {
		bookCount = 3
	}
	ok2, err := s.items.ConsumeItemsByTemplateID(ctx, charID, ItemFairySkillUpItem, bookCount)
	if err != nil {
		return &SkillResult{Flag: false, Code: 2}, err
	}
	if !ok2 {
		return &SkillResult{Flag: false, Code: 2}, ErrSkillFailed
	}
	f.SkillFlag[sindex] = newID
	if err := s.saveState(ctx, charID, state); err != nil {
		if refundErr := s.refundBook(ctx, charID, ItemFairySkillUpItem, domainitem.ItemTypeConsumable, bookCount, true); refundErr != nil {
			return nil, fmt.Errorf("persist upgraded fairy skill: %w; refund books: %v", err, refundErr)
		}
		return nil, err
	}
	return &SkillResult{Flag: true, ID: fairyID, Sindex: sindex, Skill: newID}, nil
}

func (s *Service) SetSkillConfig(ctx context.Context, charID, fairyID int64, config map[string]interface{}) error {
	unlock := s.lockCharacterMutation(charID)
	defer unlock()

	state, err := s.loadState(ctx, charID)
	if err != nil {
		return err
	}
	f, ok := state.fairy(fairyID)
	if !ok {
		return ErrFairyMissing
	}
	validated, err := s.validateSkillConfig(f, config)
	if err != nil {
		return err
	}
	f.Config = validated
	return s.saveState(ctx, charID, state)
}

func (s *Service) ChangeState(ctx context.Context, charID, fairyID int64, newState int) (*StateChange, error) {
	unlock := s.lockCharacterMutation(charID)
	defer unlock()

	state, err := s.loadState(ctx, charID)
	if err != nil {
		return nil, err
	}
	f, ok := state.fairy(fairyID)
	if !ok {
		return nil, ErrFairyMissing
	}
	if newState != fairyStateGuard {
		newState = fairyStateRest
	}
	rested := make([]int64, 0)
	if newState == fairyStateGuard {
		for id, other := range state.Fairies {
			if id != fairyID && other != nil && other.State == fairyStateGuard {
				other.State = fairyStateRest
				rested = append(rested, id)
			}
		}
	}
	f.State = newState
	if err := s.saveState(ctx, charID, state); err != nil {
		return nil, err
	}
	resCode := 0
	if newState == fairyStateGuard {
		resCode = s.skinResCode(ctx, charID, f.Tid)
	}
	return &StateChange{CharID: charID, FairyID: fairyID, State: newState, ResCode: resCode, RestedFairyIDs: rested}, nil
}

func (s *Service) Delete(ctx context.Context, charID, fairyID int64) (*DeleteResult, error) {
	unlock := s.lockCharacterMutation(charID)
	defer unlock()

	state, err := s.loadState(ctx, charID)
	if err != nil {
		return nil, err
	}
	f, ok := state.fairy(fairyID)
	if !ok {
		return nil, ErrFairyMissing
	}
	delete(state.Fairies, fairyID)
	if err := s.saveState(ctx, charID, state); err != nil {
		return nil, err
	}
	return &DeleteResult{FairyID: fairyID, WasGuarding: f.State == fairyStateGuard}, nil
}

func (s *Service) refundBook(ctx context.Context, charID int64, templateID int, itemType domainitem.ItemType, count int, bound bool) error {
	if itemType == 0 {
		itemType = domainitem.ItemTypeConsumable
	}
	_, err := s.items.AddItemWithBind(ctx, charID, templateID, itemType, count, bound)
	return err
}

func (s *Service) validateSkillConfig(f *Fairy, config map[string]interface{}) (map[string]interface{}, error) {
	if s.skillData == nil || len(config) != 4 {
		return nil, ErrSkillFailed
	}
	selected, ok := exactInt(config[skillConfigSel])
	if !ok || selected < -1 || selected > 2 {
		return nil, ErrSkillFailed
	}
	out := map[string]interface{}{skillConfigSel: selected}
	for groupIndex := 0; groupIndex < 3; groupIndex++ {
		groupKey := strconv.Itoa(groupIndex)
		group, ok := config[groupKey].(map[string]interface{})
		if !ok || len(group) != 5 {
			return nil, ErrSkillFailed
		}
		normalized := make(map[string]interface{}, 5)
		families := map[int]struct{}{}
		for slotIndex := 0; slotIndex < 5; slotIndex++ {
			slotKey := "c" + strconv.Itoa(slotIndex)
			skillID, ok := exactInt(group[slotKey])
			if !ok || skillID < -1 || skillID == 0 {
				return nil, ErrSkillFailed
			}
			normalized[slotKey] = skillID
			if skillID == -1 {
				continue
			}
			if !fairyLearnedSkill(f, skillID) {
				return nil, ErrSkillFailed
			}
			family, _, canonical := fairySkillFamily(skillID)
			template := s.skillData.GetSkill(skillID)
			kind, validKind := skillTemplateKind(template)
			if !canonical || template == nil || int(template.ID) != skillID || !validKind || kind != 1 {
				return nil, ErrSkillFailed
			}
			if _, duplicate := families[family]; duplicate {
				return nil, ErrSkillDuplicate
			}
			families[family] = struct{}{}
		}
		out[groupKey] = normalized
	}
	return out, nil
}

func bookValue(book *models.ItemTemplateTemplate, value func(*models.ItemTemplateTemplate) float64) float64 {
	if book == nil {
		return math.NaN()
	}
	return value(book)
}

func skillTemplateLevel(skill *models.SkillTemplate) (int, bool) {
	if skill == nil {
		return 0, false
	}
	return exactCatalogInt(skill.Level)
}

func skillTemplateKind(skill *models.SkillTemplate) (int, bool) {
	if skill == nil {
		return 0, false
	}
	return exactCatalogInt(skill.Kind)
}

func exactCatalogInt(value float64) (int, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value {
		return 0, false
	}
	return int(value), true
}

func fairyLearnedSkill(f *Fairy, skillID int) bool {
	for _, learned := range f.SkillFlag {
		if learned == skillID {
			return true
		}
	}
	return false
}

func exactInt(value interface{}) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if math.Trunc(typed) != typed {
			return 0, false
		}
		return int(typed), true
	case float32:
		if float32(math.Trunc(float64(typed))) != typed {
			return 0, false
		}
		return int(typed), true
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	default:
		return 0, false
	}
}

func (s *Service) nextSkillSlot(f *Fairy) string {
	for i := 0; i < 20; i++ {
		key := "s" + strconv.Itoa(i+1)
		if _, ok := f.SkillFlag[key]; !ok {
			return key
		}
	}
	return ""
}

func (s *Service) skinResCode(ctx context.Context, charID int64, tid int) int {
	skin, err := s.loadSkinState(ctx, charID)
	if err != nil {
		return tid
	}
	if skin.ActFairy > 0 {
		return skin.ActFairy
	}
	return tid
}
