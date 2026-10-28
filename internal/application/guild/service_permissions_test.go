package guild

import (
	"context"
	"errors"
	"testing"

	domainguild "mcgame-server/internal/domain/guild"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type permissionRepository struct {
	Repository
	guild    *domainguild.Guild
	members  []*domainguild.GuildMember
	settings []domainguild.RankSetting
}

func (r *permissionRepository) GetByID(context.Context, int64) (*domainguild.Guild, error) {
	return r.guild, nil
}

func (r *permissionRepository) GetMembers(context.Context, int64) ([]*domainguild.GuildMember, error) {
	return r.members, nil
}

func (r *permissionRepository) LoadRankSettings(context.Context, int64) ([]domainguild.RankSetting, error) {
	return r.settings, nil
}

func TestAuthorizeActionUsesDisplayedRankSettingsNotMemberBooleans(t *testing.T) {
	settings := DefaultRankSettings()
	settings[domainguild.RankMember-1].CanSlot = false
	repo := &permissionRepository{
		guild:    &domainguild.Guild{ID: 8, LeaderID: 1},
		members:  []*domainguild.GuildMember{{GuildID: 8, CharacterID: 2, Rank: domainguild.RankMember, CanManageWarehouse: true}},
		settings: settings,
	}
	service := NewService(repo, nil, zap.NewNop())
	if err := service.AuthorizeAction(context.Background(), 8, 2, domainguild.PermissionSlot); !errors.Is(err, pkgerrors.ErrUnauthorized) {
		t.Fatalf("legacy member flag bypass error = %v", err)
	}

	repo.settings[domainguild.RankMember-1].CanSlot = true
	if err := service.AuthorizeAction(context.Background(), 8, 2, domainguild.PermissionSlot); err != nil {
		t.Fatal(err)
	}
}

func TestAuthorizeActionFailsClosedOnMalformedSettings(t *testing.T) {
	repo := &permissionRepository{
		guild:    &domainguild.Guild{ID: 8, LeaderID: 1},
		members:  []*domainguild.GuildMember{{GuildID: 8, CharacterID: 2, Rank: domainguild.RankOfficer}},
		settings: []domainguild.RankSetting{{Rank: domainguild.RankOfficer, Name: "Trưởng lão", CanSlot: true}},
	}
	service := NewService(repo, nil, zap.NewNop())
	if err := service.AuthorizeAction(context.Background(), 8, 2, domainguild.PermissionSlot); err == nil {
		t.Fatal("malformed settings must fail closed")
	}
}
