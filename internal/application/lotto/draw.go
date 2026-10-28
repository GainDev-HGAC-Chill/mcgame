// Open-sourced by BaoLT

// Wish-Tree draw RPCs (getLottoData / lottoByClient). The draw pool is TBL_PLAN filtered
// by the t (type) column and weighted by the r column; a single draw picks one plan row
// proportional to r and deposits {ti,ii,n,q,b} into the lotto holding-bag, from where the
// player claims it via the existing getSingle/getAll RPCs.
//
// Costs and the daily free-quota are not in any data table; they are taken verbatim from
// the client (Language.as wish-button labels): per draw-type x{1,5,10} gold (vàng) costs,
// and a 5-per-day free allowance for type-1 single wishes. Both are documented calibration
// constants — retune against a live-log if the real values differ.
//
// Draw deducts gold (DB-backed char update) before depositing winnings (feature-state
// upsert) — consume-then-credit, so a save failure cannot mint free items. The two writes
// are not in one transaction; like every other feature service here it relies on the
// per-connection serial RTMP dispatch (monolith deploy) to avoid a same-character race.
package lotto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	domainachievement "mcgame-server/internal/domain/achievement"
	domainpet "mcgame-server/internal/domain/pet"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"

	"github.com/google/uuid"
)

const (
	lottoTypeMin      = 1
	lottoTypeMax      = 3
	lottoDisplaySlots = 18
	lottoDisplayTypes = 4
	lottoBagCap       = 500
	freeWishType      = 1
	freeWishDailyMax  = 5
)

var (
	ErrLottoInvalidType  = errors.New("lotto: invalid wish type")
	ErrLottoInvalidNum   = errors.New("lotto: invalid wish count")
	ErrLottoNoPool       = errors.New("lotto: empty draw pool")
	ErrLottoNoGameData   = errors.New("lotto: game data unavailable")
	ErrLottoNoCharRepo   = errors.New("lotto: character repository unavailable")
	ErrLottoInsufficient = errors.New("lotto: not enough gold")
	ErrLottoBagFull      = errors.New("lotto: holding bag full")
)

var lottoGoldCost = map[int]map[int]int64{
	1: {1: 20, 5: 90, 10: 160},
	2: {1: 100, 5: 450, 10: 800},
	3: {1: 200, 5: 900, 10: 1600},
}

func lottoToday() string { return time.Now().Format("2006-01-02") }

func (s *Service) GetLottoData(ctx context.Context, charID int64) (map[string]any, error) {
	if s.gameData == nil {
		return nil, ErrLottoNoGameData
	}
	award := make([]any, 0, lottoDisplayTypes)
	for t := 1; t <= lottoDisplayTypes; t++ {
		award = append(award, s.displayPoolIDs(t))
	}
	return map[string]any{
		"lottoAward":      award,
		"highestAwardArr": []any{},
	}, nil
}

func (s *Service) displayPoolIDs(t int) []any {
	pool := s.gameData.PlansByType(t)
	sort.Slice(pool, func(i, j int) bool { return pool[i].ID < pool[j].ID })
	ids := make([]any, 0, lottoDisplaySlots)
	for _, p := range pool {
		if len(ids) >= lottoDisplaySlots {
			break
		}
		ids = append(ids, p.ID)
	}
	return ids
}

