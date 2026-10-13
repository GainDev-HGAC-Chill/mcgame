package auth

import (
	"context"
	"reflect"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	domainwsp "mcgame-server/internal/domain/warsprite"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type callbackLoginStateProvider struct {
	state map[string]interface{}
}

func (p *callbackLoginStateProvider) BuildCharacterLoginState(context.Context, int64) (map[string]interface{}, error) {
	return p.state, nil
}

func TestChooseCharacterCallbackSequenceDoesNotInitializeLazyPanels(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, nil, nil, zap.NewNop())
	state := map[string]interface{}{
		"warSprite":    domainwsp.Default().ToClientPayload(),
		"jewelOffRate": 7,
	}
	h.loginState = &callbackLoginStateProvider{state: state}
	char := domainchar.NewCharacter(uuid.New(), "Gain", 1, 0)
	char.ID = 1
	char.WarSprite = 229
	char.BattleSprite = 230
	for i := 0; i < 2; i++ {
		ctx, sender := characterListContext("scene/")
		h.sendOnChooseCharactorCallback(ctx, char)
		want := []string{"onReadCharData", "onChooseCharactor", "updateUserBar", "updateJewelOffPrice", "allServerMailsWarning", "initMountTimer", "initDecoShowTimer", "initPRSShowTimer"}
		if !reflect.DeepEqual(sender.methods, want) {
			t.Fatalf("callback sequence = %v, want %v", sender.methods, want)
		}
		for j, args := range sender.args {
			if len(args) != 1 || args[0] == nil {
				t.Fatalf("%s arguments = %#v", sender.methods[j], args)
			}
		}
		payload := sender.args[1][0].(map[string]interface{})
		if _, exists := payload["warSpritePanel"]; exists {
			t.Fatal("login must not carry a lazy-panel push payload")
		}
		data := payload["cData"].(map[string]interface{})
		if data["warSprite"] != char.WarSprite || data["battleSprite"] != char.BattleSprite {
			t.Fatalf("sprite currency scalars overwritten: %#v %#v", data["warSprite"], data["battleSprite"])
		}
		if sender.args[3][0] != 7 {
			t.Fatalf("jewel discount lost: %#v", sender.args[3])
		}
	}
	if _, exists := state["warSprite"]; !exists {
		t.Fatal("login mapping mutated provider state")
	}
}

func TestLoginStateCallbacksIgnoreLegacyWarSpritePanel(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, nil, nil, zap.NewNop())
	ctx, sender := characterListContext("scene/")
	h.sendLoginStateCallbacks(ctx, map[string]interface{}{"warSpritePanel": domainwsp.Default().ToClientPayload()})
	if !reflect.DeepEqual(sender.methods, []string{"updateJewelOffPrice", "allServerMailsWarning", "initMountTimer"}) {
		t.Fatalf("unsafe login callbacks: %v", sender.methods)
	}
}
