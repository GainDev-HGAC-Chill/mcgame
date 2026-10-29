package npc

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	appgathering "mcgame-server/internal/application/gathering"
	domaingathering "mcgame-server/internal/domain/gathering"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type gatheringHandlerTransport struct {
	mu      sync.Mutex
	methods []string
}

func (t *gatheringHandlerTransport) WriteCommand(_ context.Context, method string, _ []interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.methods = append(t.methods, method)
	return nil
}

func (t *gatheringHandlerTransport) WriteCommandSync(ctx context.Context, method string, args []interface{}) error {
	return t.WriteCommand(ctx, method, args)
}

func (t *gatheringHandlerTransport) WriteResult(context.Context, int64, interface{}) error {
	return nil
}

func (t *gatheringHandlerTransport) WriteError(context.Context, int64, error) error {
	return nil
}

func (t *gatheringHandlerTransport) Kind() string {
	return "test"
}

func (t *gatheringHandlerTransport) Close() error {
	return nil
}

func (t *gatheringHandlerTransport) snapshot() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.methods...)
}

type gatheringHandlerCatalog struct {
	npc       *models.NpcTemplate
	equipment *models.EquiptTemplateTemplate
}

func (c gatheringHandlerCatalog) GetItem(int) *models.ItemTemplateTemplate {
	return nil
}

func (c gatheringHandlerCatalog) GetNPC(id int) *models.NpcTemplate {
	if c.npc != nil && c.npc.GetID() == id {
		return c.npc
	}
	return nil
}

func (c gatheringHandlerCatalog) GetEquipment(id int) *models.EquiptTemplateTemplate {
	if c.equipment != nil && int(c.equipment.ID) == id {
		return c.equipment
	}
	return nil
}

type gatheringHandlerSkillReader struct{}

func (gatheringHandlerSkillReader) GetLifeSkillLevel(context.Context, int64, int) (int, error) {
	return 1, nil
}

type gatheringHandlerItemReader struct {
	item *domainitem.Item
}

func (r gatheringHandlerItemReader) GetItemInSlot(context.Context, int64, domainitem.SlotType, int) (*domainitem.Item, error) {
	return r.item, nil
}

type gatheringHandlerRandom struct{}

func (gatheringHandlerRandom) IntN(int) int {
	return 0
}

func (gatheringHandlerRandom) Float64() float64 {
	return 0
}

type gatheringHandlerCommandRepo struct {
	mu    sync.Mutex
	calls int
}

func (r *gatheringHandlerCommandRepo) CompleteGathering(context.Context, domaingathering.CompletionCommand) (*domaingathering.CompletionResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return &domaingathering.CompletionResult{Succeeded: true}, nil
}

func (r *gatheringHandlerCommandRepo) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func TestCancelFishingBeforeTimerPerformsZeroCommands(t *testing.T) {
	durability := 10
	npcTemplate := &models.NpcTemplate{ID: 1189, Type: 21, Lv: 1}
	catalog := gatheringHandlerCatalog{
		npc:       npcTemplate,
		equipment: &models.EquiptTemplateTemplate{ID: 1437, Type: 1201, ProTime: 0.05, SuccRate: 100},
	}
	repository := &gatheringHandlerCommandRepo{}
	service := appgathering.NewService(repository, nil, gatheringHandlerSkillReader{}, gatheringHandlerItemReader{item: &domainitem.Item{
		ID: 91, TemplateID: 1437, SlotType: domainitem.SlotTypeEquipped, SlotIndex: 13, Durability: &durability,
	}}, catalog)
	service.SetRandomSource(gatheringHandlerRandom{})

	handler := NewHandler(nil, nil, nil, nil, nil, zap.NewNop())
	handler.SetGatheringService(service)
	transport := &gatheringHandlerTransport{}
	connection := rtmp.NewConnection(7, nil, nil, zap.NewNop())
	connection.SetTransport(transport)
	ctx := &rtmp.RPCContext{Context: context.Background(), ConnID: 7, CharacterID: "3", Connection: connection}

	if err := handler.startFishing(ctx, 3, npcTemplate); err != nil {
		t.Fatalf("startFishing() error = %v", err)
	}
	if _, err := handler.QuitGatheringClinet(ctx, nil); err != nil {
		t.Fatalf("QuitGatheringClinet() error = %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	if repository.callCount() != 0 {
		t.Fatalf("cancelled fishing executed %d completion commands", repository.callCount())
	}
	want := []string{"onSetCharState", "onGathering", "quitGathering", "onSetCharState"}
	got := transport.snapshot()
	if len(got) != len(want) {
		t.Fatalf("callback methods = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("callback methods = %v, want %v", got, want)
		}
	}
}

func TestFinishGatheringCommandErrorClosesAndResetsWithoutReward(t *testing.T) {
	handler := NewHandler(nil, nil, nil, nil, nil, zap.NewNop())
	transport := &gatheringHandlerTransport{}
	connection := rtmp.NewConnection(8, nil, nil, zap.NewNop())
	connection.SetTransport(transport)
	ctx := &rtmp.RPCContext{Context: context.Background(), ConnID: 8, CharacterID: "3", Connection: connection}

	handler.finishGathering(ctx, &domaingathering.CompletionResult{
		Succeeded: true,
		Reward:    &domainitem.Item{ID: 1, CharacterID: 3, TemplateID: 2361, SlotType: domainitem.SlotTypeTempBag},
	}, errors.New("command failed"))

	want := []string{"quitGathering", "onSetCharState"}
	got := transport.snapshot()
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("callback methods = %v, want %v", got, want)
	}
}
