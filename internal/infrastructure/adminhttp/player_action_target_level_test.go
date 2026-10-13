package adminhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"
)

type forbiddenTargetItems struct{ ItemProvider }

func (forbiddenTargetItems) FindByCharacterAndSlotType(context.Context, int64, item.SlotType) ([]*item.Item, error) {
	panic("target level must use locked cache inventory, not repository reads")
}

type targetTemplates struct {
	ItemTemplateProvider
	required float64
	missing  bool
}

func (p targetTemplates) GetEquipmentTemplate(int) *models.EquiptTemplateTemplate {
	if p.missing {
		return nil
	}
	return &models.EquiptTemplateTemplate{ReqLevel: p.required, Name: "Kiếm"}
}

type targetStats struct{ bonus int }

func (s targetStats) AggregateEquipmentStats(context.Context, int64) character.EquipmentStatBonuses {
	bonuses := character.NewEquipmentStatBonuses()
	bonuses.AddFlat(character.PropAttack, s.bonus)
	return bonuses
}

type targetCache struct {
	mu    sync.Mutex
	char  *character.Character
	items []*item.Item
}

func (c *targetCache) UpdateCachedCharacter(_ int64, fn func(*character.Character)) *character.Character {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.char == nil {
		return nil
	}
	fn(c.char)
	snapshot := *c.char
	return &snapshot
}

func (c *targetCache) UpdateCachedCharacterConditional(_ int64, fn func(*character.Character, []*item.Item) (bool, error)) (*character.Character, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.char == nil {
		return nil, false, nil
	}
	snapshot := *c.char
	changed, err := fn(&snapshot, c.items)
	if changed && err == nil {
		*c.char = snapshot
	}
	result := *c.char
	return &result, changed && err == nil, err
}

type targetSessions struct {
	SessionProvider
	conn *rtmp.Connection
}

func (s targetSessions) GetConnectionByCharacterID(string) *rtmp.Connection { return s.conn }

func targetServer() (*Server, *stubCharacterUpdater) {
	updater := &stubCharacterUpdater{}
	return &Server{logger: zap.NewNop(), characterUpdater: updater, characterCacheUpdater: &targetCache{}, itemProvider: forbiddenTargetItems{}, templateProvider: targetTemplates{}, progressionStats: targetStats{}}, updater
}

