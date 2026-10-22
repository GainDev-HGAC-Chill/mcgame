// Special training course state follows the TXKC Flash wire contract.
package specialtraining

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	domainchar "mcgame-server/internal/domain/character"
	domainfeature "mcgame-server/internal/domain/statfeature"
)

const (
	MaxLevel        = 70
	ExpPerLevel     = 500
	PremiumCost     = 680
	ExpPurchaseCost = 300
	DailyRewardExp  = 120
	WeeklyRewardExp = 280
)

var (
	ErrInactive          = errors.New("specialtraining: season inactive")
	ErrAlreadyPremium    = errors.New("specialtraining: premium already open")
	ErrInsufficientGold  = errors.New("specialtraining: insufficient gold")
	ErrInvalidCount      = errors.New("specialtraining: invalid purchase count")
	ErrAtMaxLevel        = errors.New("specialtraining: already at max level")
	ErrTaskUnavailable   = errors.New("specialtraining: task unavailable")
	ErrTaskIncomplete    = errors.New("specialtraining: task incomplete")
	ErrTaskClaimed       = errors.New("specialtraining: task already claimed")
	ErrRewardUnavailable = errors.New("specialtraining: reward unavailable")
)

type StateRepository interface {
	ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error)
	UpsertCharacterFeatureState(context.Context, *domainfeature.CharacterFeatureState) error
}

type CharacterRepository interface {
	FindByID(context.Context, int64) (*domainchar.Character, error)
	Update(context.Context, *domainchar.Character) error
}

type Config struct {
	Key      string
	Start    time.Time
	End      time.Time
	Location *time.Location
}

type TaskState struct {
	Progress int  `json:"progress"`
	Claimed  bool `json:"claimed"`
}

type State struct {
	Season      string               `json:"season"`
	Level       int                  `json:"level"`
	Exp         int                  `json:"exp"`
	Premium     bool                 `json:"premium"`
	FreeClaimed int                  `json:"freeClaimed"`
	PaidClaimed int                  `json:"paidClaimed"`
	DailyKey    string               `json:"dailyKey"`
	WeekKey     string               `json:"weekKey"`
	DailyOpen   bool                 `json:"dailyOpen"`
	Daily       map[string]TaskState `json:"daily"`
	Weekly      map[string]TaskState `json:"weekly"`
}

type MutationResult struct {
	Payload   map[string]interface{}
	Gold      int64
	GoldSpent int64
}

type Service struct {
	repo          StateRepository
	charRepo      CharacterRepository
	config        Config
	now           func() time.Time
	mutationLocks [64]sync.Mutex
}

var dailyRequirements = map[int]int{37: 3, 40: 1, 27: 1, 7: 1, 6: 1, 8: 1, 4: 3, 2: 3, 1: 3, 26: 1}
var weeklyRequirements = map[int]int{1: 5, 2: 30, 3: 30, 4: 30, 5: 50, 6: 1000}

func NewService(repo StateRepository, charRepo CharacterRepository, config Config) *Service {
	return &Service{repo: repo, charRepo: charRepo, config: config, now: time.Now}
}

func (s *Service) SetClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

func (s *Service) GetData(ctx context.Context, charID int64) (map[string]interface{}, error) {
	unlock := s.lock(charID)
	defer unlock()
	now, err := s.currentTime()
	if err != nil {
		return nil, err
	}
	state, changed, err := s.loadState(ctx, charID, now)
	if err != nil {
		return nil, err
	}
	if changed {
		if err := s.saveState(ctx, charID, state); err != nil {
			return nil, err
		}
	}
	return s.payload(state, now), nil
}

func (s *Service) OpenPremium(ctx context.Context, charID int64) (MutationResult, error) {
	return s.mutateGold(ctx, charID, PremiumCost, func(state *State) error {
		if state.Premium {
			return ErrAlreadyPremium
		}
		state.Premium = true
		return nil
	})
}

func (s *Service) BuyEXP(ctx context.Context, charID int64, count int) (MutationResult, error) {
	if count <= 0 || count > math.MaxInt/ExpPerLevel || count > math.MaxInt/ExpPurchaseCost {
		return MutationResult{}, ErrInvalidCount
	}
	cost := int64(count * ExpPurchaseCost)
	return s.mutateGold(ctx, charID, cost, func(state *State) error {
		if state.Level >= MaxLevel {
			return ErrAtMaxLevel
		}
		remaining := (MaxLevel-state.Level)*ExpPerLevel - state.Exp
		if count*ExpPerLevel > remaining {
			return ErrInvalidCount
		}
		s.addEXP(state, count*ExpPerLevel)
		return nil
	})
}

func (s *Service) OpenDailyTasks(ctx context.Context, charID int64) (map[string]interface{}, error) {
	return s.mutateState(ctx, charID, func(state *State) error {
		state.DailyOpen = true
		return nil
	})
}

