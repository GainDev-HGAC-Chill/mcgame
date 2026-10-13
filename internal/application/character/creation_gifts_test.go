package character

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
	domainchar "mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
)

type defaultsTestProvider struct {
	value domainchar.NewCharacterDefaults
	err   error
	calls int
}

func (p *defaultsTestProvider) NewCharacterDefaults(context.Context) (domainchar.NewCharacterDefaults, error) {
	p.calls++
	return p.value, p.err
}

type packageTestWriter struct {
	repo   *stubCharacterRepo
	bundle *domainchar.CreationPackage
	calls  int
	err    error
}

func (w *packageTestWriter) CreateWithPackage(ctx context.Context, char *domainchar.Character, bundle *domainchar.CreationPackage) error {
	w.calls++
	if w.err != nil {
		return w.err
	}
	w.bundle = bundle
	return w.repo.Create(ctx, char)
}

func creationTestService(t *testing.T) (*Service, *defaultsTestProvider, *packageTestWriter) {
	t.Helper()
	repo := &stubCharacterRepo{}
	writer := &packageTestWriter{repo: repo}
	provider := &defaultsTestProvider{value: domainchar.DefaultNewCharacterDefaults()}
	s := NewService(repo, zap.NewNop())
	s.SetCreationDefaults(provider, writer)
	s.SetGameDataManager(mustNewGameDataManagerWithTables(t, map[string][]string{
		"TBL_CLASS": {`{"id":1,"att_strength":15,"att_agility":10,"att_stamina":15,"att_intelligence":5,"att_energy":5}`},
		"TBL_ITEM_TEMPLATE": {
			`{"id":1,"kind":5,"type":501,"stack_max":10}`,
			`{"id":2,"kind":6,"type":604,"stack_max":99,"bind_type":1}`,
			`{"id":3,"kind":5,"type":550,"stack_max":1}`,
			`{"id":4,"kind":5,"type":507,"stack_max":1}`,
			`{"id":5,"kind":5,"type":501,"stack_max":10,"single_flag":1}`,
		},
		"TBL_EQUIPT_TEMPLATE": {`{"id":1,"kind":1,"endure_max":50,"color_code":3,"main_prop1":20,"main_prop_num1":25}`, `{"id":2,"kind":9,"endure_max":20}`},
		"TBL_CREATURE":        {`{"id":91,"name":"Test Pet","class_id":2,"grow_base":0.8,"apt_strength":10,"apt_agility":10,"apt_stamina":10,"apt_intelligence":10,"apt_energy":10}`, `{"id":92,"class_id":1}`, `{"id":93,"class_id":2,"is_bind":1}`},
		"TBL_CREATURE_SKILL":  {`{"id":1,"cid":91,"sid":101}`},
	}))
	return s, provider, writer
}

