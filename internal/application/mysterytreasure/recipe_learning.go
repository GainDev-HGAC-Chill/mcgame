// Open-sourced by BaoLT

package mysterytreasure

import (
	"context"
	"errors"
	"strconv"
)

const (
	mysteryFormulaFirstTemplateID = 5042
	mysteryFormulaLastTemplateID  = 5107
	mysteryFormulaRecipeOffset    = 5041
)

var (
	ErrFormulaNotMysteryRecipe = errors.New("mysterytreasure: formula is not a mystery recipe")
	ErrRecipeAlreadyLearned    = errors.New("mysterytreasure: recipe already learned")
)

func (s *Service) LearnRecipeByFormula(ctx context.Context, characterID int64, itemTemplateID int) (int, error) {
	recipeID, ok := mysteryRecipeIDForFormula(itemTemplateID)
	if !ok {
		return 0, ErrFormulaNotMysteryRecipe
	}
	if s == nil || s.gameData == nil || s.gameData.MysGetRecipe(recipeID) == nil {
		return 0, ErrNoRecipe
	}

	state, err := s.Load(ctx, characterID)
	if err != nil {
		return 0, err
	}
	key := strconv.Itoa(recipeID)
	if _, exists := state.LearnedRec[key]; exists {
		return recipeID, ErrRecipeAlreadyLearned
	}
	state.LearnedRec[key] = recipeID
	if err := s.Save(ctx, characterID, state); err != nil {
		return 0, err
	}
	return recipeID, nil
}

func (s *Service) ForgetRecipe(ctx context.Context, characterID int64, recipeID int) error {
	if recipeID <= 0 {
		return nil
	}
	state, err := s.Load(ctx, characterID)
	if err != nil {
		return err
	}
	delete(state.LearnedRec, strconv.Itoa(recipeID))
	return s.Save(ctx, characterID, state)
}

func mysteryRecipeIDForFormula(itemTemplateID int) (int, bool) {
	if itemTemplateID < mysteryFormulaFirstTemplateID || itemTemplateID > mysteryFormulaLastTemplateID {
		return 0, false
	}
	return itemTemplateID - mysteryFormulaRecipeOffset, true
}
