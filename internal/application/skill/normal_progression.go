// Open-sourced by BaoLT

// Normal character-skill catalog resolution and authoritative progression.
package skill

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domainchar "mcgame-server/internal/domain/character"
	domainskill "mcgame-server/internal/domain/skill"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

var (
	ErrNormalSkillBookNotFound  = errors.New("normal skill book mapping not found")
	ErrNormalSkillNotLearned    = errors.New("normal skill not learned")
	ErrNormalSkillInvalidStep   = errors.New("normal skill upgrade must target the next level")
	ErrSkillExperienceNotEnough = errors.New("skill experience not enough")
)

type UpgradeResult struct {
	Skill      *domainskill.CharacterSkill
	Character  *domainchar.Character
	Changed    bool
	ExpSpent   int64
	MoneySpent int64
	MoneyKey   string
}

func (s *Service) lockCharacterMutation(charID int64) func() {
	lock := &s.mutationLocks[uint64(charID)%uint64(len(s.mutationLocks))]
	lock.Lock()
	return lock.Unlock
}

func (s *Service) ResolveOrdinarySkillBook(templateID int) (*models.SkillTemplate, error) {
	if s == nil || s.gameData == nil {
		return nil, ErrNormalSkillBookNotFound
	}
	book := s.gameData.GetItem(templateID)
	if book == nil || int(book.Type) != 505 {
		return nil, ErrNormalSkillBookNotFound
	}
	name, ok := ordinarySkillBookName(book.Name)
	if !ok {
		return nil, ErrNormalSkillBookNotFound
	}
	candidates := make([]*models.SkillTemplate, 0, 1)
	for _, template := range s.gameData.GetAllSkills() {
		if !isOrdinarySkillTemplate(template) || skillTemplateLevel(template) != 1 {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(template.Name), name) {
			continue
		}
		if strings.TrimSpace(template.ReqClass) != strings.TrimSpace(book.ReqClass) {
			continue
		}
		candidates = append(candidates, template)
	}
	if len(candidates) != 1 {
		return nil, ErrNormalSkillBookNotFound
	}
	return candidates[0], nil
}

func (s *Service) LearnOrdinarySkillByBook(ctx context.Context, charID int64, templateID int) (*domainskill.CharacterSkill, *models.SkillTemplate, error) {
	unlock := s.lockCharacterMutation(charID)
	defer unlock()

	template, err := s.ResolveOrdinarySkillBook(templateID)
	if err != nil {
		return nil, nil, err
	}
	validated, char, err := s.validateNormalSkillRequest(ctx, charID, int(template.ID), false)
	if err != nil {
		return nil, nil, err
	}
	learned, err := s.skillRepo.FindByCharacterID(ctx, charID)
	if err != nil {
		return nil, nil, err
	}
	if current, _ := s.findSkillFamilyEntry(learned, validated); current != nil {
		return nil, nil, pkgerrors.ErrAlreadyExists
	}
	entry, err := s.createNormalSkill(ctx, charID, int(validated.ID), validated, char)
	if err != nil {
		return nil, nil, err
	}
	return entry, validated, nil
}

