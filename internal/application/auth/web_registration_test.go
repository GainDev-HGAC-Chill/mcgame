package auth

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	domainauth "mcgame-server/internal/domain/auth"
	pkgerrors "mcgame-server/pkg/errors"
)

type registrationAccountRepo struct {
	domainauth.AccountRepository
	account *domainauth.Account
	err     error
}

func (r *registrationAccountRepo) ExistsByUsername(context.Context, string) (bool, error) {
	return r.account != nil, r.err
}

func (r *registrationAccountRepo) Create(_ context.Context, account *domainauth.Account) error {
	r.account = account
	return r.err
}

func (r *registrationAccountRepo) FindByUsername(_ context.Context, name string) (*domainauth.Account, error) {
	if r.account == nil || r.account.Username != name {
		return nil, pkgerrors.ErrAccountNotFound
	}
	return r.account, nil
}

func (r *registrationAccountRepo) UpdateLastLogin(context.Context, uuid.UUID) error { return nil }

type registrationSessionRepo struct {
	domainauth.SessionRepository
	saved int
}

func (r *registrationSessionRepo) DeleteByAccountID(context.Context, uuid.UUID) error { return nil }
func (r *registrationSessionRepo) Save(context.Context, *domainauth.Session) error {
	r.saved++
	return nil
}

func TestWebRegistrationValidation(t *testing.T) {
	for _, tc := range []struct {
		name         string
		username     string
		password     string
		confirmation string
		valid        bool
	}{
		{"one", "1", "1", "1", true},
		{"whitespace", " Ab C ", " \t ", " \t ", true},
		{"only spaces", " ", " ", " ", true},
		{"unicode", "Tài khoản😀", "mật😀khẩu", "mật😀khẩu", true},
		{"replacement rune", "�", "�", "�", true},
		{"username ceiling", strings.Repeat("😀", 25), "1", "1", true},
		{"username over", strings.Repeat("😀", 25) + "a", "1", "1", false},
		{"username bmp ceiling", strings.Repeat("ế", 50), "1", "1", true},
		{"username bmp over", strings.Repeat("ế", 51), "1", "1", false},
		{"password ceiling", "1", strings.Repeat("😀", 10), strings.Repeat("😀", 10), true},
		{"password over", "1", strings.Repeat("😀", 10) + "1", strings.Repeat("😀", 10) + "1", false},
		{"empty username", "", "1", "1", false},
		{"empty password", "1", "", "", false},
		{"confirmation mismatch", "1", " A ", "A", false},
		{"username nul", "a\x00", "1", "1", false},
		{"username utf8", "\xff", "1", "1", false},
		{"password utf8", "1", "\xff", "\xff", false},
		{"confirmation utf8", "1", "1", "\xff", false},
		{"password nul", "1", "\x00", "\x00", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := &WebRegistrationRequest{tc.username, tc.password, tc.confirmation}
			if err := req.Validate(); (err == nil) != tc.valid {
				t.Fatalf("validation valid=%v, error=%v", tc.valid, err)
			}
		})
	}
	if (*WebRegistrationRequest)(nil).Validate() == nil {
		t.Fatal("nil request accepted")
	}
}

func TestWebRegistrationFlashCompatibility(t *testing.T) {
	for _, tc := range []struct{ name, username, password, digest string }{
		{"one", "1", "1", "c4ca4238a0b923820dcc509a6f75849b"},
		{"unicode", " Tên😀 ", "😀", "2a02eac39d716a70ecf37579185927b6"},
		{"bmp", " CASE ", "é", "66ddcd97cfdeabb2f6fb8a999b4bc76f"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &registrationAccountRepo{}
			sessions := &registrationSessionRepo{}
			core, logs := observer.New(zap.DebugLevel)
			service := NewService(repo, sessions, zap.New(core))
			req := &WebRegistrationRequest{tc.username, tc.password, tc.password}
			if err := service.RegisterWeb(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			account := repo.account
			if account.Username != tc.username || account.Email != "" || account.Gold != 0 || sessions.saved != 0 {
				t.Fatal("registration changed identity or created session/wallet state")
			}
			if !domainauth.IsBcryptHash(account.PasswordHash) || !account.CheckPassword(tc.digest) || account.CheckPassword(tc.password) {
				t.Fatal("stored password is not compatible with Flash digest")
			}
			double := md5.Sum([]byte(tc.digest))
			if account.CheckPassword(hex.EncodeToString(double[:])) {
				t.Fatal("double digest accepted")
			}
			for _, entry := range logs.All() {
				for key := range entry.ContextMap() {
					if strings.Contains(strings.ToLower(key), "password") || strings.Contains(strings.ToLower(key), "hash") {
						t.Fatal("credentials logged")
					}
				}
			}
			if _, err := service.Login(context.Background(), &LoginRequest{Username: tc.username, Password: tc.digest}); err != nil {
				t.Fatal(err)
			}
			if err := service.RegisterWeb(context.Background(), req); !errors.Is(err, pkgerrors.ErrAlreadyExists) {
				t.Fatal("duplicate not mapped")
			}
		})
	}
}

func TestRegisterBehaviorUnchanged(t *testing.T) {
	repo := &registrationAccountRepo{}
	service := NewService(repo, &registrationSessionRepo{}, zap.NewNop())
	if err := service.Register(context.Background(), &RegisterRequest{Username: "existing-contract", Password: "unchanged", Email: "local@example.invalid"}); err != nil {
		t.Fatal(err)
	}
	if !repo.account.CheckPassword("unchanged") || repo.account.Email != "local@example.invalid" {
		t.Fatal("existing Register contract changed")
	}
}

func TestRegisterWebRejectsBeforeRepository(t *testing.T) {
	service := NewService(nil, nil, zap.NewNop())
	if err := service.RegisterWeb(context.Background(), &WebRegistrationRequest{Username: "1"}); !errors.Is(err, pkgerrors.ErrInvalidInput) {
		t.Fatal("invalid input not rejected")
	}
}
