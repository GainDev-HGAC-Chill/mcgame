package character

import (
	"context"
	"errors"
	"fmt"
	"testing"

	domainchar "mcgame-server/internal/domain/character"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestCreateStarterRewardsPersistedForEveryNewCharacter(t *testing.T) {
	accountID := uuid.New()
	for classID := 1; classID <= 6; classID++ {
		for gender := 0; gender <= 1; gender++ {
			t.Run(fmt.Sprintf("class%d-gender%d", classID, gender), func(t *testing.T) {
				repo := &stubCharacterRepo{}
				svc := NewService(repo, zap.NewNop())
				svc.SetGameDataManager(mustNewGameDataManagerWithClass(t, fmt.Sprintf(`{"id":%d,"att_strength":15,"att_agility":10,"att_stamina":15,"att_intelligence":5,"att_energy":5,"apt_strength":20,"apt_agility":15,"apt_stamina":25,"apt_intelligence":10,"apt_energy":12}`, classID)))
				created, err := svc.Create(context.Background(), &CreateRequest{
					AccountID: accountID,
					Name:      "starter",
					ClassID:   classID,
					Gender:    gender,
				})
				if err != nil {
					t.Fatal(err)
				}
				if created.Level != 90 || created.Gold != 999999 || created.GoldBind != 999999 {
					t.Fatalf("starter package: level=%d gold=%d goldBind=%d", created.Level, created.Gold, created.GoldBind)
				}
				if repo.created == nil || repo.created.Level != 90 || repo.created.Gold != 999999 || repo.created.GoldBind != 999999 {
					t.Fatal("starter package must be passed to repository.Create, not granted after persistence")
				}
				if created.Experience != 0 || created.AttrPoints != 445 || created.MaxAttrPoints != 445 || created.DistributedAttrPoints != 0 {
					t.Fatalf("progression is inconsistent: exp=%d points=%d max=%d distributed=%d", created.Experience, created.AttrPoints, created.MaxAttrPoints, created.DistributedAttrPoints)
				}
				if created.Strength != 104 || created.Agility != 99 || created.Stamina != 104 || created.Intelligence != 94 || created.Spirit != 94 {
					t.Fatal("starter levels must add normal stat growth after class defaults")
				}
				if created.ClassAptStrength != 20 || created.ClassAptStamina != 25 {
					t.Fatal("class aptitude was lost")
				}
				if created.CurrentHP != created.MaxHP || created.CurrentMP != created.MaxMP || created.MaxHP <= 0 || created.MaxMP <= 0 {
					t.Fatal("new characters must start with full level-90 HP and MP")
				}
				if created.AccountID != accountID || created.ClassID != classID || created.Gender != gender || created.GMLevel != 0 {
					t.Fatal("starter package must preserve identity and normal privileges")
				}
				if created.Money != 0 || created.MoneyBind != 0 || created.BagSlotNum != 1 || created.BankSlotNum != 1 {
					t.Fatal("starter package changed unrelated currency or capacity")
				}
			})
		}
	}
}

func TestStarterRewardsNotRepeatedOnSelectOrLoad(t *testing.T) {
	repo := &stubCharacterRepo{}
	svc := NewService(repo, zap.NewNop())
	accountID := uuid.New()
	created, err := svc.Create(context.Background(), &CreateRequest{AccountID: accountID, Name: "starter", ClassID: 1})
	if err != nil {
		t.Fatal(err)
	}
	created.Gold -= 123
	created.GoldBind -= 456
	created.RaiseLevelTo(91)
	for i := 0; i < 2; i++ {
		selected, err := svc.Select(context.Background(), &SelectRequest{AccountID: accountID, CharacterID: created.ID})
		if err != nil {
			t.Fatal(err)
		}
		loaded, err := svc.GetByID(context.Background(), created.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, char := range []*domainchar.Character{selected, loaded} {
			if char.Level != 91 || char.Gold != 999876 || char.GoldBind != 999543 {
				t.Fatal("select/load regranted or reset starter package")
			}
		}
	}
}

func TestExistingCharactersDoNotReceiveStarterRewards(t *testing.T) {
	char := domainchar.NewCharacter(uuid.New(), "existing", 1, 0)
	char.ID = 42
	char.RaiseLevelTo(12)
	char.Gold = 20
	char.GoldBind = 30
	repo := &stubCharacterRepo{char: char}
	svc := NewService(repo, zap.NewNop())
	if _, err := svc.Select(context.Background(), &SelectRequest{AccountID: char.AccountID, CharacterID: char.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetByID(context.Background(), char.ID); err != nil {
		t.Fatal(err)
	}
	if char.Level != 12 || char.Gold != 20 || char.GoldBind != 30 || repo.created != nil {
		t.Fatal("existing character changed")
	}
}

func TestStarterRewardsCreationFailures(t *testing.T) {
	createErr := errors.New("create failed")
	for _, tc := range []struct {
		name string
		repo *stubCharacterRepo
		want error
	}{
		{"duplicate", &stubCharacterRepo{exists: true}, pkgerrors.ErrAlreadyExists},
		{"lookup", &stubCharacterRepo{existsErr: createErr}, createErr},
		{"persist", &stubCharacterRepo{createErr: createErr}, createErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			learner := &stubSkillLearner{}
			svc := NewService(tc.repo, zap.NewNop())
			svc.SetSkillLearner(learner)
			svc.SetGameDataManager(mustNewGameDataManagerWithClass(t, `{"id":1,"start_skill":"1010"}`))
			created, err := svc.Create(context.Background(), &CreateRequest{AccountID: uuid.New(), Name: "starter", ClassID: 1})
			if !errors.Is(err, tc.want) || created != nil || len(learner.calls) != 0 {
				t.Fatalf("failed create must not succeed or seed skills: character=%v error=%v calls=%d", created, err, len(learner.calls))
			}
			if tc.name != "persist" && tc.repo.created != nil {
				t.Fatal("repository.Create called after failed name validation")
			}
		})
	}
}
