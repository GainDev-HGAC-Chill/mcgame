package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	"mcgame-server/internal/domain/statfeature"
)

type packageTestTransaction struct {
	queries       int
	failAt        int
	commitErr     error
	committed     bool
	rolledBack    bool
	durableWrites int
	marker        []byte
}
type packageTestRow struct {
	tx    *packageTestTransaction
	sql   string
	args  []any
	index int
}

func (tx *packageTestTransaction) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	tx.queries++
	return packageTestRow{tx, sql, args, tx.queries}
}
func (r packageTestRow) Scan(dest ...any) error {
	if r.index == r.tx.failAt {
		return errors.New("injected statement failure")
	}
	if strings.Contains(r.sql, "upsert_character_feature_state") {
		if r.args[0] != int64(77) || r.args[1] != character.CreationPackageFeatureKey {
			panic("wrong marker ownership")
		}
		r.tx.marker = append([]byte{}, r.args[2].([]byte)...)
		*dest[0].(*bool) = true
	} else {
		if !strings.Contains(r.sql, "create_character_full") && r.args[0] != int64(77) {
			panic("wrong gift ownership")
		}
		*dest[0].(*int64) = 77
	}
	return nil
}
func (tx *packageTestTransaction) Commit(context.Context) error {
	if tx.commitErr != nil {
		return tx.commitErr
	}
	tx.committed = true
	tx.durableWrites = tx.queries
	return nil
}
func (tx *packageTestTransaction) Rollback(context.Context) error {
	if !tx.committed {
		tx.rolledBack = true
		tx.durableWrites = 0
	}
	return nil
}

func creationPersistenceFixture() (*character.Character, *character.CreationPackage) {
	d := character.DefaultNewCharacterDefaults()
	d.Items = []character.StarterItem{{TemplateTableID: 29, TemplateID: 1, Quantity: 1}}
	d.Pets = []character.StarterPet{{TemplateID: 91, Quantity: 1, Growth: 1.9}}
	hash, _ := d.SnapshotHash()
	char := character.NewCharacter(uuid.New(), "test-only-mock", 1, 0)
	char.RaiseLevelTo(d.Level)
	char.Gold, char.GoldBind = d.Gold, d.GoldBind
	bundle := &character.CreationPackage{Defaults: d, Hash: hash, Items: []*item.Item{item.NewItem(0, 1, item.ItemTypeConsumable, item.SlotTypeBag, 0)}, Pets: []*pet.Pet{{TemplateID: 91, GrowRate: 1.9, Property: map[string]interface{}{}}}}
	return char, bundle
}

func TestCreationPackageAtomicRollbackAndRetry(t *testing.T) {
	for _, failAt := range []int{1, 2, 3, 4, 5} {
		t.Run(string(rune('0'+failAt)), func(t *testing.T) {
			char, bundle := creationPersistenceFixture()
			tx := &packageTestTransaction{failAt: failAt}
			if failAt == 5 {
				tx.commitErr = errors.New("commit rejected")
			}
			begin := func(context.Context) (creationTransaction, error) { return tx, nil }
			if err := persistCreationPackage(context.Background(), char, bundle, begin); err == nil {
				t.Fatal("failure reported success")
			}
			if tx.committed || !tx.rolledBack || tx.durableWrites != 0 || char.ID != 0 || bundle.Items[0].ID != 0 || bundle.Pets[0].ID != 0 || bundle.Items[0].CharacterID != 0 {
				t.Fatal("failed transaction published partial creation")
			}
			tx = &packageTestTransaction{}
			if err := persistCreationPackage(context.Background(), char, bundle, begin); err != nil {
				t.Fatal("retry failed", err)
			}
			if !tx.committed || tx.durableWrites != 4 || char.ID != 77 || len(tx.marker) == 0 {
				t.Fatal("character, item, pet and marker not committed together")
			}
			queries := tx.queries
			if err := persistCreationPackage(context.Background(), char, bundle, begin); err == nil || tx.queries != queries {
				t.Fatal("replayed committed character gifts")
			}
		})
	}
}

func TestCreationSnapshotMutationRejectedBeforeTransaction(t *testing.T) {
	char, bundle := creationPersistenceFixture()
	bundle.Defaults.Gold++
	called := false
	err := persistCreationPackage(context.Background(), char, bundle, func(context.Context) (creationTransaction, error) {
		called = true
		return nil, errors.New("must not begin")
	})
	if err == nil || called {
		t.Fatal("mutated defaults reached transaction")
	}
}

func TestGenericFeatureStateCannotOverwriteCreationSnapshot(t *testing.T) {
	repo := &StatFeatureRepository{}
	err := repo.UpsertCharacterFeatureState(context.Background(), &statfeature.CharacterFeatureState{CharacterID: 77, FeatureKey: character.CreationPackageFeatureKey})
	if err == nil {
		t.Fatal("reserved snapshot key accepted by generic upsert")
	}
}

func TestCreationPackageBeginFailureDoesNotPublishIDs(t *testing.T) {
	char, bundle := creationPersistenceFixture()
	want := errors.New("begin unavailable")
	err := persistCreationPackage(context.Background(), char, bundle, func(context.Context) (creationTransaction, error) { return nil, want })
	if !errors.Is(err, want) || char.ID != 0 || bundle.Items[0].ID != 0 || bundle.Pets[0].ID != 0 {
		t.Fatal("begin failure changed package", err)
	}
}
