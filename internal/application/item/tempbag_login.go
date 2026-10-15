// Open-sourced by BaoLT

// Temporary-bag (Túi Tạm) payload builder shared by the getTBag RPC and the
// onChooseCharactor login snapshot so both emit identical tBag data.
package item

import (
	"strconv"
	"time"

	domainchar "mcgame-server/internal/domain/character"
	domainitem "mcgame-server/internal/domain/item"
)

func (s *Service) BuildTBagPayload(char *domainchar.Character, items []*domainitem.Item) map[string]interface{} {
	tempList := make(map[string]interface{})
	mxTempList := make(map[string]interface{})
	installedBags := make(map[string]interface{})
	normalItemCount := 0
	mxItemCount := 0

	for _, it := range items {
		if it == nil || it.SlotType != domainitem.SlotTypeTempBag {
			continue
		}
		if slot, installed := tempBagInstalledSlot(it.SlotIndex); installed {
			expiresAt, timed := tempBagExpiry(it)
			installedBags[strconv.Itoa(slot)] = map[string]interface{}{
				"tid": it.TemplateID,
				"ot":  timed && !time.Now().Before(expiresAt),
			}
			continue
		}
		if it.SlotIndex >= 100 {
			mxTempList[strconv.Itoa(it.SlotIndex-100)] = tempBagItemPayload(it)
			mxItemCount++
		} else {
			tempList[strconv.Itoa(it.SlotIndex)] = tempBagItemPayload(it)
			normalItemCount++
		}
	}

	normalCount := char.TempBagSlots
	mxCount := char.MxTempBagSlots

	tempbagNum := min(max(normalCount, 0), 5)
	mxTempbagNum := min(max(mxCount, 0), 5)

	slotCapacity := []int{6, 6, 12, 12, 12}
	extraSlots := s.tempBagCapacity(items, time.Now())
	if len(installedBags) == 0 {
		for i := 0; i < normalCount && i < 5; i++ {
			extraSlots += slotCapacity[i]
		}
	}
	mxExtraSlots := 0
	for i := 0; i < mxCount && i < 5; i++ {
		mxExtraSlots += slotCapacity[i]
	}
	dire := 0

	if char != nil && char.PMProcessData != nil {
		if enabled, ok := char.PMProcessData["tempBagDire"].(bool); ok && enabled {
			dire = 1
		}
	}
	return map[string]interface{}{
		"tempList":    tempList,
		"tempBag":     installedBags,
		"tempSlotNum": extraSlots,
		"tempbagNum":  tempbagNum,
		"curNum":      normalItemCount,
		"ot":          0,
		"dire":        dire,
		"mx": map[string]interface{}{
			"tempSlotNum": 12 + mxExtraSlots,
			"tempbagNum":  mxTempbagNum,
			"curNum":      mxItemCount,
			"tempBag":     map[string]interface{}{},
			"tempList":    mxTempList,
		},
	}
}

func tempBagItemPayload(it *domainitem.Item) map[string]interface{} {
	binded := 0
	if it.IsBound {
		binded = 1
	}
	return map[string]interface{}{
		"t": it.TemplateID,
		"n": it.StackCount,
		"b": binded,
		"c": it.ColorCode,
	}
}
