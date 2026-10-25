package dress

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	domainchar "mcgame-server/internal/domain/character"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"
)

type lifecycleFeatureRepo struct {
	state     *domainfeature.CharacterFeatureState
	upsertErr error
}

func (r *lifecycleFeatureRepo) ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error) {
	if r.state == nil {
		return nil, nil
	}
	return []*domainfeature.CharacterFeatureState{r.state}, nil
}

func (r *lifecycleFeatureRepo) UpsertCharacterFeatureState(_ context.Context, state *domainfeature.CharacterFeatureState) error {
	if r.upsertErr != nil {
		return r.upsertErr
	}
	r.state = state
	return nil
}

type lifecycleCharacterRepo struct {
	character *domainchar.Character
	updates   int
}

func (r *lifecycleCharacterRepo) FindByID(context.Context, int64) (*domainchar.Character, error) {
	copy := *r.character
	return &copy, nil
}

func (r *lifecycleCharacterRepo) Update(_ context.Context, character *domainchar.Character) error {
	copy := *character
	r.character = &copy
	r.updates++
	return nil
}

type lifecycleGameData struct {
	holes map[int]*models.DecoHoleTemplate
	runes map[int]*models.DecoRuneTemplate
	shows map[int]*models.DecoShowTemplate
}

func (g *lifecycleGameData) GetDecoHole(id int) *models.DecoHoleTemplate { return g.holes[id] }
func (g *lifecycleGameData) GetDecoRune(id int) *models.DecoRuneTemplate { return g.runes[id] }
func (g *lifecycleGameData) GetDecoShow(id int) *models.DecoShowTemplate { return g.shows[id] }
func (g *lifecycleGameData) GetDecoShowsBySuitID(suitID int) []*models.DecoShowTemplate {
	result := make([]*models.DecoShowTemplate, 0)
	for _, show := range g.shows {
		if int(show.SuitID) == suitID {
			result = append(result, show)
		}
	}
	return result
}

func newLifecycleService(gold int64) (*DecoHoleService, *lifecycleFeatureRepo, *lifecycleCharacterRepo) {
	character := domainchar.NewCharacter(uuid.New(), "deco", 1, 0)
	character.ID = 77
	character.Gold = gold
	features := &lifecycleFeatureRepo{}
	characters := &lifecycleCharacterRepo{character: character}
	data := &lifecycleGameData{
		holes: map[int]*models.DecoHoleTemplate{
			1: {ID: 1, Position: 1, Level: 0, PropType1: 1, PropNum1: 50},
			2: {ID: 2, Position: 2, Level: 0},
			3: {ID: 3, Position: 3, Level: 0},
			4: {ID: 4, Position: 4, Level: 0},
		},
		runes: map[int]*models.DecoRuneTemplate{},
		shows: map[int]*models.DecoShowTemplate{
			101: {ID: 101, Position: 1, SuitID: 9, ActiveGold: 188, T: 1, PropType1: 4, PropNum1: 25},
		},
	}
	return NewDecoHoleService(features, characters, data, zap.NewNop()), features, characters
}

func TestActivateDecoShowPersistsOwnershipAndChargesOnce(t *testing.T) {
	service, _, characters := newLifecycleService(500)
	result, err := service.ActivateDecoShow(context.Background(), 77, 101, 1)
	if err != nil {
		t.Fatalf("ActivateDecoShow() error = %v", err)
	}
	if result.SpentGold != 188 || characters.character.Gold != 312 {
		t.Fatalf("spent/gold = %d/%d, want 188/312", result.SpentGold, characters.character.Gold)
	}
	hole := result.State.Positions[1]
	if hole.Did != 101 || hole.IsShow != 1 || hole.ShowLvl != 1 {
		t.Fatalf("equipped state = %#v", hole)
	}
	if _, err := service.ActivateDecoShow(context.Background(), 77, 101, 1); !errors.Is(err, ErrDecoShowAlreadyOwned) {
		t.Fatalf("duplicate activation error = %v", err)
	}
	if characters.character.Gold != 312 {
		t.Fatalf("duplicate activation changed gold to %d", characters.character.Gold)
	}
}

func TestActivateDecoShowRestoresGoldWhenStateSaveFails(t *testing.T) {
	service, features, characters := newLifecycleService(500)
	features.upsertErr = errors.New("state unavailable")
	if _, err := service.ActivateDecoShow(context.Background(), 77, 101, 1); err == nil {
		t.Fatal("expected save error")
	}
	if characters.character.Gold != 500 || characters.updates != 2 {
		t.Fatalf("gold/updates = %d/%d, want 500/2", characters.character.Gold, characters.updates)
	}
}

func TestDecoShowOwnershipControlsEquipAndSuitActivation(t *testing.T) {
	service, _, _ := newLifecycleService(500)
	if _, err := service.EquipDecoShow(context.Background(), 77, 101, 1); !errors.Is(err, ErrDecoShowNotOwned) {
		t.Fatalf("unowned equip error = %v", err)
	}
	if _, err := service.ActivateDecoShow(context.Background(), 77, 101, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := service.EquipOffDecoShow(context.Background(), 77, 101, 1); err != nil {
		t.Fatal(err)
	}
	result, err := service.GetDecoSuitProp(context.Background(), 77, 9, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !result.ActFlag || result.ActArr["1"] != 101 {
		t.Fatalf("suit activation = %#v", result)
	}
	if len(result.SuitProp) != 1 || result.SuitProp[0]["t"] != 4 || result.SuitProp[0]["propVal"] != 25 {
		t.Fatalf("suitProp array = %#v", result.SuitProp)
	}
}

func TestAggregateDecoBonusesUsesCatalogBackedHoleAndEquippedShow(t *testing.T) {
	service, _, _ := newLifecycleService(500)
	if _, err := service.ActivateDecoShow(context.Background(), 77, 101, 1); err != nil {
		t.Fatal(err)
	}
	bonuses := service.AggregateCharacterStatBonuses(context.Background(), 77)
	if bonuses.Flat[domainchar.PropMaxHP] != 50 || bonuses.Flat[domainchar.PropAttack] != 25 {
		t.Fatalf("bonuses = %#v", bonuses)
	}
}
