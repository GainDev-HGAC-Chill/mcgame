// Safe administrative level transitions preserve assets and reject ambiguous allocation state.
package character

import "fmt"

func (c *Character) SetTargetLevel(target int) (bool, error) {
	if target < 1 || target > MaxLevel {
		return false, fmt.Errorf("cấp đích phải từ 1 đến %d", MaxLevel)
	}
	if c.Level < 1 || c.Level > MaxLevel {
		return false, fmt.Errorf("cấp hiện tại không hợp lệ; cần kiểm tra dữ liệu trước khi thay đổi")
	}
	if target == c.Level {
		return false, nil
	}
	levelPoints := (c.Level - 1) * AttributePointsPerLevel
	if c.MaxAttrPoints < levelPoints || c.DistributedAttrPoints < 0 || c.AttrPoints < 0 ||
		c.DistributedAttrPoints > c.MaxAttrPoints || c.AttrPoints != c.MaxAttrPoints-c.DistributedAttrPoints {
		return false, fmt.Errorf("dữ liệu điểm thuộc tính không hợp lệ; cần kiểm tra trước khi thay đổi cấp")
	}
	bonusPoints := c.MaxAttrPoints - levelPoints
	if target > c.Level {
		gained := (target - c.Level) * AttributePointsPerLevel
		if c.MaxAttrPoints > int(^uint(0)>>1)-gained {
			return false, fmt.Errorf("điểm thuộc tính vượt giới hạn lưu trữ")
		}
		return c.RaiseLevelTo(target), nil
	}
	allocated := c.DistributedAttrPoints
	if allocated < 0 {
		return false, fmt.Errorf("dữ liệu điểm thuộc tính không hợp lệ")
	}
	if allocated > 0 {
		remaining := allocated
		for _, points := range []int{c.AptStrength, c.AptAgility, c.AptStamina, c.AptIntelligence, c.AptEnergy} {
			if points < 0 || points > remaining {
				return false, fmt.Errorf("không thể tách điểm đã phân phối khỏi tư chất; cần kiểm tra dữ liệu trước khi hạ cấp")
			}
			remaining -= points
		}
		if remaining != 0 {
			return false, fmt.Errorf("không thể tách điểm đã phân phối khỏi tư chất; cần kiểm tra dữ liệu trước khi hạ cấp")
		}
	}
	delta := c.Level - target
	if c.Strength < delta || c.Agility < delta || c.Stamina < delta || c.Intelligence < delta || c.Spirit < delta {
		return false, fmt.Errorf("thuộc tính cơ bản không khớp cấp hiện tại; không thể hạ cấp an toàn")
	}
	c.Level = target
	c.Experience = 0
	c.Strength -= delta
	c.Agility -= delta
	c.Stamina -= delta
	c.Intelligence -= delta
	c.Spirit -= delta
	if allocated > 0 {
		c.AptStrength = 0
		c.AptAgility = 0
		c.AptStamina = 0
		c.AptIntelligence = 0
		c.AptEnergy = 0
	}
	c.DistributedAttrPoints = 0
	c.MaxAttrPoints = (target-1)*AttributePointsPerLevel + bonusPoints
	c.AttrPoints = c.MaxAttrPoints
	c.LastPoint = "0"
	c.RefreshRuntimeStats()
	return true, nil
}
