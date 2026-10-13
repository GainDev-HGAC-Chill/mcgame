package auth

import (
	"context"
	"testing"

	appchar "mcgame-server/internal/application/character"
	domainchar "mcgame-server/internal/domain/character"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type starterCharacterRepo struct {
	authTestCharacterRepo
}

func (r *starterCharacterRepo) Create(_ context.Context, char *domainchar.Character) error {
	char.ID = 101
	snapshot := *char
	r.char = &snapshot
	return nil
}

func (r *starterCharacterRepo) FindByAccountID(_ context.Context, accountID uuid.UUID) ([]*domainchar.Character, error) {
	if r.char == nil || r.char.AccountID != accountID {
		return nil, nil
	}
	snapshot := *r.char
	return []*domainchar.Character{&snapshot}, nil
}

func TestNewCharReturnsStarterPackageAndLevel90CharacterList(t *testing.T) {
	repo := &starterCharacterRepo{}
	svc := appchar.NewService(repo, zap.NewNop())
	h := NewHandler(nil, svc, nil, nil, nil, nil, nil, zap.NewNop())
	ctx, sender := characterListContext("scene/")
	ctx.AccountID = uuid.New().String()
	result, err := h.NewChar(ctx, []interface{}{map[string]interface{}{
		"name": "new-starter", "classId": float64(1), "gender": float64(0),
	}})
	if err != nil {
		t.Fatal(err)
	}
	dto, ok := result.(map[string]interface{})
	if !ok || dto["level"] != 90 || dto["gold"] != int64(999999) || dto["goldBind"] != int64(999999) {
		t.Fatalf("newChar starter response: %#v", result)
	}
	if repo.char == nil || repo.char.Level != 90 || repo.char.Gold != 999999 || repo.char.GoldBind != 999999 {
		t.Fatal("starter package not saved with character")
	}
	if len(sender.methods) != 1 || sender.methods[0] != "onIcl" || len(sender.args[0]) != 1 {
		t.Fatalf("newChar list callback: %v %#v", sender.methods, sender.args)
	}
	list, ok := sender.args[0][0].([]map[string]interface{})
	if !ok || len(list) != 1 || list[0]["level"] != 90 || list[0]["exp"] != repo.char.CumulativeExpCapped() {
		t.Fatalf("newChar list must expose level-90 cumulative exp: %#v", sender.args)
	}
	if list[0]["exp"].(int64) <= 0 {
		t.Fatal("level-90 cumulative exp must not be zero even though within-level exp is zero")
	}
}
