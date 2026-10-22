package specialtraining

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	domainfeature "mcgame-server/internal/domain/statfeature"
)

func (s *Service) loadState(ctx context.Context, charID int64, now time.Time) (*State, bool, error) {
	state := defaultState(s.config.Key, now)
	found := false
	rows, err := s.repo.ListCharacterFeatureStates(ctx, charID)
	if err != nil {
		return nil, false, err
	}
	for _, row := range rows {
		if row != nil && row.FeatureKey == domainfeature.FeatureSpecialTraining {
			state = stateFromMap(row.State)
			found = true
			break
		}
	}
	changed := !found
	if state.Season != s.config.Key {
		state = defaultState(s.config.Key, now)
		changed = true
	}
	dayKey := now.Format("2006-01-02")
	weekKey := startOfWeek(now).Format("2006-01-02")
	if state.DailyKey != dayKey {
		state.DailyKey = dayKey
		state.DailyOpen = false
		state.Daily = make(map[string]TaskState)
		changed = true
	}
	if state.WeekKey != weekKey {
		state.WeekKey = weekKey
		state.Weekly = make(map[string]TaskState)
		changed = true
	}
	return state, changed, nil
}

func (s *Service) currentTime() (time.Time, error) {
	if s.config.Location == nil || s.config.Key == "" || !s.config.End.After(s.config.Start) {
		return time.Time{}, ErrInactive
	}
	now := s.now().In(s.config.Location)
	if now.Before(s.config.Start) || !now.Before(s.config.End) {
		return time.Time{}, ErrInactive
	}
	return now, nil
}

func (s *Service) saveState(ctx context.Context, charID int64, state *State) error {
	return s.repo.UpsertCharacterFeatureState(ctx, &domainfeature.CharacterFeatureState{CharacterID: charID, FeatureKey: domainfeature.FeatureSpecialTraining, State: stateToMap(state)})
}

func (s *Service) payload(state *State, now time.Time) map[string]interface{} {
	days := int(math.Ceil(s.config.End.Sub(now).Hours() / 24))
	flag := map[string]interface{}{
		"lev": state.Level, "exp": state.Exp, "p": boolInt(state.Premium), "awd1": state.FreeClaimed, "awd2": state.PaidClaimed,
		"wd": taskWire(state.Weekly, weeklyRequirements),
	}
	if state.DailyOpen {
		flag["dd"] = taskWire(state.Daily, dailyRequirements)
		flag["spIndex"] = -1
	}
	return map[string]interface{}{
		"start": s.config.Start.In(s.config.Location).Format("2006-01-02"),
		"end":   s.config.End.In(s.config.Location).Format("2006-01-02"),
		"days":  days,
		"flag":  flag,
	}
}

func (s *Service) addEXP(state *State, amount int) {
	if state.Level >= MaxLevel {
		state.Exp = 0
		return
	}
	state.Exp += amount
	for state.Exp >= ExpPerLevel && state.Level < MaxLevel {
		state.Exp -= ExpPerLevel
		state.Level++
	}
	if state.Level >= MaxLevel {
		state.Exp = 0
	}
}

func defaultState(season string, now time.Time) *State {
	return &State{Season: season, Level: 1, DailyKey: now.Format("2006-01-02"), WeekKey: startOfWeek(now).Format("2006-01-02"), Daily: make(map[string]TaskState), Weekly: make(map[string]TaskState)}
}

func taskCollection(state *State, taskType int) (map[string]TaskState, map[int]int, bool) {
	switch taskType {
	case 1:
		return state.Daily, dailyRequirements, true
	case 2:
		return state.Weekly, weeklyRequirements, true
	default:
		return nil, nil, false
	}
}

func taskWire(tasks map[string]TaskState, requirements map[int]int) map[string]interface{} {
	result := make(map[string]interface{}, len(requirements))
	for id, required := range requirements {
		key := strconv.Itoa(id)
		task := tasks[key]
		switch {
		case task.Claimed:
			result[key] = "exped"
		case task.Progress >= required:
			result[key] = "true"
		default:
			result[key] = fmt.Sprintf("%d/%d", task.Progress, required)
		}
	}
	return result
}

func startOfWeek(now time.Time) time.Time {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	delta := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -delta)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