func (s *Service) Draw(ctx context.Context, charID int64, drawType, num int) ([]any, error) {
	if drawType < lottoTypeMin || drawType > lottoTypeMax {
		return nil, ErrLottoInvalidType
	}
	cost, ok := lottoGoldCost[drawType][num]
	if !ok {
		return nil, ErrLottoInvalidNum
	}
	if s.gameData == nil {
		return nil, ErrLottoNoGameData
	}
	pool := s.gameData.PlansByType(drawType)
	if len(pool) == 0 {
		return nil, ErrLottoNoPool
	}

	state, err := s.loadState(ctx, charID)
	if err != nil {
		return nil, err
	}
	expectedState := state.ToMap()
	today := lottoToday()
	if state.FreeWishDay != today {
		state.FreeWishN = 0
		state.FreeWishDay = today
	}

	bag := state.bag(BagLotto)
	if len(*bag)+num > lottoBagCap {
		return nil, ErrLottoBagFull
	}

	free := drawType == freeWishType && num == 1 && state.FreeWishN < freeWishDailyMax
	if free {
		cost = 0
	}
	if s.chars == nil {
		return nil, ErrLottoNoCharRepo
	}
	char, err := s.chars.FindByID(ctx, charID)
	if err != nil {
		return nil, err
	}
	if char == nil {
		return nil, ErrLottoNoCharRepo
	}
	if char.Gold < cost {
		return nil, ErrLottoInsufficient
	}
	committer, ok := s.featureRepo.(domainfeature.LottoDrawCommitRepository)
	if !ok {
		return nil, fmt.Errorf("lotto: atomic draw repository unavailable")
	}
	drawID := uuid.NewString()
	rolledPlans := make([]*models.PlanTemplate, 0, num)
	for i := 0; i < num; i++ {
		plan := s.rollPlan(pool)
		rolledPlans = append(rolledPlans, plan)
		*bag = append(*bag, planToEntry(plan))
	}
	if free {
		state.FreeWishN++
	}
	facts, err := wishPetFacts(charID, drawID, rolledPlans, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	newGold := char.Gold - cost
	if err := committer.CommitLottoDraw(ctx, charID, newGold, expectedState, state.ToMap(), facts); err != nil {
		return nil, err
	}
	char.Gold = newGold
	if synchronizer, ok := s.chars.(interface{ SyncPersistedGold(int64, int64) }); ok {
		synchronizer.SyncPersistedGold(charID, newGold)
	}
	return entriesToList(*bag), nil
}

func wishPetFacts(charID int64, drawID string, plans []*models.PlanTemplate, occurredAt time.Time) ([]domainachievement.Fact, error) {
	facts := make([]domainachievement.Fact, 0)
	for ordinal, plan := range plans {
		if plan == nil || plan.Ti != 12 {
			continue
		}
		growRate := plan.Q
		if growRate >= 10 {
			growRate /= 10
		}
		if growRate <= 0 || math.IsNaN(growRate) || math.IsInf(growRate, 0) || plan.ID <= 0 || plan.Ii <= 0 || plan.Ii != math.Trunc(plan.Ii) {
			return nil, fmt.Errorf("lotto: invalid Wish Pet plan %d", plan.ID)
		}
		payload := domainachievement.WishPetDrawnFactPayload{
			DrawID:             drawID,
			Ordinal:            ordinal,
			PlanID:             plan.ID,
			CreatureTemplateID: int64(plan.Ii),
			GrowRate:           growRate,
			ColorTier:          domainpet.ColorByGrowRate(growRate),
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		facts = append(facts, domainachievement.Fact{
			FactID:      fmt.Sprintf("wish-pet-drawn:v1:%s:%d", drawID, ordinal),
			CharacterID: charID,
			Kind:        domainachievement.FactKindWishPetDrawn,
			Payload:     raw,
			OccurredAt:  occurredAt,
		})
	}
	return facts, nil
}

func (s *Service) rollPlan(pool []*models.PlanTemplate) *models.PlanTemplate {
	total := 0.0
	for _, p := range pool {
		if p.R > 0 {
			total += p.R
		}
	}
	if total <= 0 {
		s.rngMu.Lock()
		idx := s.rng.Intn(len(pool))
		s.rngMu.Unlock()
		return pool[idx]
	}
	s.rngMu.Lock()
	roll := s.rng.Float64() * total
	s.rngMu.Unlock()
	acc := 0.0
	for _, p := range pool {
		if p.R <= 0 {
			continue
		}
		acc += p.R
		if roll < acc {
			return p
		}
	}
	return pool[len(pool)-1]
}

func planToEntry(p *models.PlanTemplate) LottoEntry {
	n := int(p.N)
	if n <= 0 {
		n = 1
	}
	return LottoEntry{
		Ti: strconv.Itoa(int(p.Ti)),
		Ii: strconv.Itoa(int(p.Ii)),
		N:  n,
		Q:  strconv.FormatFloat(p.Q, 'f', -1, 64),
		B:  int(p.B),
	}
}

// GetGold returns the latest persisted gold balance.
func (s *Service) GetGold(ctx context.Context, charID int64) (int64, error) {
	if s.chars == nil {
		return 0, ErrLottoNoCharRepo
	}

	char, err := s.chars.FindByID(ctx, charID)
	if err != nil {
		return 0, err
	}
	if char == nil {
		return 0, ErrLottoNoCharRepo
	}

	return char.Gold, nil
}