func (s *Service) RecordTask(ctx context.Context, charID int64, taskType, taskID, amount int) error {
	if amount <= 0 {
		return ErrInvalidCount
	}
	_, err := s.mutateState(ctx, charID, func(state *State) error {
		tasks, requirements, ok := taskCollection(state, taskType)
		if !ok || (taskType == 1 && !state.DailyOpen) {
			return ErrTaskUnavailable
		}
		required, ok := requirements[taskID]
		if !ok {
			return ErrTaskUnavailable
		}
		key := strconv.Itoa(taskID)
		task := tasks[key]
		if !task.Claimed {
			task.Progress = min(required, task.Progress+amount)
			tasks[key] = task
		}
		return nil
	})
	return err
}

func (s *Service) ClaimTaskEXP(ctx context.Context, charID int64, taskType, taskID int) (map[string]interface{}, error) {
	return s.mutateState(ctx, charID, func(state *State) error {
		tasks, requirements, ok := taskCollection(state, taskType)
		if !ok || (taskType == 1 && !state.DailyOpen) {
			return ErrTaskUnavailable
		}
		required, ok := requirements[taskID]
		if !ok {
			return ErrTaskUnavailable
		}
		key := strconv.Itoa(taskID)
		task := tasks[key]
		if task.Claimed {
			return ErrTaskClaimed
		}
		if task.Progress < required {
			return ErrTaskIncomplete
		}
		task.Claimed = true
		tasks[key] = task
		if taskType == 1 {
			s.addEXP(state, DailyRewardExp)
		} else {
			s.addEXP(state, WeeklyRewardExp)
		}
		return nil
	})
}

func (s *Service) ReserveNextReward(ctx context.Context, charID int64, premium bool) (Reward, int, map[string]interface{}, error) {
	var reward Reward
	reservedLevel := 0
	payload, err := s.mutateState(ctx, charID, func(state *State) error {
		claimed := &state.FreeClaimed
		rewards := FreeRewards
		if premium {
			if !state.Premium {
				return ErrRewardUnavailable
			}
			claimed = &state.PaidClaimed
			rewards = PremiumRewards
		}
		next := *claimed + 1
		if next > state.Level || next > MaxLevel {
			return ErrRewardUnavailable
		}
		reward = rewards[next]
		if reward.ItemID <= 0 || reward.Count <= 0 {
			return ErrRewardUnavailable
		}
		*claimed = next
		reservedLevel = next
		return nil
	})
	return reward, reservedLevel, payload, err
}

func (s *Service) RollbackReward(ctx context.Context, charID int64, premium bool, level int) error {
	_, err := s.mutateState(ctx, charID, func(state *State) error {
		claimed := &state.FreeClaimed
		if premium {
			claimed = &state.PaidClaimed
		}
		if *claimed == level {
			*claimed--
		}
		return nil
	})
	return err
}

func (s *Service) mutateGold(ctx context.Context, charID, cost int64, mutation func(*State) error) (MutationResult, error) {
	unlock := s.lock(charID)
	defer unlock()
	now, err := s.currentTime()
	if err != nil {
		return MutationResult{}, err
	}
	state, _, err := s.loadState(ctx, charID, now)
	if err != nil {
		return MutationResult{}, err
	}
	if err := mutation(state); err != nil {
		return MutationResult{}, err
	}
	char, err := s.charRepo.FindByID(ctx, charID)
	if err != nil {
		return MutationResult{}, fmt.Errorf("specialtraining: load character: %w", err)
	}
	if char == nil {
		return MutationResult{}, errors.New("specialtraining: character not found")
	}
	if char.Gold < cost {
		return MutationResult{}, ErrInsufficientGold
	}
	oldGold := char.Gold
	char.Gold -= cost
	if err := s.charRepo.Update(ctx, char); err != nil {
		char.Gold = oldGold
		return MutationResult{}, err
	}
	if err := s.saveState(ctx, charID, state); err != nil {
		char.Gold = oldGold
		if rollbackErr := s.charRepo.Update(ctx, char); rollbackErr != nil {
			return MutationResult{}, fmt.Errorf("%w; rollback gold: %v", err, rollbackErr)
		}
		return MutationResult{}, err
	}
	return MutationResult{Payload: s.payload(state, now), Gold: char.Gold, GoldSpent: cost}, nil
}

func (s *Service) mutateState(ctx context.Context, charID int64, mutation func(*State) error) (map[string]interface{}, error) {
	unlock := s.lock(charID)
	defer unlock()
	now, err := s.currentTime()
	if err != nil {
		return nil, err
	}
	state, _, err := s.loadState(ctx, charID, now)
	if err != nil {
		return nil, err
	}
	if err := mutation(state); err != nil {
		return nil, err
	}
	if err := s.saveState(ctx, charID, state); err != nil {
		return nil, err
	}
	return s.payload(state, now), nil
}

func (s *Service) lock(charID int64) func() {
	lock := &s.mutationLocks[uint64(charID)%uint64(len(s.mutationLocks))]
	lock.Lock()
	return lock.Unlock
}
