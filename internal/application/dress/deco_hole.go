// Open-sourced by BaoLT

// Decoration hole level-up service for the addDecoHoleLevel RPC.
// State is persisted via the generic character feature-state store under FeatureDecoHole.
// Each position (1-4) tracks the current hole template id, open level, and accumulated rune stats.
// Currency deductions use the character's DecoSilver balance (CurrencyDecoSilver = 216).
package dress

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	domainchar "mcgame-server/internal/domain/character"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

const (
	decoHolePositionMin = 1
	decoHolePositionMax = 4
)

var (
	ErrDecoHoleInvalidPosition       = errors.New("deco: invalid hole position")
	ErrDecoHoleInsufficientSil       = errors.New("deco: not enough decoration silver")
	ErrDecoHoleInsufficientMoneyBind = errors.New("deco: not enough bound silver")
)

type HolePosition struct {
	ID         int64
	Cid        int64
	Did        int64
	Hid        int64
	IsShow     int
	N          int
	Position   int
	ShowLvl    int
	ActiveFlag string
	R          [16]int
}

type DecoHoleState struct {
	Positions map[int]*HolePosition
}

type featureStateRepo interface {
	ListCharacterFeatureStates(ctx context.Context, charID int64) ([]*domainfeature.CharacterFeatureState, error)
	UpsertCharacterFeatureState(ctx context.Context, state *domainfeature.CharacterFeatureState) error
}

type charRepoForDeco interface {
	FindByID(ctx context.Context, id int64) (*domainchar.Character, error)
	Update(ctx context.Context, char *domainchar.Character) error
}

type DecoHoleService struct {
	featureRepo   featureStateRepo
	charRepo      charRepoForDeco
	gameDataMgr   decoHoleGameData
	logger        *zap.Logger
	rng           *rand.Rand
	rngMu         sync.Mutex
	mutationLocks [64]sync.Mutex
}

type decoHoleGameData interface {
	GetDecoHole(id int) *models.DecoHoleTemplate
	GetDecoRune(id int) *models.DecoRuneTemplate
	GetDecoShow(id int) *models.DecoShowTemplate
	GetDecoShowsBySuitID(suitID int) []*models.DecoShowTemplate
}

func NewDecoHoleService(
	featureRepo featureStateRepo,
	charRepo charRepoForDeco,
	gameData decoHoleGameData,
	logger *zap.Logger,
) *DecoHoleService {
	return &DecoHoleService{
		featureRepo: featureRepo,
		charRepo:    charRepo,
		gameDataMgr: gameData,
		logger:      logger,
		rng:         rand.New(rand.NewSource(randSeed())),
	}
}

func randSeed() int64 {
	return time.Now().UnixNano()
}

func (s *DecoHoleService) loadState(ctx context.Context, charID int64) (*DecoHoleState, error) {
	states, err := s.featureRepo.ListCharacterFeatureStates(ctx, charID)
	if err != nil {
		return nil, fmt.Errorf("deco_hole: load feature state: %w", err)
	}
	for _, st := range states {
		if st == nil || st.FeatureKey != domainfeature.FeatureDecoHole {
			continue
		}
		return decodeDecoHoleState(charID, st.State), nil
	}
	return s.bootstrapState(charID), nil
}

func (s *DecoHoleService) bootstrapState(charID int64) *DecoHoleState {
	return bootstrapStateStatic(charID)
}

func (s *DecoHoleService) saveState(ctx context.Context, charID int64, state *DecoHoleState) error {
	encoded := encodeDecoHoleState(state)
	return s.featureRepo.UpsertCharacterFeatureState(ctx, &domainfeature.CharacterFeatureState{
		CharacterID: charID,
		FeatureKey:  domainfeature.FeatureDecoHole,
		State:       encoded,
	})
}

type DecoHoleLevelResult struct {
	State           *DecoHoleState
	SpentDecoSilver int64
	SpentMoneyBind  int64
	DecoSilverAfter int64
	MoneyBindAfter  int64
}

func (s *DecoHoleService) AddDecoHoleLevel(ctx context.Context, charID int64, position int) (*DecoHoleLevelResult, error) {
	return s.addDecoHoleLevels(ctx, charID, position, 1)
}

func (s *DecoHoleService) AddDecoHoleLevelTenTimes(ctx context.Context, charID int64, position int) (*DecoHoleLevelResult, error) {
	return s.addDecoHoleLevels(ctx, charID, position, 10)
}

