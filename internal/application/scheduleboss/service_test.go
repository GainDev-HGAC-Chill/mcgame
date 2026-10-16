// Open-sourced by BaoLT

// Tests for OnBossKilled idempotency, cooldown calculation, and shared in-memory fakes.
package scheduleboss

import (
	"context"
	"testing"
	"time"

	domain "mcgame-server/internal/domain/scheduleboss"
)

type memStateRepo struct {
	existing  map[domain.BossKey]struct{}
	inserted  map[domain.BossKey]domain.BossState
	killed    map[domain.BossKey]domain.BossState
	alive     map[domain.BossKey]bool
	due       []domain.BossState
	aliveList map[int][]domain.BossState

	markAliveChanged  bool
	markKilledChanged bool
}

func newMemStateRepo() *memStateRepo {
	return &memStateRepo{
		existing:          make(map[domain.BossKey]struct{}),
		inserted:          make(map[domain.BossKey]domain.BossState),
		killed:            make(map[domain.BossKey]domain.BossState),
		alive:             make(map[domain.BossKey]bool),
		aliveList:         make(map[int][]domain.BossState),
		markAliveChanged:  true,
		markKilledChanged: true,
	}
}

func (m *memStateRepo) Upsert(context.Context, domain.BossState) error { return nil }

func (m *memStateRepo) Get(_ context.Context, nid, ch int) (*domain.BossState, error) {
	alive, ok := m.alive[domain.BossKey{NID: nid, ChannelID: ch}]
	if !ok {
		return nil, nil
	}
	return &domain.BossState{NID: nid, ChannelID: ch, IsAlive: alive}, nil
}

func (m *memStateRepo) FetchDueSpawns(context.Context, time.Time) ([]domain.BossState, error) {
	return m.due, nil
}

func (m *memStateRepo) MarkAlive(_ context.Context, nid, ch int, _ time.Time) (bool, error) {
	m.alive[domain.BossKey{NID: nid, ChannelID: ch}] = true
	return m.markAliveChanged, nil
}

func (m *memStateRepo) MarkKilled(_ context.Context, nid, ch int, _ int64, next time.Time, _ time.Time) (bool, error) {
	m.killed[domain.BossKey{NID: nid, ChannelID: ch}] = domain.BossState{NID: nid, ChannelID: ch, IsAlive: false, NextSpawnAt: next}
	return m.markKilledChanged, nil
}

func (m *memStateRepo) ListAliveByChannel(_ context.Context, ch int) ([]domain.BossState, error) {
	return m.aliveList[ch], nil
}

func (m *memStateRepo) ListExistingKeys(context.Context) (map[domain.BossKey]struct{}, error) {
	return m.existing, nil
}

func (m *memStateRepo) InsertMissing(_ context.Context, rows []domain.BossState) error {
	for _, r := range rows {
		m.inserted[domain.BossKey{NID: r.NID, ChannelID: r.ChannelID}] = r
	}
	return nil
}

type mapConfigGetter map[int]domain.BossConfig

func (m mapConfigGetter) LookupConfig(nid int) (*domain.BossConfig, bool) {
	c, ok := m[nid]
	if !ok {
		return nil, false
	}
	return &c, true
}

func TestOnBossKilled_AppliesCooldown(t *testing.T) {
	cfg := domain.BossConfig{NID: 706, Name: "Thất Sắc Kê", Level: 5, Kind: domain.Ground, Tier: domain.TierNormal, IsActive: true}
	state := newMemStateRepo()
	svc := New(state, mapConfigGetter{706: cfg}, nil, nil, nil)
	now := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	svc.clock = func() time.Time { return now }

	if err := svc.OnBossKilled(context.Background(), 706, 3, 42); err != nil {
		t.Fatalf("OnBossKilled: %v", err)
	}
	got, ok := state.killed[domain.BossKey{NID: 706, ChannelID: 3}]
	if !ok {
		t.Fatalf("expected MarkKilled to be called")
	}
	if !got.NextSpawnAt.Equal(now.Add(8 * time.Hour)) {
		t.Fatalf("next_spawn_at = %s; want now+8h", got.NextSpawnAt)
	}
}

