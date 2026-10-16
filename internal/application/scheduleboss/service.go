// Open-sourced by BaoLT

// Schedule Boss application service: kill hook, scene-filter snapshot, config lookup cache.
package scheduleboss

import (
	"context"
	"time"

	"go.uber.org/zap"

	domain "mcgame-server/internal/domain/scheduleboss"
)

const (
	CallbackBossOn  = "onBossOn"
	CallbackBossOff = "onBossOff"
)

type ConfigGetter interface {
	LookupConfig(nid int) (*domain.BossConfig, bool)
}

type MapConfigGetter interface {
	ConfigsForMap(mapID int) []domain.BossConfig
}

type SceneBroadcaster interface {
	BroadcastToScene(channelID, mapID int, excludeConnID uint32, callback string, args ...any)
}

type PayloadBuilder interface {
	BuildBossOn(cfg domain.BossConfig) map[string]any
}

type Service struct {
	state    domain.StateRepo
	configs  ConfigGetter
	scene    SceneBroadcaster
	payloads PayloadBuilder
	logger   *zap.Logger
	clock    func() time.Time
}

func New(state domain.StateRepo, configs ConfigGetter, scene SceneBroadcaster, payloads PayloadBuilder, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		state:    state,
		configs:  configs,
		scene:    scene,
		payloads: payloads,
		logger:   logger,
		clock:    time.Now,
	}
}

func (s *Service) OnBossKilled(ctx context.Context, nid, channelID int, killerID int64) error {
	cfg, ok := s.configs.LookupConfig(nid)
	if !ok || !cfg.Eligible() {
		return nil
	}
	now := s.clock()
	next := now.Add(domain.Cooldown(cfg.Level, cfg.Kind))
	changed, err := s.state.MarkKilled(ctx, nid, channelID, killerID, next, now)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if s.scene != nil {
		s.scene.BroadcastToScene(channelID, cfg.MapID, 0, CallbackBossOff, nid)
	}
	s.logger.Info("schedule boss killed",
		zap.Int("nid", nid),
		zap.Int("channel_id", channelID),
		zap.Int64("killer_id", killerID),
		zap.Time("next_spawn_at", next))
	return nil
}

func (s *Service) HiddenScheduleBosses(ctx context.Context, channelID int, candidateNIDs []int) (map[int]struct{}, error) {
	hidden := make(map[int]struct{})
	tracked := make(map[int]struct{})
	for _, nid := range candidateNIDs {
		cfg, ok := s.configs.LookupConfig(nid)
		if !ok || !cfg.Eligible() {
			continue
		}
		tracked[nid] = struct{}{}
		hidden[nid] = struct{}{}
	}
	if len(tracked) == 0 {
		return hidden, nil
	}
	alive, err := s.state.ListAliveByChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	for _, st := range alive {
		if _, ok := tracked[st.NID]; ok {
			delete(hidden, st.NID)
		}
	}
	return hidden, nil
}

func (s *Service) CanEngage(ctx context.Context, nid, mapID, channelID int) (bool, error) {
	cfg, ok := s.LookupMapBoss(nid)
	if !ok || cfg.MapID != mapID || !cfg.AllowsChannel(channelID) {
		return false, nil
	}
	if cfg.Tier != domain.TierNormal {
		return true, nil
	}
	state, err := s.state.Get(ctx, nid, channelID)
	if err != nil {
		return false, err
	}
	return state != nil && state.IsAlive, nil
}

func (s *Service) VisibleBossesForScene(ctx context.Context, mapID, channelID int) ([]domain.BossConfig, error) {
	getter, ok := s.configs.(MapConfigGetter)
	if !ok || channelID < 0 {
		return nil, nil
	}
	configs := getter.ConfigsForMap(mapID)
	if len(configs) == 0 {
		return nil, nil
	}
	alive, err := s.state.ListAliveByChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	aliveByNID := make(map[int]struct{}, len(alive))
	for _, item := range alive {
		if item.IsAlive {
			aliveByNID[item.NID] = struct{}{}
		}
	}
	visible := make([]domain.BossConfig, 0, len(configs))
	for _, cfg := range configs {
		if !cfg.IsMapBoss() || !cfg.AllowsChannel(channelID) {
			continue
		}
		if cfg.Tier == domain.TierNormal {
			if _, ok := aliveByNID[cfg.NID]; !ok {
				continue
			}
		}
		visible = append(visible, cfg)
	}
	return visible, nil
}

type ConfigCache struct {
	byNID map[int]*domain.BossConfig
	byMap map[int][]domain.BossConfig
}

func NewConfigCache(items []domain.BossConfig) *ConfigCache {
	c := &ConfigCache{
		byNID: make(map[int]*domain.BossConfig, len(items)),
		byMap: make(map[int][]domain.BossConfig),
	}
	for i := range items {
		it := items[i]
		c.byNID[it.NID] = &it
		if it.MapID > 0 {
			c.byMap[it.MapID] = append(c.byMap[it.MapID], it)
		}
	}
	return c
}

func (c *ConfigCache) LookupConfig(nid int) (*domain.BossConfig, bool) {
	v, ok := c.byNID[nid]
	return v, ok
}

func (c *ConfigCache) ConfigsForMap(mapID int) []domain.BossConfig {
	items := c.byMap[mapID]
	return append([]domain.BossConfig(nil), items...)
}

func (s *Service) LookupConfig(nid int) (*domain.BossConfig, bool) {
	if s == nil || s.configs == nil {
		return nil, false
	}
	return s.configs.LookupConfig(nid)
}

func (s *Service) LookupMapBoss(nid int) (*domain.BossConfig, bool) {
	cfg, ok := s.LookupConfig(nid)
	if !ok || !cfg.IsMapBoss() {
		return nil, false
	}
	return cfg, true
}