func (s *DecoHoleService) lockCharacterMutation(charID int64) func() {
	lock := &s.mutationLocks[uint64(charID)%uint64(len(s.mutationLocks))]
	lock.Lock()
	return lock.Unlock
}

func (s *DecoHoleService) addDecoHoleLevels(ctx context.Context, charID int64, position, attempts int) (*DecoHoleLevelResult, error) {
	if position < decoHolePositionMin || position > decoHolePositionMax {
		return nil, ErrDecoHoleInvalidPosition
	}
	unlock := s.lockCharacterMutation(charID)
	defer unlock()

	char, err := s.charRepo.FindByID(ctx, charID)
	if err != nil {
		return nil, fmt.Errorf("deco_hole: load character: %w", err)
	}

	state, err := s.loadState(ctx, charID)
	if err != nil {
		return nil, err
	}

	hole, ok := state.Positions[position]
	if !ok {
		return nil, ErrDecoHoleInvalidPosition
	}

	originalDecoSilver := char.DecoSilver
	originalMoneyBind := char.MoneyBind
	spentDecoSilver := 0
	spentMoneyBind := int64(0)
	stateChanged := false

	for attempt := 0; attempt < attempts; attempt++ {
		tpl := s.gameDataMgr.GetDecoHole(int(hole.Hid))
		if tpl == nil || int(tpl.NextID) <= 0 {
			break
		}
		nextTpl := s.gameDataMgr.GetDecoHole(int(tpl.NextID))
		if nextTpl == nil || int(nextTpl.Position) != position || int(nextTpl.Level) != int(tpl.Level)+1 {
			break
		}
		costDecoSilver := int(tpl.CostNum)
		costMoneyBind := int64(tpl.CostSil)
		if char.DecoSilver < costDecoSilver {
			if attempt == 0 {
				return nil, ErrDecoHoleInsufficientSil
			}
			break
		}
		if char.MoneyBind < costMoneyBind {
			if attempt == 0 {
				return nil, ErrDecoHoleInsufficientMoneyBind
			}
			break
		}
		char.DecoSilver -= costDecoSilver
		char.MoneyBind -= costMoneyBind
		spentDecoSilver += costDecoSilver
		spentMoneyBind += costMoneyBind

		s.rngMu.Lock()
		roll := s.rng.Intn(100)
		s.rngMu.Unlock()
		if roll < int(tpl.Rate) {
			hole.Hid = nextTpl.ID
			stateChanged = true
		}
	}

	if spentDecoSilver == 0 && spentMoneyBind == 0 {
		return &DecoHoleLevelResult{State: state, DecoSilverAfter: int64(char.DecoSilver), MoneyBindAfter: char.MoneyBind}, nil
	}
	if err := s.charRepo.Update(ctx, char); err != nil {
		char.DecoSilver = originalDecoSilver
		char.MoneyBind = originalMoneyBind
		return nil, fmt.Errorf("deco_hole: persist character deduction: %w", err)
	}
	if stateChanged {
		if err := s.saveState(ctx, charID, state); err != nil {
			char.DecoSilver = originalDecoSilver
			char.MoneyBind = originalMoneyBind
			if restoreErr := s.charRepo.Update(ctx, char); restoreErr != nil {
				return nil, fmt.Errorf("deco_hole: persist state: %w; restore currency: %v", err, restoreErr)
			}
			return nil, fmt.Errorf("deco_hole: persist state: %w", err)
		}
	}

	return &DecoHoleLevelResult{State: state, SpentDecoSilver: int64(spentDecoSilver), SpentMoneyBind: spentMoneyBind, DecoSilverAfter: int64(char.DecoSilver), MoneyBindAfter: char.MoneyBind}, nil
}

func (s *DecoHoleService) GetDecoHoleState(ctx context.Context, charID int64) (*DecoHoleState, error) {
	return s.loadState(ctx, charID)
}

func DecoHoleStateToWire(charID int64, state *DecoHoleState) map[string]interface{} {
	if state == nil {
		return map[string]interface{}{}
	}
	wire := make(map[string]interface{}, len(state.Positions))
	for pos, hole := range state.Positions {
		wire[strconv.Itoa(pos)] = holeToWire(charID, hole)
	}
	return wire
}

