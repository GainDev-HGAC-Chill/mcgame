package prs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	appprs "mcgame-server/internal/application/prs"
	domainchar "mcgame-server/internal/domain/character"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"

	"github.com/yutopp/go-amf0"
	"go.uber.org/zap"
)

type panelRepository struct {
	domainfeature.Repository
	states      []*domainfeature.CharacterFeatureState
	err         error
	characterID int64
	writes      int
}

func (r *panelRepository) ListCharacterFeatureStates(ctx context.Context, id int64) ([]*domainfeature.CharacterFeatureState, error) {
	if ctx == nil {
		return nil, errors.New("nil request context")
	}
	r.characterID = id
	return r.states, r.err
}

func (r *panelRepository) UpsertCharacterFeatureState(context.Context, *domainfeature.CharacterFeatureState) error {
	r.writes++
	return errors.New("panel must be read-only")
}

type panelTransport struct {
	methods []string
	args    [][]interface{}
}

func (s *panelTransport) WriteCommand(_ context.Context, method string, args []interface{}) error {
	s.methods = append(s.methods, method)
	s.args = append(s.args, args)
	return nil
}

type actionFeatureRepository struct {
	domainfeature.Repository
	state *domainfeature.CharacterFeatureState
}

func (r *actionFeatureRepository) ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error) {
	if r.state == nil {
		return nil, nil
	}
	return []*domainfeature.CharacterFeatureState{r.state}, nil
}

func (r *actionFeatureRepository) UpsertCharacterFeatureState(_ context.Context, state *domainfeature.CharacterFeatureState) error {
	r.state = state
	return nil
}

type actionCharacterRepository struct {
	domainchar.Repository
	char *domainchar.Character
}

func (r *actionCharacterRepository) FindByID(context.Context, int64) (*domainchar.Character, error) {
	return r.char, nil
}

func (r *actionCharacterRepository) Update(_ context.Context, char *domainchar.Character) error {
	r.char = char
	return nil
}
func (s *panelTransport) WriteCommandSync(ctx context.Context, method string, args []interface{}) error {
	return s.WriteCommand(ctx, method, args)
}
func (s *panelTransport) WriteResult(context.Context, int64, interface{}) error { return nil }
func (s *panelTransport) WriteError(context.Context, int64, error) error        { return nil }
func (s *panelTransport) Kind() string                                          { return "test" }
func (s *panelTransport) Close() error                                          { return nil }

func assertPanelContract(t *testing.T, result interface{}) map[string]interface{} {
	t.Helper()
	panel, ok := result.(map[string]interface{})
	if !ok || panel == nil {
		t.Fatalf("Responder got null/non-object PRS panel: %#v", result)
	}
	for _, key := range []string{"actArr", "actLimitObj", "chipBag", "excBag"} {
		if value, ok := panel[key].(map[string]interface{}); !ok || value == nil {
			t.Fatalf("PRS panel %s is null/non-object: %#v", key, panel[key])
		}
	}
	for _, key := range []string{"tid", "useSid", "isShow"} {
		if _, ok := panel[key]; !ok {
			t.Fatalf("PRS panel missing %s", key)
		}
	}
	return panel
}

func TestInitPanelResponderContractAndReadOnlySavedState(t *testing.T) {
	repo := &panelRepository{states: []*domainfeature.CharacterFeatureState{{
		FeatureKey: domainfeature.FeaturePRS,
		State: map[string]interface{}{
			"tid": "2", "actArr": map[string]interface{}{"0": float64(1)},
			"actLimitObj": map[string]interface{}{}, "useSid": float64(1),
			"chipBag": map[string]interface{}{}, "excBag": map[string]interface{}{},
		},
	}}}
	h := NewHandler(appprs.NewService(repo, nil, nil, nil), nil)
	sender := &panelTransport{}
	conn := rtmp.NewConnection(1, nil, nil, zap.NewNop())
	conn.SetTransport(sender)
	ctx := &rtmp.RPCContext{CharacterID: "101", Connection: conn}
	result, err := h.InitPanel(ctx, []interface{}{float64(999)})
	if err != nil {
		t.Fatal(err)
	}
	panel := assertPanelContract(t, result)
	if panel["tid"] != "2" || panel["useSid"] != 1 || !reflect.DeepEqual(panel["actArr"], map[string]interface{}{"0": 1}) {
		t.Fatalf("saved PRS state lost: %#v", panel)
	}
	if repo.characterID != 101 || repo.writes != 0 || len(sender.methods) != 0 {
		t.Fatalf("panel must use authenticated ID, read only, and reply through Responder: repo=%#v, pushes=%v", repo, sender.methods)
	}
}

