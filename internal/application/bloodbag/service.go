// Blood-bag service validates and persists the four automatic recovery reserves.
package bloodbag

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	domainchar "mcgame-server/internal/domain/character"
	domainfeature "mcgame-server/internal/domain/statfeature"
)

const Capacity int64 = 10_000_000

var (
	ErrInvalidRequest   = errors.New("blood bag: invalid request")
	ErrCapacityExceeded = errors.New("blood bag: capacity exceeded")
	ErrInsufficientGold = errors.New("blood bag: not enough gold")
)

type State struct {
	CharacterHP int64
	CharacterMP int64
	PetHP       int64
	PetMP       int64
}

type Result struct {
	State   State
	Cost    int64
	NewGold int64
}

type Service struct {
	charRepo    characterRepository
	featureRepo featureRepository
	locks       sync.Map
}

type characterRepository interface {
	FindByID(context.Context, int64) (*domainchar.Character, error)
	Update(context.Context, *domainchar.Character) error
}

type featureRepository interface {
	ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error)
	UpsertCharacterFeatureState(context.Context, *domainfeature.CharacterFeatureState) error
}

func NewService(charRepo characterRepository, featureRepo featureRepository) *Service {
	return &Service{charRepo: charRepo, featureRepo: featureRepo}
}

func (s *Service) Purchase(ctx context.Context, charID int64, bagType int, amount int64) (*Result, error) {
	if s == nil || s.charRepo == nil || s.featureRepo == nil || charID <= 0 || bagType < 1 || bagType > 4 || amount <= 0 {
		return nil, ErrInvalidRequest
	}
	value, _ := s.locks.LoadOrStore(charID, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	state, err := s.Load(ctx, charID)
	if err != nil {
		return nil, err
	}
	current := state.Value(bagType)
	if amount > Capacity-current {
		return nil, ErrCapacityExceeded
	}
	cost := (amount + 4999) / 5000
	char, err := s.charRepo.FindByID(ctx, charID)
	if err != nil {
		return nil, err
	}
	if char == nil || char.Gold < cost {
		return nil, ErrInsufficientGold
	}

	previousGold := char.Gold
	newGold := previousGold - cost
	state.SetValue(bagType, current+amount)
	if atomicRepo, ok := s.featureRepo.(domainfeature.BloodBagPurchaseRepository); ok {
		if err := atomicRepo.CommitBloodBagPurchase(ctx, charID, newGold, state.Map()); err != nil {
			return nil, fmt.Errorf("blood bag: commit purchase: %w", err)
		}
		char.Gold = newGold
		if err := s.charRepo.Update(ctx, char); err != nil {
			return nil, fmt.Errorf("blood bag: sync character cache: %w", err)
		}
		return &Result{State: state, Cost: cost, NewGold: newGold}, nil
	}

	char.Gold = newGold
	if err := s.charRepo.Update(ctx, char); err != nil {
		return nil, fmt.Errorf("blood bag: deduct gold: %w", err)
	}
	if err := s.featureRepo.UpsertCharacterFeatureState(ctx, &domainfeature.CharacterFeatureState{
		CharacterID: charID,
		FeatureKey:  domainfeature.FeatureBloodBag,
		State:       state.Map(),
	}); err != nil {
		char.Gold = previousGold
		_ = s.charRepo.Update(ctx, char)
		return nil, fmt.Errorf("blood bag: persist reserve: %w", err)
	}

	return &Result{State: state, Cost: cost, NewGold: newGold}, nil
}

func (s *Service) Load(ctx context.Context, charID int64) (State, error) {
	states, err := s.featureRepo.ListCharacterFeatureStates(ctx, charID)
	if err != nil {
		return State{}, err
	}
	for _, feature := range states {
		if feature != nil && feature.FeatureKey == domainfeature.FeatureBloodBag {
			return StateFromMap(feature.State), nil
		}
	}
	return State{}, nil
}

func StateFromMap(values map[string]interface{}) State {
	return State{
		CharacterHP: mapInt64(values, "characterHp"),
		CharacterMP: mapInt64(values, "characterMp"),
		PetHP:       mapInt64(values, "petHp"),
		PetMP:       mapInt64(values, "petMp"),
	}
}

func (s State) Map() map[string]interface{} {
	return map[string]interface{}{
		"characterHp": s.CharacterHP,
		"characterMp": s.CharacterMP,
		"petHp":       s.PetHP,
		"petMp":       s.PetMP,
	}
}

func (s State) String() string {
	return fmt.Sprintf("|%d|%d|%d|%d", s.CharacterHP, s.CharacterMP, s.PetHP, s.PetMP)
}

func (s State) Value(bagType int) int64 {
	switch bagType {
	case 1:
		return s.CharacterHP
	case 2:
		return s.CharacterMP
	case 3:
		return s.PetHP
	case 4:
		return s.PetMP
	default:
		return 0
	}
}

func (s *State) SetValue(bagType int, value int64) {
	switch bagType {
	case 1:
		s.CharacterHP = value
	case 2:
		s.CharacterMP = value
	case 3:
		s.PetHP = value
	case 4:
		s.PetMP = value
	}
}

func mapInt64(values map[string]interface{}, key string) int64 {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case int:
		return bounded(int64(value))
	case int64:
		return bounded(value)
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return 0
		}
		return bounded(int64(value))
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err == nil {
			return bounded(parsed)
		}
	}
	return 0
}

func bounded(value int64) int64 {
	if value < 0 {
		return 0
	}
	if value > Capacity {
		return Capacity
	}
	return value
}
