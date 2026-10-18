// Pet fight configuration validates and isolates mode-specific lineup snapshots.
package petarena

import (
	"context"
	"errors"
	"math"
	"strconv"

	domainpet "mcgame-server/internal/domain/pet"
	domainpetarena "mcgame-server/internal/domain/petarena"
	pkgerrors "mcgame-server/pkg/errors"
)

const (
	ConfigContextFarm   = "farm"
	ConfigContextHMTXLS = "hmtxls"
	ConfigContextPPVE   = "ppve"
	ConfigContextArena  = "arena"
)

const (
	petActionAttack  = -10
	petActionDefence = -20
	petActionSkill   = -50
)

func (s *Service) GetPetFightConfig(ctx context.Context, charID int64, configKey string) (map[string]interface{}, error) {
	if s.arenaRepo == nil {
		return map[string]interface{}{}, nil
	}
	if !validConfigContext(configKey) {
		return nil, pkgerrors.ErrInvalidInput
	}
	if configKey == ConfigContextArena {
		entry, err := s.arenaRepo.GetEntry(ctx, charID, s.config.Season)
		if err != nil {
			if errors.Is(err, pkgerrors.ErrNotFound) {
				return map[string]interface{}{}, nil
			}
			return nil, err
		}
		if entry == nil || entry.ConfData == nil {
			return map[string]interface{}{}, nil
		}
		return entry.ConfData, nil
	}

	config, err := s.arenaRepo.GetFightConfig(ctx, charID, configKey)
	if err != nil {
		if errors.Is(err, pkgerrors.ErrNotFound) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	if config == nil || config.ConfData == nil {
		return map[string]interface{}{}, nil
	}
	return config.ConfData, nil
}

func (s *Service) SavePetFightConfig(ctx context.Context, charID int64, configKey string, confData map[string]interface{}) error {
	if s.arenaRepo == nil || s.petRepo == nil {
		return pkgerrors.ErrSystemError
	}
	if configKey != ConfigContextFarm && configKey != ConfigContextHMTXLS {
		return pkgerrors.ErrInvalidInput
	}

	lineup, ok := stringMap(confData["conf1"])
	if !ok {
		return pkgerrors.ErrInvalidInput
	}
	if _, err := s.validateFightLineup(ctx, charID, lineup, 1, 5); err != nil {
		return err
	}

	config := &domainpetarena.FightConfig{
		CharacterID: charID,
		ConfigKey:   configKey,
		ConfData:    confData,
	}
	return s.arenaRepo.SaveFightConfig(ctx, config)
}

func (s *Service) RegisterArenaEntry(ctx context.Context, charID int64, teamName string, confData map[string]interface{}) (*domainpetarena.Entry, error) {
	if s.arenaRepo == nil || s.petRepo == nil {
		return nil, pkgerrors.ErrSystemError
	}
	petIDs, err := s.validateFightLineup(ctx, charID, confData, 2, 5)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{"conf1": confData}
	entry, err := s.arenaRepo.GetEntry(ctx, charID, s.config.Season)
	if err != nil && !errors.Is(err, pkgerrors.ErrNotFound) {
		return nil, err
	}
	if entry == nil {
		entry = &domainpetarena.Entry{
			CharacterID: charID,
			Season:      s.config.Season,
			MaxFights:   20,
			LastRank:    -1,
		}
	}
	entry.TeamName = teamName
	entry.ConfData = payload
	if entry.MaxFights <= 0 {
		entry.MaxFights = 20
	}
	if entry.LastRank == 0 {
		entry.LastRank = -1
	}
	if err := s.arenaRepo.SaveEntry(ctx, entry); err != nil {
		return nil, err
	}
	_ = s.SetBattlePet(ctx, charID, petIDs[0])
	return entry, nil
}

func (s *Service) arenaLeadPetID(ctx context.Context, charID int64) (int64, error) {
	entry, err := s.arenaRepo.GetEntry(ctx, charID, s.config.Season)
	if err != nil {
		return 0, err
	}
	if entry == nil {
		return 0, pkgerrors.ErrInvalidInput
	}
	lineup, ok := stringMap(entry.ConfData["conf1"])
	if !ok {
		return 0, pkgerrors.ErrInvalidInput
	}
	petIDs, err := s.validateFightLineup(ctx, charID, lineup, 2, 5)
	if err != nil {
		return 0, err
	}
	return petIDs[0], nil
}

func (s *Service) validateFightLineup(ctx context.Context, charID int64, lineup map[string]interface{}, minPets, maxPets int) ([]int64, error) {
	if len(lineup) < minPets || len(lineup) > maxPets {
		return nil, pkgerrors.ErrInvalidInput
	}

	pets := make(map[int64]*domainpet.Pet, len(lineup))
	positions := make(map[int]struct{}, len(lineup))
	ordered := make([]int64, 0, len(lineup))
	for position := 5; position <= 9; position++ {
		raw, exists := lineup[strconv.Itoa(position)]
		if !exists {
			continue
		}
		entry, ok := stringMap(raw)
		if !ok {
			return nil, pkgerrors.ErrInvalidInput
		}
		storedPosition, ok := exactInt(entry["pos"])
		if !ok || storedPosition != position {
			return nil, pkgerrors.ErrInvalidInput
		}
		if _, duplicate := positions[storedPosition]; duplicate {
			return nil, pkgerrors.ErrInvalidInput
		}
		positions[storedPosition] = struct{}{}

		petID, ok := exactInt64(entry["pid"])
		if !ok || petID <= 0 {
			return nil, pkgerrors.ErrInvalidInput
		}
		if _, duplicate := pets[petID]; duplicate {
			return nil, pkgerrors.ErrInvalidInput
		}
		ownedPet, err := s.petRepo.FindByID(ctx, petID)
		if err != nil {
			return nil, err
		}
		if ownedPet == nil || ownedPet.CharacterID != charID {
			return nil, pkgerrors.ErrNotFound
		}
		if err := validatePetCommands(ownedPet, entry["cmdList"]); err != nil {
			return nil, err
		}
		pets[petID] = ownedPet
		ordered = append(ordered, petID)
	}
	if len(ordered) != len(lineup) {
		return nil, pkgerrors.ErrInvalidInput
	}
	return ordered, nil
}

func validatePetCommands(p *domainpet.Pet, raw interface{}) error {
	commands, ok := interfaceSlice(raw)
	if !ok || len(commands) > 24 {
		return pkgerrors.ErrInvalidInput
	}
	learned := make(map[int]struct{})
	for _, skill := range p.LearnedSkillSlots() {
		learned[skill.ID] = struct{}{}
	}
	legacyLevels := make(map[int]int)
	for _, skill := range p.GetSkills() {
		if skill.ID > 0 && skill.Level > 0 {
			legacyLevels[skill.ID] = skill.Level
		}
	}

	for _, rawCommand := range commands {
		if rawCommand == nil {
			continue
		}
		command, ok := stringMap(rawCommand)
		if !ok {
			return pkgerrors.ErrInvalidInput
		}
		action, ok := exactInt(command["action"])
		if !ok {
			return pkgerrors.ErrInvalidInput
		}
		target, ok := exactInt(command["t"])
		if !ok || !validTargetCode(target) {
			return pkgerrors.ErrInvalidInput
		}
		switch action {
		case petActionAttack:
			if id, ok := exactInt(command["id"]); !ok || id != 1000 {
				return pkgerrors.ErrInvalidInput
			}
			if level, ok := exactInt(command["level"]); !ok || level != -1 {
				return pkgerrors.ErrInvalidInput
			}
		case petActionDefence:
			if id, ok := exactInt(command["id"]); !ok || id != 1001 {
				return pkgerrors.ErrInvalidInput
			}
			if level, ok := exactInt(command["level"]); !ok || level != -1 {
				return pkgerrors.ErrInvalidInput
			}
		case petActionSkill:
			skillID, ok := exactInt(command["id"])
			if !ok || skillID <= 0 {
				return pkgerrors.ErrInvalidInput
			}
			if _, ok := learned[skillID]; !ok {
				return pkgerrors.ErrInvalidInput
			}
			level, ok := exactInt(command["level"])
			if !ok || level <= 0 {
				return pkgerrors.ErrInvalidInput
			}
			if knownLevel, known := legacyLevels[skillID]; known && level != knownLevel {
				return pkgerrors.ErrInvalidInput
			}
		default:
			return pkgerrors.ErrInvalidInput
		}
	}
	return nil
}

func validConfigContext(configKey string) bool {
	switch configKey {
	case ConfigContextFarm, ConfigContextHMTXLS, ConfigContextPPVE, ConfigContextArena:
		return true
	default:
		return false
	}
}

func validTargetCode(target int) bool {
	return target >= 1 && target <= 4 || target >= 50 && target <= 54
}

func stringMap(raw interface{}) (map[string]interface{}, bool) {
	switch typed := raw.(type) {
	case map[string]interface{}:
		return typed, true
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(typed))
		for key, value := range typed {
			stringKey := toString(key)
			if stringKey == "" {
				return nil, false
			}
			result[stringKey] = value
		}
		return result, true
	default:
		return nil, false
	}
}

func interfaceSlice(raw interface{}) ([]interface{}, bool) {
	switch typed := raw.(type) {
	case []interface{}:
		return typed, true
	case nil:
		return []interface{}{}, true
	default:
		return nil, false
	}
}

func exactInt(raw interface{}) (int, bool) {
	value, ok := exactInt64(raw)
	if !ok || int64(int(value)) != value {
		return 0, false
	}
	return int(value), true
}

func exactInt64(raw interface{}) (int64, bool) {
	switch typed := raw.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || math.Trunc(typed) != typed || typed > math.MaxInt64 || typed < math.MinInt64 {
			return 0, false
		}
		return int64(typed), true
	case string:
		value, err := strconv.ParseInt(typed, 10, 64)
		return value, err == nil
	default:
		return 0, false
	}
}

func toString(raw interface{}) string {
	switch typed := raw.(type) {
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		if math.Trunc(typed) == typed {
			return strconv.FormatInt(int64(typed), 10)
		}
	}
	return ""
}