func TestInitPanelFailureAlwaysReturnsCompleteEmptyResponderObject(t *testing.T) {
	for _, test := range []struct {
		name           string
		ctx            *rtmp.RPCContext
		repo           *panelRepository
		missingService bool
	}{
		{name: "nil context", repo: &panelRepository{}},
		{name: "invalid character", ctx: &rtmp.RPCContext{CharacterID: "bad"}, repo: &panelRepository{}},
		{name: "unconfigured service", ctx: &rtmp.RPCContext{CharacterID: "101"}, missingService: true},
		{name: "repository error", ctx: &rtmp.RPCContext{CharacterID: "101"}, repo: &panelRepository{err: errors.New("unavailable")}},
		{name: "no state", ctx: &rtmp.RPCContext{CharacterID: "101"}, repo: &panelRepository{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var service *appprs.Service
			if !test.missingService {
				service = appprs.NewService(test.repo, nil, nil, nil)
			}
			result, err := NewHandler(service, nil).InitPanel(test.ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			panel := assertPanelContract(t, result)
			if !reflect.DeepEqual(panel, emptyPanel()) {
				t.Fatalf("fallback panel = %#v", panel)
			}
		})
	}
}

func TestEmptyPanelAMFResultIsObjectWithNonNullContainers(t *testing.T) {
	var body bytes.Buffer
	encoder := amf0.NewEncoder(&body)
	for _, value := range []interface{}{nil, emptyPanel()} {
		if err := encoder.Encode(value); err != nil {
			t.Fatal(err)
		}
	}
	decoder := amf0.NewDecoder(&body)
	var commandObject, result interface{}
	if err := decoder.Decode(&commandObject); err != nil {
		t.Fatal(err)
	}
	if commandObject != nil {
		t.Fatalf("command object = %#v", commandObject)
	}
	if err := decoder.Decode(&result); err != nil {
		t.Fatal(err)
	}
	assertPanelContract(t, result)
}

func TestAddSoulTreeLvlPushesFullStateForExactNextRow(t *testing.T) {
	mgr := gamedata.NewManager(nil, zap.NewNop())
	raw := []json.RawMessage{
		json.RawMessage(`{"id":1,"cost_stone":100,"next_id":20,"level":1}`),
		json.RawMessage(`{"id":20,"cost_stone":200,"next_id":0,"level":2}`),
	}
	if err := mgr.GetCache().LoadTable(models.TablePrsTree, raw); err != nil {
		t.Fatal(err)
	}
	featureRepo := &actionFeatureRepository{state: &domainfeature.CharacterFeatureState{
		CharacterID: 101,
		FeatureKey:  domainfeature.FeaturePRS,
		State:       map[string]interface{}{"tid": "1"},
	}}
	charRepo := &actionCharacterRepository{char: &domainchar.Character{ID: 101, RealSoulStone: 500}}
	h := NewHandler(appprs.NewService(featureRepo, charRepo, mgr, zap.NewNop()), zap.NewNop())
	sender := &panelTransport{}
	conn := rtmp.NewConnection(1, nil, nil, zap.NewNop())
	conn.SetTransport(sender)
	ctx := &rtmp.RPCContext{Context: context.Background(), CharacterID: "101", Connection: conn}

	if _, err := h.AddSoulTreeLvl(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if len(sender.methods) != 1 || sender.methods[0] != "updatePRSPanel" || len(sender.args[0]) != 1 {
		t.Fatalf("callbacks = %v args=%#v", sender.methods, sender.args)
	}
	panel := assertPanelContract(t, sender.args[0][0])
	if panel["tid"] != "20" || charRepo.char.RealSoulStone != 300 {
		t.Fatalf("callback/cost used wrong next row: panel=%#v stone=%d", panel, charRepo.char.RealSoulStone)
	}
}