func TestConfiguredCreationGiftsAndNoReplay(t *testing.T) {
	s, p, w := creationTestService(t)
	p.value.Level, p.value.Gold, p.value.GoldBind = 12, 15, 20
	p.value.Items = []domainchar.StarterItem{
		{TemplateTableID: 29, TemplateID: 1, Quantity: 11},
		{TemplateTableID: 29, TemplateID: 1, Quantity: 3},
		{TemplateTableID: 19, TemplateID: 1, Quantity: 1, Bound: true, Color: 4},
		{TemplateTableID: 29, TemplateID: 2, Quantity: 12, Bound: true, Color: 3},
		{TemplateTableID: 29, TemplateID: 3, Quantity: 1},
		{TemplateTableID: 29, TemplateID: 4, Quantity: 1},
	}
	p.value.Pets = []domainchar.StarterPet{{TemplateID: 91, Quantity: 2, Bound: true, Growth: 1.9}}
	char, err := s.Create(context.Background(), &CreateRequest{AccountID: uuid.New(), Name: "configured", ClassID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if char.Level != 12 || char.Gold != 15 || char.GoldBind != 20 || char.AttrPoints != 55 {
		t.Fatal("wrong scalar defaults", char)
	}
	if len(w.bundle.Items) != 6 || len(w.bundle.Pets) != 2 {
		t.Fatalf("package sizes: items%d pets%d", len(w.bundle.Items), len(w.bundle.Pets))
	}
	if w.bundle.Items[0].StackCount != 10 || w.bundle.Items[1].StackCount != 4 || w.bundle.Items[2].ItemType != item.ItemTypeEquipment || w.bundle.Items[2].Properties["mainPropNum1"] != float64(25) || *w.bundle.Items[2].Durability != 50 {
		t.Fatal("stack/equipment initialization mismatch")
	}
	if w.bundle.Items[4].SlotType != item.SlotTypeQuestBag || w.bundle.Items[5].SlotType != item.SlotTypePetItemBag {
		t.Fatal("special bag routing mismatch")
	}
	pet := w.bundle.Pets[0]
	if pet.GrowRate != 1.9 || !pet.IsBound() || pet.Level != 1 || pet.Property["skill1"] != 101 || pet.Property["skill6"] != -1 || pet.CurrentHP != pet.MaxHP {
		t.Fatalf("pet initialization mismatch: %+v", pet)
	}
	hash := w.bundle.Hash
	p.value.Items[0].Quantity = 100
	p.value.Pets[0].Growth = 2.5
	if w.bundle.Defaults.Items[0].Quantity != 11 || w.bundle.Defaults.Pets[0].Growth != 1.9 {
		t.Fatal("snapshot aliases editable config")
	}
	newHash, _ := w.bundle.Defaults.SnapshotHash()
	if newHash != hash {
		t.Fatal("snapshot changed")
	}
	char.Gold = 0
	for i := 0; i < 2; i++ {
		if _, err := s.Select(context.Background(), &SelectRequest{AccountID: char.AccountID, CharacterID: char.ID}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.GetByID(context.Background(), char.ID); err != nil {
			t.Fatal(err)
		}
	}
	if char.Gold != 0 || w.calls != 1 || p.calls != 1 {
		t.Fatal("selection/load replayed creation package")
	}
	p.value = domainchar.DefaultNewCharacterDefaults()
	p.value.Level = 3
	second, err := s.Create(context.Background(), &CreateRequest{AccountID: char.AccountID, Name: "sibling", ClassID: 1})
	if err != nil || second.Level != 3 || w.calls != 2 || p.calls != 2 {
		t.Fatal("config refresh not applied to future sibling", second, err)
	}
}

func TestCreationPackageRejectsBeforeAnyPersistence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*defaultsTestProvider)
	}{
		{"config read", func(p *defaultsTestProvider) { p.err = errors.New("unavailable") }},
		{"bad scalar", func(p *defaultsTestProvider) { p.value.Level = 0 }},
		{"unknown item", func(p *defaultsTestProvider) {
			p.value.Items = []domainchar.StarterItem{{TemplateTableID: 29, TemplateID: 999, Quantity: 1}}
		}},
		{"bound item", func(p *defaultsTestProvider) {
			p.value.Items = []domainchar.StarterItem{{TemplateTableID: 29, TemplateID: 2, Quantity: 1}}
		}},
		{"material color", func(p *defaultsTestProvider) {
			p.value.Items = []domainchar.StarterItem{{TemplateTableID: 29, TemplateID: 2, Quantity: 1, Bound: true, Color: 5}}
		}},
		{"consumable color", func(p *defaultsTestProvider) {
			p.value.Items = []domainchar.StarterItem{{TemplateTableID: 29, TemplateID: 1, Quantity: 1, Color: 1}}
		}},
		{"regular overflow", func(p *defaultsTestProvider) {
			p.value.Items = []domainchar.StarterItem{{TemplateTableID: 29, TemplateID: 1, Quantity: 301}}
		}},
		{"single flag overflow", func(p *defaultsTestProvider) {
			p.value.Items = []domainchar.StarterItem{{TemplateTableID: 29, TemplateID: 5, Quantity: 31}}
		}},
		{"quest overflow", func(p *defaultsTestProvider) {
			p.value.Items = []domainchar.StarterItem{{TemplateTableID: 29, TemplateID: 3, Quantity: 31}}
		}},
		{"pet item overflow", func(p *defaultsTestProvider) {
			p.value.Items = []domainchar.StarterItem{{TemplateTableID: 29, TemplateID: 4, Quantity: 31}}
		}},
		{"unknown pet", func(p *defaultsTestProvider) {
			p.value.Pets = []domainchar.StarterPet{{TemplateID: 999, Quantity: 1, Growth: 1.9}}
		}},
		{"non pet creature", func(p *defaultsTestProvider) {
			p.value.Pets = []domainchar.StarterPet{{TemplateID: 92, Quantity: 1, Growth: 1.9}}
		}},
		{"bound pet", func(p *defaultsTestProvider) {
			p.value.Pets = []domainchar.StarterPet{{TemplateID: 93, Quantity: 1, Growth: 1.9}}
		}},
		{"pet overflow", func(p *defaultsTestProvider) {
			p.value.Pets = []domainchar.StarterPet{{TemplateID: 91, Quantity: 6, Growth: 1.9}, {TemplateID: 91, Quantity: 1, Growth: 1.9}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, p, w := creationTestService(t)
			tc.mutate(p)
			char, err := s.Create(context.Background(), &CreateRequest{AccountID: uuid.New(), Name: "invalid", ClassID: 1})
			if err == nil || char != nil || w.calls != 0 || w.repo.created != nil {
				t.Fatal("invalid package reached persistence", char, err, w.calls)
			}
		})
	}
}

func TestConfiguredCreationRequiresAtomicWriterAndCanRetry(t *testing.T) {
	s, p, w := creationTestService(t)
	s.SetCreationDefaults(p, nil)
	if _, err := s.Create(context.Background(), &CreateRequest{AccountID: uuid.New(), Name: "no-writer", ClassID: 1}); err == nil || w.repo.created != nil {
		t.Fatal("fell back to non-atomic creation")
	}
	s.SetCreationDefaults(p, w)
	w.err = errors.New("atomic rollback")
	req := &CreateRequest{AccountID: uuid.New(), Name: "retry", ClassID: 1}
	if char, err := s.Create(context.Background(), req); err == nil || char != nil || w.repo.created != nil {
		t.Fatal("failed transaction succeeded")
	}
	w.err = nil
	if _, err := s.Create(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	w.repo.exists = true
	if _, err := s.Create(context.Background(), req); err == nil || w.calls != 2 {
		t.Fatal("committed retry replayed gifts")
	}
}

func TestCreationPackageAcceptsExactSpecialBagAndPetCapacities(t *testing.T) {
	s, provider, writer := creationTestService(t)
	provider.value.Items = []domainchar.StarterItem{
		{TemplateTableID: 29, TemplateID: 1, Quantity: 300},
		{TemplateTableID: 29, TemplateID: 3, Quantity: 30},
		{TemplateTableID: 29, TemplateID: 4, Quantity: 30},
	}
	provider.value.Pets = []domainchar.StarterPet{{TemplateID: 91, Quantity: 6, Growth: 1.9}}
	if _, err := s.Create(context.Background(), &CreateRequest{AccountID: uuid.New(), Name: "exact-boundary", ClassID: 1}); err != nil {
		t.Fatal(err)
	}
	if len(writer.bundle.Items) != 90 || len(writer.bundle.Pets) != 6 {
		t.Fatal("wrong boundary capacities")
	}
	counts := map[item.SlotType]int{}
	for _, it := range writer.bundle.Items {
		if it.SlotIndex != counts[it.SlotType] {
			t.Fatal("duplicate/non-sequential starter slot")
		}
		counts[it.SlotType]++
	}
	for _, count := range counts {
		if count != 30 {
			t.Fatal("incorrect bag capacity", counts)
		}
	}
}