func (s *Service) UpgradeSkillWithResult(ctx context.Context, charID int64, requestedSkillID int) (*UpgradeResult, error) {
	unlock := s.lockCharacterMutation(charID)
	defer unlock()

	requestedTemplate, char, err := s.validateNormalSkillRequest(ctx, charID, requestedSkillID, false)
	if err != nil {
		return nil, err
	}
	if !isOrdinarySkillTemplate(requestedTemplate) {
		return nil, pkgerrors.ErrInvalidInput
	}
	exact, err := s.skillRepo.FindByCharacterAndSkill(ctx, charID, requestedSkillID)
	if err == nil && exact != nil {
		return &UpgradeResult{Skill: exact, Character: char}, nil
	}
	if err != nil && !pkgerrors.Is(err, pkgerrors.ErrNotFound) {
		return nil, err
	}
	learnedSkills, err := s.skillRepo.FindByCharacterID(ctx, charID)
	if err != nil {
		return nil, err
	}
	current, currentTemplate := s.findSkillFamilyEntry(learnedSkills, requestedTemplate)
	if current == nil || currentTemplate == nil {
		return nil, ErrNormalSkillNotLearned
	}
	if !sameOrdinarySkillFamily(currentTemplate, requestedTemplate) || skillTemplateLevel(requestedTemplate) != skillTemplateLevel(currentTemplate)+1 {
		return nil, ErrNormalSkillInvalidStep
	}
	expCost := positiveInt64(requestedTemplate.ExpSkill)
	moneyCost := positiveInt64(requestedTemplate.Price)
	if char.Experience < expCost {
		return nil, ErrSkillExperienceNotEnough
	}
	moneyKey, currentMoney := selectedSkillMoney(char)
	if currentMoney < moneyCost {
		return nil, pkgerrors.ErrInsufficientFunds
	}
	oldSkillID := current.SkillID
	oldLevel := current.Level
	current.SkillID = requestedSkillID
	current.Level = skillTemplateLevel(requestedTemplate)
	if err := s.skillRepo.Update(ctx, current); err != nil {
		return nil, err
	}
	oldExperience := char.Experience
	oldMoney := currentMoney
	char.Experience -= expCost
	setSelectedSkillMoney(char, moneyKey, oldMoney-moneyCost)
	if err := s.charRepo.Update(ctx, char); err != nil {
		char.Experience = oldExperience
		setSelectedSkillMoney(char, moneyKey, oldMoney)
		current.SkillID = oldSkillID
		current.Level = oldLevel
		if rollbackErr := s.skillRepo.Update(ctx, current); rollbackErr != nil {
			return nil, fmt.Errorf("persist skill resources: %w; rollback skill: %v", err, rollbackErr)
		}
		return nil, err
	}
	s.logger.Info("Character upgraded skill",
		zap.Int64("character_id", charID),
		zap.Int("skill_id", requestedSkillID),
		zap.Int("skill_level", current.Level),
		zap.Int64("exp_spent", expCost),
		zap.Int64("money_spent", moneyCost))
	return &UpgradeResult{
		Skill: current, Character: char, Changed: true,
		ExpSpent: expCost, MoneySpent: moneyCost, MoneyKey: moneyKey,
	}, nil
}

func ordinarySkillBookName(raw string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(raw), "-", 2)
	if len(parts) != 2 || !strings.EqualFold(strings.TrimSpace(parts[0]), "Sách Kỹ Năng") {
		return "", false
	}
	name := strings.TrimSpace(parts[1])
	return name, name != ""
}

func isOrdinarySkillTemplate(template *models.SkillTemplate) bool {
	if template == nil {
		return false
	}
	env := int(template.UseEnv)
	kind := int(template.Kind)
	return (env == 0 && kind == 1) || (env == 1 && kind == 2)
}

func sameOrdinarySkillFamily(current, requested *models.SkillTemplate) bool {
	if !isOrdinarySkillTemplate(current) || !isOrdinarySkillTemplate(requested) {
		return false
	}
	return current.CodeName != "" && current.CodeName == requested.CodeName &&
		int(current.UseEnv) == int(requested.UseEnv) && current.ReqClass == requested.ReqClass
}

func positiveInt64(value float64) int64 {
	if value <= 0 {
		return 0
	}
	return int64(value)
}

func selectedSkillMoney(char *domainchar.Character) (string, int64) {
	if char.SelectedMoneyType == 2 {
		return "money", char.Money
	}
	return "moneyBind", char.MoneyBind
}

func setSelectedSkillMoney(char *domainchar.Character, key string, value int64) {
	if key == "money" {
		char.Money = value
		return
	}
	char.MoneyBind = value
}
