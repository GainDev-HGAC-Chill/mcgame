// Open-sourced by BaoLT

package mysterytreasure

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestLearnRecipeByFormulaPersistsCanonicalRecipeID(t *testing.T) {
	repo := newFakeRepo()
	service := NewService(repo, zap.NewNop())
	service.SetGameData(&fakeGameData{recipes: map[int]*models.MystreRecipeTemplate{
		1:  {ID: 1},
		66: {ID: 66},
	}})

	for _, test := range []struct {
		itemTemplateID int
		recipeID       int
	}{
		{itemTemplateID: 5042, recipeID: 1},
		{itemTemplateID: 5107, recipeID: 66},
	} {
		t.Run(strconv.Itoa(test.itemTemplateID), func(t *testing.T) {
			got, err := service.LearnRecipeByFormula(context.Background(), 7, test.itemTemplateID)
			if err != nil {
				t.Fatalf("LearnRecipeByFormula() error = %v", err)
			}
			if got != test.recipeID {
				t.Fatalf("recipe ID = %d, want %d", got, test.recipeID)
			}
		})
	}

	state, err := service.Load(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if state.LearnedRec["1"] != 1 || state.LearnedRec["66"] != 66 {
		t.Fatalf("learned recipes = %#v", state.LearnedRec)
	}
}

func TestLearnRecipeByFormulaRejectsDuplicateAndUnrelatedFormula(t *testing.T) {
	service := NewService(newFakeRepo(), zap.NewNop())
	service.SetGameData(&fakeGameData{recipes: map[int]*models.MystreRecipeTemplate{6: {ID: 6}}})

	if _, err := service.LearnRecipeByFormula(context.Background(), 9, 5047); err != nil {
		t.Fatal(err)
	}
	if _, err := service.LearnRecipeByFormula(context.Background(), 9, 5047); !errors.Is(err, ErrRecipeAlreadyLearned) {
		t.Fatalf("duplicate error = %v", err)
	}
	if _, err := service.LearnRecipeByFormula(context.Background(), 9, 2413); !errors.Is(err, ErrFormulaNotMysteryRecipe) {
		t.Fatalf("unrelated formula error = %v", err)
	}
}

func TestForgetRecipeSupportsItemConsumptionRollback(t *testing.T) {
	service := NewService(newFakeRepo(), zap.NewNop())
	service.SetGameData(&fakeGameData{recipes: map[int]*models.MystreRecipeTemplate{11: {ID: 11}}})
	ctx := context.Background()

	recipeID, err := service.LearnRecipeByFormula(ctx, 4, 5052)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ForgetRecipe(ctx, 4, recipeID); err != nil {
		t.Fatal(err)
	}
	state, err := service.Load(ctx, 4)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := state.LearnedRec["11"]; exists {
		t.Fatalf("rollback left recipe in state: %#v", state.LearnedRec)
	}
}
