// Guild rank request decoding keeps AMF numeric values strict and typed.
package guild

import (
	"strconv"

	appguild "mcgame-server/internal/application/guild"
	domainguild "mcgame-server/internal/domain/guild"
	pkgerrors "mcgame-server/pkg/errors"
)

func parseGuildRankSettings(values map[string]interface{}) ([]domainguild.RankSetting, error) {
	settings := appguild.DefaultRankSettings()
	for index := range settings {
		setting := &settings[index]
		suffix := strconv.Itoa(setting.Rank)
		name, ok := values["name"+suffix].(string)
		if !ok || name == "" {
			return nil, pkgerrors.ErrInvalidInput
		}
		setting.Name = name
		if setting.Rank == domainguild.RankLeader {
			continue
		}
		var err error
		if setting.CanAdd, err = parseGuildPermission(values["canAdd"+suffix]); err != nil {
			return nil, err
		}
		if setting.CanQuest, err = parseGuildPermission(values["canQuest"+suffix]); err != nil {
			return nil, err
		}
		if setting.CanSlot, err = parseGuildPermission(values["canSlot"+suffix]); err != nil {
			return nil, err
		}
		if setting.CanInfo, err = parseGuildPermission(values["canInfo"+suffix]); err != nil {
			return nil, err
		}
		if setting.CanDel, err = parseGuildPermission(values["canDel"+suffix]); err != nil {
			return nil, err
		}
		if setting.CanDuty, err = parseGuildPermission(values["canDuty"+suffix]); err != nil {
			return nil, err
		}
	}
	if err := domainguild.ValidateRankSettings(settings); err != nil {
		return nil, pkgerrors.ErrInvalidInput
	}
	return settings, nil
}

func parseGuildPermission(value interface{}) (bool, error) {
	if number, ok := value.(float64); ok && number != 0 && number != 1 {
		return false, pkgerrors.ErrInvalidInput
	}
	parsed, err := parseIntArg(value)
	if err != nil || (parsed != 0 && parsed != 1) {
		return false, pkgerrors.ErrInvalidInput
	}
	return parsed == 1, nil
}
