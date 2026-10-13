package prs

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	appprs "mcgame-server/internal/application/prs"
	domainfeature "mcgame-server/internal/domain/statfeature"
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
}

func (s *panelTransport) WriteCommand(_ context.Context, method string, _ []interface{}) error {
	s.methods = append(s.methods, method)
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