func TestCanEngageRequiresMapChannelAndAliveState(t *testing.T) {
	cfg := domain.BossConfig{NID: 706, MapID: 3, Kind: domain.Ground, Tier: domain.TierNormal, ChannelScope: "all", IsActive: true}
	state := newMemStateRepo()
	state.alive[domain.BossKey{NID: 706, ChannelID: 2}] = true
	svc := New(state, NewConfigCache([]domain.BossConfig{cfg}), nil, nil, nil)

	tests := []struct {
		name    string
		mapID   int
		channel int
		want    bool
	}{
		{"valid", 3, 2, true},
		{"wrong map", 4, 2, false},
		{"dead channel", 3, 3, false},
		{"invalid channel", 3, -1, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := svc.CanEngage(context.Background(), 706, test.mapID, test.channel)
			if err != nil {
				t.Fatalf("CanEngage: %v", err)
			}
			if got != test.want {
				t.Fatalf("CanEngage() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestCanEngageAllowsLiveBossOnChannelZero(t *testing.T) {
	cfg := domain.BossConfig{NID: 707, MapID: 2, Kind: domain.Ground, Tier: domain.TierNormal, ChannelScope: "all", IsActive: true}
	state := newMemStateRepo()
	state.alive[domain.BossKey{NID: 707, ChannelID: 0}] = true
	svc := New(state, NewConfigCache([]domain.BossConfig{cfg}), nil, nil, nil)

	got, err := svc.CanEngage(context.Background(), 707, 2, 0)
	if err != nil {
		t.Fatalf("CanEngage: %v", err)
	}
	if !got {
		t.Fatal("channel zero boss should be engageable when its state is alive")
	}
}

func TestVisibleBossesForSceneIncludesScopedMythicAndAliveNormal(t *testing.T) {
	configs := []domain.BossConfig{
		{NID: 706, MapID: 3, Kind: domain.Ground, Tier: domain.TierNormal, ChannelScope: "all", IsActive: true},
		{NID: 169, MapID: 3, Kind: domain.Ground, Tier: domain.TierMythic, ChannelScope: "channel:1", IsActive: true},
		{NID: 2204, MapID: 3, Kind: domain.Kind("daily_only"), Tier: domain.TierNormal, ChannelScope: "all", IsActive: true},
	}
	state := newMemStateRepo()
	state.aliveList[1] = []domain.BossState{{NID: 706, ChannelID: 1, IsAlive: true}}
	svc := New(state, NewConfigCache(configs), nil, nil, nil)

	visible, err := svc.VisibleBossesForScene(context.Background(), 3, 1)
	if err != nil {
		t.Fatalf("VisibleBossesForScene: %v", err)
	}
	if len(visible) != 2 || visible[0].NID != 706 || visible[1].NID != 169 {
		t.Fatalf("visible = %+v, want normal 706 and mythic 169", visible)
	}

	visible, err = svc.VisibleBossesForScene(context.Background(), 3, 2)
	if err != nil {
		t.Fatalf("VisibleBossesForScene channel 2: %v", err)
	}
	if len(visible) != 0 {
		t.Fatalf("channel 2 visible = %+v, want none", visible)
	}

	state.aliveList[0] = []domain.BossState{{NID: 706, ChannelID: 0, IsAlive: true}}
	visible, err = svc.VisibleBossesForScene(context.Background(), 3, 0)
	if err != nil {
		t.Fatalf("VisibleBossesForScene channel 0: %v", err)
	}
	if len(visible) != 1 || visible[0].NID != 706 {
		t.Fatalf("channel 0 visible = %+v, want normal 706", visible)
	}
}