func TestParseTargetLevel(t *testing.T) {
	for _, raw := range []string{`{}`, `null`, `{"targetLevel":null}`, `{"targetLevel":0}`, `{"targetLevel":-1}`, `{"targetLevel":99999}`, `{"targetLevel":1.5}`, `{"targetLevel":"5"}`, `{"targetLevel":true}`, `{"targetLevel":5,"level":5}`, `{"targetLevel":5} {}`} {
		if _, err := parseTargetLevel(json.RawMessage(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if n, err := parseTargetLevel(json.RawMessage(`{"targetLevel":5}`)); err != nil || n != 5 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestExecuteTargetLevelCachedAndSame(t *testing.T) {
	s, updater := targetServer()
	char := character.NewCharacter(uuid.New(), "target", 1, 0)
	char.ID = 7
	char.Gold = 12345
	s.characterCacheUpdater = &targetCache{char: char}
	w := httptest.NewRecorder()
	s.executeSetTargetLevel(w, context.Background(), char, json.RawMessage(`{"targetLevel":10}`))
	if w.Code != http.StatusOK || updater.updated != nil || char.Level != 10 || char.Gold != 12345 {
		t.Fatalf("raise failed: %s", w.Body)
	}
	char.Experience = 42
	updater.updated = nil
	w = httptest.NewRecorder()
	s.executeSetTargetLevel(w, context.Background(), char, json.RawMessage(`{"targetLevel":10}`))
	if w.Code != http.StatusOK || updater.updated != nil || char.Experience != 42 {
		t.Fatalf("same-level mutation: %s", w.Body)
	}
}

func TestExecuteTargetLevelEquipmentRejection(t *testing.T) {
	for _, missing := range []bool{false, true} {
		s, updater := targetServer()
		s.templateProvider = targetTemplates{required: 8, missing: missing}
		char := character.NewCharacter(uuid.New(), "target", 1, 0)
		char.RaiseLevelTo(10)
		s.characterCacheUpdater = &targetCache{char: char, items: []*item.Item{{ID: 23, TemplateID: 9, SlotType: item.SlotTypeEquipped}}}
		before := *char
		w := httptest.NewRecorder()
		s.executeSetTargetLevel(w, context.Background(), char, json.RawMessage(`{"targetLevel":5}`))
		if w.Code != http.StatusBadRequest || updater.updated != nil || !reflect.DeepEqual(before, *char) {
			t.Fatalf("unsafe downgrade: %s", w.Body)
		}
	}
}

func TestExecuteTargetLevelUsesCacheEvenOfflineAndFreshLevel(t *testing.T) {
	s, updater := targetServer()
	stale := character.NewCharacter(uuid.New(), "target", 1, 0)
	stale.RaiseLevelTo(10)
	fresh := *stale
	fresh.RaiseLevelTo(20)
	fresh.Gold = 999
	fresh.AllocateAttribute("stamina")
	s.characterCacheUpdater = &targetCache{char: &fresh}
	w := httptest.NewRecorder()
	s.executeSetTargetLevel(w, context.Background(), stale, json.RawMessage(`{"targetLevel":15}`))
	if w.Code != http.StatusOK || updater.updated != nil || fresh.Level != 15 || fresh.DistributedAttrPoints != 0 || fresh.AttrPoints != 70 || fresh.Gold != 999 || fresh.Strength != 24 {
		t.Fatalf("stale state overwritten: %s %+v", w.Body, fresh)
	}
}

func TestExecuteTargetLevelFailures(t *testing.T) {
	for _, mode := range []string{"uncached", "unconfigured", "transition"} {
		s, updater := targetServer()
		char := character.NewCharacter(uuid.New(), "target", 1, 0)
		char.RaiseLevelTo(10)
		switch mode {
		case "uncached":
			s.characterCacheUpdater = &targetCache{}
		case "unconfigured":
			s.characterCacheUpdater = nil
		case "transition":
			char.AllocateAttribute("strength")
			char.AptStrength += 1000
			s.characterCacheUpdater = &targetCache{char: char}
		}
		before := *char
		w := httptest.NewRecorder()
		s.executeSetTargetLevel(w, context.Background(), char, json.RawMessage(`{"targetLevel":5}`))
		if w.Code == http.StatusOK || updater.updated != nil || !reflect.DeepEqual(before, *char) {
			t.Fatalf("mode=%s mutated despite failure: %s", mode, w.Body)
		}
	}
}

func TestLegacyLevelSettersRejected(t *testing.T) {
	s, updater := targetServer()
	char := character.NewCharacter(uuid.New(), "target", 1, 0)
	for _, execute := range []func(http.ResponseWriter, context.Context, *character.Character, json.RawMessage){s.executeSetAttributes, s.executeSetProgression} {
		w := httptest.NewRecorder()
		execute(w, context.Background(), char, json.RawMessage(`{"level":2}`))
		if w.Code != http.StatusBadRequest || updater.updated != nil || char.Level != 1 {
			t.Fatalf("legacy level write accepted: %s", w.Body)
		}
	}
}

type targetTransport struct {
	rtmp.Transport
	method string
	args   []interface{}
	err    error
}

func (t *targetTransport) WriteCommand(_ context.Context, method string, args []interface{}) error {
	t.method, t.args = method, args
	return t.err
}

func TestExecuteTargetLevelOnlineCallbackAndEquipmentBonuses(t *testing.T) {
	for _, failCallback := range []bool{false, true} {
		s, updater := targetServer()
		char := character.NewCharacter(uuid.New(), "target", 1, 0)
		char.RaiseLevelTo(10)
		char.AllocateAttribute("strength")
		s.characterCacheUpdater = &targetCache{char: char, items: []*item.Item{{ID: 7, SlotType: item.SlotTypeEquipped}}}
		s.templateProvider = targetTemplates{required: 5}
		s.progressionStats = targetStats{bonus: 200}
		transport := &targetTransport{}
		if failCallback {
			transport.err = errors.New("callback failed")
		}
		conn := rtmp.NewConnection(1, nil, nil, zap.NewNop())
		conn.SetTransport(transport)
		s.sessionProvider = targetSessions{conn: conn}
		w := httptest.NewRecorder()
		s.executeSetTargetLevel(w, context.Background(), char, json.RawMessage(`{"targetLevel":5}`))
		if w.Code != http.StatusOK || updater.updated != nil || char.Level != 5 || char.DistributedAttrPoints != 0 || transport.method != "onUPP" || len(transport.args) != 1 {
			t.Fatalf("online transition failed: %s", w.Body)
		}
		payload := transport.args[0].(map[string]interface{})
		props := payload["property"].(map[string]interface{})
		if props["finalAttack"] != char.Attack+200 || payload["bagSlotNum"] != nil {
			t.Fatalf("bonus/noise mismatch: %v", payload)
		}
		var response adminActionResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		wantDelivery := "live_session"
		if failCallback {
			wantDelivery = "live_session_callback_failed"
		}
		if response.DeliveryMode != wantDelivery {
			t.Fatalf("delivery=%s want=%s", response.DeliveryMode, wantDelivery)
		}
	}
}

type targetCharacterProvider struct{ char *character.Character }

func (p targetCharacterProvider) FindByID(context.Context, int64) (*character.Character, error) {
	return p.char, nil
}

func TestTargetLevelEndpointActionShape(t *testing.T) {
	s, updater := targetServer()
	char := character.NewCharacter(uuid.New(), "target", 1, 0)
	char.ID = 7
	s.characterProvider = targetCharacterProvider{char: char}
	s.characterCacheUpdater = &targetCache{char: char}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/players/action", strings.NewReader(`{"action":"set_target_level","targetId":7,"payload":{"targetLevel":5}}`))
	s.handlePlayerAction(w, req)
	if w.Code != http.StatusOK || updater.updated != nil || char.Level != 5 {
		t.Fatalf("endpoint rejected action: %s", w.Body)
	}
}

func TestTargetLevelStaleSamePrefetchUsesCurrentLevel(t *testing.T) {
	s, updater := targetServer()
	stale := character.NewCharacter(uuid.New(), "target", 1, 0)
	stale.RaiseLevelTo(10)
	fresh := *stale
	fresh.RaiseLevelTo(20)
	s.characterCacheUpdater = &targetCache{char: &fresh}
	w := httptest.NewRecorder()
	s.executeSetTargetLevel(w, context.Background(), stale, json.RawMessage(`{"targetLevel":10}`))
	if w.Code != http.StatusOK || fresh.Level != 10 || fresh.Strength != 19 || updater.updated != nil {
		t.Fatalf("stale same-level decision: %s", w.Body)
	}
}

func TestTargetLevelUncachedOfflineFailsClosedIncludingSame(t *testing.T) {
	for _, target := range []string{`{"targetLevel":5}`, `{"targetLevel":1}`} {
		s, updater := targetServer()
		char := character.NewCharacter(uuid.New(), "target", 1, 0)
		before := *char
		w := httptest.NewRecorder()
		s.executeSetTargetLevel(w, context.Background(), char, json.RawMessage(target))
		if w.Code != http.StatusConflict || updater.updated != nil || !reflect.DeepEqual(before, *char) {
			t.Fatalf("offline state changed: %s", w.Body)
		}
	}
}

func TestTargetLevelCompetingCachedChangesPreserveWealth(t *testing.T) {
	s, updater := targetServer()
	stale := character.NewCharacter(uuid.New(), "target", 1, 0)
	stale.RaiseLevelTo(10)
	fresh := *stale
	fresh.Gold = 987654
	cache := &targetCache{char: &fresh}
	s.characterCacheUpdater = cache
	var wg sync.WaitGroup
	for _, payload := range []string{`{"targetLevel":5}`, `{"targetLevel":15}`, `{"targetLevel":20}`} {
		wg.Add(1)
		go func(raw string) {
			defer wg.Done()
			w := httptest.NewRecorder()
			s.executeSetTargetLevel(w, context.Background(), stale, json.RawMessage(raw))
			if w.Code != http.StatusOK {
				t.Errorf("concurrent mutation rejected: %s", w.Body)
			}
		}(payload)
	}
	wg.Wait()
	if updater.updated != nil || fresh.Gold != 987654 || fresh.Strength != fresh.Level+9 || fresh.AttrPoints != (fresh.Level-1)*character.AttributePointsPerLevel {
		t.Fatalf("competing updates corrupted state: %+v", fresh)
	}
}

func TestTargetLevelOnlineMissingCacheDoesNotOverwriteDatabase(t *testing.T) {
	s, updater := targetServer()
	char := character.NewCharacter(uuid.New(), "target", 1, 0)
	s.sessionProvider = targetSessions{conn: rtmp.NewConnection(1, nil, nil, zap.NewNop())}
	s.characterCacheUpdater = &targetCache{}
	w := httptest.NewRecorder()
	s.executeSetTargetLevel(w, context.Background(), char, json.RawMessage(`{"targetLevel":5}`))
	if w.Code != http.StatusConflict || updater.updated != nil || char.Level != 1 {
		t.Fatalf("online state overwritten: %s", w.Body)
	}
}
