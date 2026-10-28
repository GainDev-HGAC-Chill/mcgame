// Guild rank settings are the shared wire and authorization authority.
package guild

import "fmt"

type Permission string

const (
	PermissionAdd   Permission = "canAdd"
	PermissionQuest Permission = "canQuest"
	PermissionSlot  Permission = "canSlot"
	PermissionInfo  Permission = "canInfo"
	PermissionDel   Permission = "canDel"
	PermissionDuty  Permission = "canDuty"
)

type RankSetting struct {
	Rank     int    `json:"rank"`
	Name     string `json:"name"`
	CanAdd   bool   `json:"canAdd"`
	CanQuest bool   `json:"canQuest"`
	CanSlot  bool   `json:"canSlot"`
	CanInfo  bool   `json:"canInfo"`
	CanDel   bool   `json:"canDel"`
	CanDuty  bool   `json:"canDuty"`
}

func (s RankSetting) Validate() error {
	if s.Rank < RankLeader || s.Rank > RankMember {
		return fmt.Errorf("invalid guild rank: %d", s.Rank)
	}
	if s.Name == "" {
		return fmt.Errorf("guild rank %d has empty name", s.Rank)
	}
	if s.Rank == RankLeader && !(s.CanAdd && s.CanQuest && s.CanSlot && s.CanInfo && s.CanDel && s.CanDuty) {
		return fmt.Errorf("guild leader permissions cannot be disabled")
	}
	return nil
}

func (s RankSetting) Allows(permission Permission) bool {
	switch permission {
	case PermissionAdd:
		return s.Rank == RankLeader || s.CanAdd
	case PermissionQuest:
		return s.Rank == RankLeader || s.CanQuest
	case PermissionSlot:
		return s.Rank == RankLeader || s.CanSlot
	case PermissionInfo:
		return s.Rank == RankLeader || s.CanInfo
	case PermissionDel:
		return s.Rank == RankLeader || s.CanDel
	case PermissionDuty:
		return s.Rank == RankLeader || s.CanDuty
	default:
		return false
	}
}

func (s RankSetting) ToDTO() map[string]interface{} {
	return map[string]interface{}{
		"rank":     s.Rank,
		"name":     s.Name,
		"canAdd":   boolInt(s.CanAdd),
		"canQuest": boolInt(s.CanQuest),
		"canSlot":  boolInt(s.CanSlot),
		"canInfo":  boolInt(s.CanInfo),
		"canDel":   boolInt(s.CanDel),
		"canDuty":  boolInt(s.CanDuty),
	}
}

func RankSettingsDTO(settings []RankSetting) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(settings))
	for _, setting := range settings {
		result = append(result, setting.ToDTO())
	}
	return result
}

func RankSettingsFromDTO(values []map[string]interface{}) ([]RankSetting, error) {
	settings := make([]RankSetting, 0, len(values))
	for _, value := range values {
		rank, ok := integerValue(value["rank"])
		if !ok {
			return nil, fmt.Errorf("invalid guild rank value")
		}
		name, ok := value["name"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid guild rank %d name", rank)
		}
		canAdd, ok := booleanValue(value["canAdd"])
		if !ok {
			return nil, fmt.Errorf("invalid guild rank %d canAdd", rank)
		}
		canQuest, ok := booleanValue(value["canQuest"])
		if !ok {
			return nil, fmt.Errorf("invalid guild rank %d canQuest", rank)
		}
		canSlot, ok := booleanValue(value["canSlot"])
		if !ok {
			return nil, fmt.Errorf("invalid guild rank %d canSlot", rank)
		}
		canInfo, ok := booleanValue(value["canInfo"])
		if !ok {
			return nil, fmt.Errorf("invalid guild rank %d canInfo", rank)
		}
		canDel, ok := booleanValue(value["canDel"])
		if !ok {
			return nil, fmt.Errorf("invalid guild rank %d canDel", rank)
		}
		canDuty, ok := booleanValue(value["canDuty"])
		if !ok {
			return nil, fmt.Errorf("invalid guild rank %d canDuty", rank)
		}
		settings = append(settings, RankSetting{Rank: rank, Name: name, CanAdd: canAdd, CanQuest: canQuest, CanSlot: canSlot, CanInfo: canInfo, CanDel: canDel, CanDuty: canDuty})
	}
	if err := ValidateRankSettings(settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func ValidateRankSettings(settings []RankSetting) error {
	if len(settings) != RankMember {
		return fmt.Errorf("guild rank setting count: %d", len(settings))
	}
	seen := make(map[int]struct{}, len(settings))
	for _, setting := range settings {
		if err := setting.Validate(); err != nil {
			return err
		}
		if _, exists := seen[setting.Rank]; exists {
			return fmt.Errorf("duplicate guild rank: %d", setting.Rank)
		}
		seen[setting.Rank] = struct{}{}
	}
	for rank := RankLeader; rank <= RankMember; rank++ {
		if _, exists := seen[rank]; !exists {
			return fmt.Errorf("missing guild rank: %d", rank)
		}
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func integerValue(value interface{}) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	default:
		return 0, false
	}
}

func booleanValue(value interface{}) (bool, bool) {
	if typed, ok := value.(bool); ok {
		return typed, true
	}
	integer, ok := integerValue(value)
	if !ok || (integer != 0 && integer != 1) {
		return false, false
	}
	return integer == 1, true
}
