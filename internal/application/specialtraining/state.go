package specialtraining

func stateToMap(state *State) map[string]interface{} {
	return map[string]interface{}{
		"season": state.Season, "level": state.Level, "exp": state.Exp, "premium": state.Premium,
		"freeClaimed": state.FreeClaimed, "paidClaimed": state.PaidClaimed,
		"dailyKey": state.DailyKey, "weekKey": state.WeekKey, "dailyOpen": state.DailyOpen,
		"daily": taskMapToWire(state.Daily), "weekly": taskMapToWire(state.Weekly),
	}
}

func stateFromMap(raw map[string]interface{}) *State {
	state := &State{
		Season: stringValue(raw["season"]), Level: intValue(raw["level"]), Exp: intValue(raw["exp"]), Premium: boolValue(raw["premium"]),
		FreeClaimed: intValue(raw["freeClaimed"]), PaidClaimed: intValue(raw["paidClaimed"]),
		DailyKey: stringValue(raw["dailyKey"]), WeekKey: stringValue(raw["weekKey"]), DailyOpen: boolValue(raw["dailyOpen"]),
		Daily: taskMapFromWire(raw["daily"]), Weekly: taskMapFromWire(raw["weekly"]),
	}
	if state.Level < 1 || state.Level > MaxLevel {
		state.Level = 1
	}
	if state.Exp < 0 || state.Exp >= ExpPerLevel {
		state.Exp = 0
	}
	if state.FreeClaimed < 0 || state.FreeClaimed > state.Level {
		state.FreeClaimed = 0
	}
	if state.PaidClaimed < 0 || state.PaidClaimed > state.Level {
		state.PaidClaimed = 0
	}
	return state
}

func taskMapToWire(tasks map[string]TaskState) map[string]interface{} {
	result := make(map[string]interface{}, len(tasks))
	for key, task := range tasks {
		result[key] = map[string]interface{}{"progress": task.Progress, "claimed": task.Claimed}
	}
	return result
}

func taskMapFromWire(value interface{}) map[string]TaskState {
	result := make(map[string]TaskState)
	raw, ok := value.(map[string]interface{})
	if !ok {
		return result
	}
	for key, entryValue := range raw {
		entry, ok := entryValue.(map[string]interface{})
		if !ok {
			continue
		}
		result[key] = TaskState{Progress: intValue(entry["progress"]), Claimed: boolValue(entry["claimed"])}
	}
	return result
}

func intValue(value interface{}) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func stringValue(value interface{}) string {
	result, _ := value.(string)
	return result
}

func boolValue(value interface{}) bool {
	result, _ := value.(bool)
	return result
}