func holeToWire(charID int64, h *HolePosition) map[string]interface{} {
	if h == nil {
		return map[string]interface{}{}
	}
	row := map[string]interface{}{
		"activeFlag": h.ActiveFlag,
		"cid":        strconv.FormatInt(charID, 10),
		"did":        strconv.FormatInt(h.Did, 10),
		"hid":        strconv.FormatInt(h.Hid, 10),
		"id":         strconv.FormatInt(h.ID, 10),
		"isShow":     strconv.Itoa(h.IsShow),
		"n":          strconv.Itoa(h.N),
		"position":   strconv.Itoa(h.Position),
		"showLvl":    strconv.Itoa(h.ShowLvl),
	}
	for i := 0; i < 16; i++ {
		row["r"+strconv.Itoa(i+1)] = strconv.Itoa(h.R[i])
	}
	return row
}

func decodeDecoHoleState(charID int64, raw map[string]interface{}) *DecoHoleState {
	state := &DecoHoleState{Positions: make(map[int]*HolePosition, decoHolePositionMax)}
	posMap, _ := raw["positions"].(map[string]interface{})
	if posMap == nil {
		return bootstrapStateStatic(charID)
	}
	for posKey, posVal := range posMap {
		posNum, err := strconv.Atoi(posKey)
		if err != nil || posNum < decoHolePositionMin || posNum > decoHolePositionMax {
			continue
		}
		m, ok := posVal.(map[string]interface{})
		if !ok {
			continue
		}
		hole := &HolePosition{Position: posNum}
		if v, ok := intFromAny(m["id"]); ok {
			hole.ID = int64(v)
		}
		if v, ok := intFromAny(m["hid"]); ok {
			hole.Hid = int64(v)
		}
		if v, ok := intFromAny(m["did"]); ok {
			hole.Did = int64(v)
		}
		if v, ok := intFromAny(m["isShow"]); ok {
			hole.IsShow = v
		}
		if v, ok := intFromAny(m["n"]); ok {
			hole.N = v
		}
		if v, ok := intFromAny(m["showLvl"]); ok {
			hole.ShowLvl = v
		}
		if s, ok := m["activeFlag"].(string); ok {
			hole.ActiveFlag = s
		}
		if hole.ActiveFlag == "" {
			hole.ActiveFlag = "{r:{},s:{}}"
		}
		for i := 0; i < 16; i++ {
			key := "r" + strconv.Itoa(i+1)
			if v, ok := intFromAny(m[key]); ok {
				hole.R[i] = v
			}
		}
		state.Positions[posNum] = hole
	}
	for pos := decoHolePositionMin; pos <= decoHolePositionMax; pos++ {
		if _, exists := state.Positions[pos]; !exists {
			base := [decoHolePositionMax]int{1, 2, 3, 4}
			state.Positions[pos] = &HolePosition{
				Cid:        charID,
				Hid:        int64(base[pos-1]),
				Position:   pos,
				ActiveFlag: "{r:{},s:{}}",
			}
		}
	}
	return state
}

func bootstrapStateStatic(charID int64) *DecoHoleState {
	state := &DecoHoleState{Positions: make(map[int]*HolePosition, decoHolePositionMax)}
	baseIDs := [decoHolePositionMax]int{1, 2, 3, 4}
	for idx, baseID := range baseIDs {
		pos := idx + 1
		state.Positions[pos] = &HolePosition{
			ID:         0,
			Cid:        charID,
			Did:        0,
			Hid:        int64(baseID),
			IsShow:     0,
			N:          0,
			Position:   pos,
			ShowLvl:    0,
			ActiveFlag: "{r:{},s:{}}",
		}
	}
	return state
}

func encodeDecoHoleState(state *DecoHoleState) map[string]interface{} {
	if state == nil {
		return map[string]interface{}{}
	}
	positions := make(map[string]interface{}, len(state.Positions))
	for pos, hole := range state.Positions {
		if hole == nil {
			continue
		}
		m := map[string]interface{}{
			"id":         hole.ID,
			"hid":        hole.Hid,
			"did":        hole.Did,
			"isShow":     hole.IsShow,
			"n":          hole.N,
			"showLvl":    hole.ShowLvl,
			"activeFlag": hole.ActiveFlag,
		}
		for i := 0; i < 16; i++ {
			m["r"+strconv.Itoa(i+1)] = hole.R[i]
		}
		positions[strconv.Itoa(pos)] = m
	}
	return map[string]interface{}{"positions": positions}
}

func intFromAny(v interface{}) (int, bool) {
	switch typed := v.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float32:
		return int(typed), true
	case float64:
		return int(typed), true
	}
	return 0, false
}
