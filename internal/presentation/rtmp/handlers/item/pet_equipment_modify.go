// Open-sourced by BaoLT

package item

import (
	"math"
	"math/rand"
	"strconv"
	"strings"

	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

const (
	petEquipmentLevelMaterialID = 2333
	petEquipmentBindMaterialID  = 2809
	petEquipmentBindMoneyCost   = int64(30000)
)

var petEquipmentLevelCosts = map[int]int{
	40: 5, 60: 10, 80: 15, 100: 20, 120: 25, 140: 50, 160: 50,
}

var petEquipmentBindCosts = map[int]int{
	40: 1, 80: 2, 100: 4, 120: 8, 140: 16, 160: 24,
}

type petEquipmentColorCost struct {
	materialID int
	quantity   int
	rate       int
}

var petEquipmentColorCosts = map[int]petEquipmentColorCost{
	0: {materialID: 2334, quantity: 1, rate: 80},
	1: {materialID: 2334, quantity: 2, rate: 50},
	2: {materialID: 2334, quantity: 4, rate: 30},
	3: {materialID: 2335, quantity: 8, rate: 20},
}

type pendingPetEquipmentBind struct {
	equipmentID int64
	bindNum1    int
	bindNum2    int
}

func defaultPetEquipmentRoll(maxValue int) int {
	if maxValue <= 0 {
		return 0
	}
	return rand.Intn(maxValue) + 1
}

func (h *Handler) PetEquLevelUp(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, equipment, material, template, ok := h.petEquipmentRequest(ctx, args)
	if !ok {
		return petEquipmentLevelFailure(0), nil
	}
	quantity, ok := petEquipmentLevelCosts[int(template.ReqLevel)]
	if !ok {
		return petEquipmentLevelFailure(material.StackCount), nil
	}
	quantity = petEquipmentSpecialQuantity(template, quantity, 2, 1.5)
	if material.TemplateID != petEquipmentLevelMaterialID || material.SlotType != domainitem.SlotTypeBag || material.StackCount < quantity {
		return petEquipmentLevelFailure(material.StackCount), nil
	}

	nextTemplate := h.itemService.GetEquipmentTemplate(int(template.NextEquTid))
	if !compatiblePetEquipmentUpgrade(template, nextTemplate) {
		return petEquipmentLevelFailure(material.StackCount), nil
	}
	cost := petEquipmentMoneyCost(nextTemplate)
	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil || char == nil || !charHasMoney(char, cost) {
		return petEquipmentLevelFailure(material.StackCount), nil
	}

	consumed, deleted, err := h.itemService.ConsumeItemStack(ctx.Context, characterID, material.ID, quantity)
	if err != nil {
		return petEquipmentLevelFailure(material.StackCount), nil
	}
	h.sendPetEquipmentMaterialChange(ctx, consumed, deleted)
	charDeductMoney(char, cost)
	if err := h.charService.Update(ctx.Context, char); err != nil {
		return petEquipmentLevelFailure(consumed.StackCount), nil
	}
	_ = h.sendCallback(ctx.Connection, "onUPP", charMoneyUpdatePayload(char))

	updated, err := h.itemService.ReplacePetEquipmentTemplate(ctx.Context, characterID, equipment.ID, int(nextTemplate.ID))
	if err != nil {
		return petEquipmentLevelFailure(consumed.StackCount), nil
	}
	h.refreshPetEquipmentOwner(ctx, characterID, updated)

	return map[string]interface{}{
		"flag": true,
		"num":  consumed.StackCount,
		"i":    updated.ID,
		"tid":  updated.TemplateID,
		"sid":  updated.CalculateSID(),
		"next": int(nextTemplate.NextEquTid),
	}, nil
}

func (h *Handler) PetEquModColor(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, equipment, material, template, ok := h.petEquipmentRequest(ctx, args)
	if !ok {
		return petEquipmentColorFailure(0), nil
	}
	displayColor := petEquipmentDisplayColor(equipment)
	colorCost, ok := petEquipmentColorCosts[displayColor]
	if !ok {
		return petEquipmentColorFailure(0), nil
	}
	quantity := petEquipmentSpecialQuantity(template, colorCost.quantity, 4, 2)
	if material.TemplateID != colorCost.materialID || material.SlotType != domainitem.SlotTypeBag || material.StackCount < quantity {
		return petEquipmentColorFailure(0), nil
	}
	cost := petEquipmentMoneyCost(template)
	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil || char == nil || !charHasMoney(char, cost) {
		return petEquipmentColorFailure(0), nil
	}

	_, _, err = h.itemService.ConsumeItemStack(ctx.Context, characterID, material.ID, quantity)
	if err != nil {
		return petEquipmentColorFailure(0), nil
	}
	charDeductMoney(char, cost)
	if err := h.charService.Update(ctx.Context, char); err != nil {
		return petEquipmentColorFailure(quantity), nil
	}
	_ = h.sendCallback(ctx.Connection, "onUPP", charMoneyUpdatePayload(char))

	if h.petEquipmentRoll(100) > colorCost.rate {
		return petEquipmentColorFailure(quantity), nil
	}
	updated, err := h.itemService.SetPetEquipmentDisplayColor(ctx.Context, characterID, equipment.ID, displayColor+1)
	if err != nil {
		return petEquipmentColorFailure(quantity), nil
	}
	h.refreshPetEquipmentOwner(ctx, characterID, updated)

	return map[string]interface{}{
		"flag": true,
		"num":  quantity,
		"i":    updated.ID,
		"c":    displayColor + 1,
		"sid":  updated.CalculateSID(),
	}, nil
}

func (h *Handler) PetEquModBind(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	characterID, equipment, material, template, ok := h.petEquipmentRequest(ctx, args)
	if !ok {
		return map[string]interface{}{"f": false}, nil
	}
	quantity, ok := petEquipmentBindCosts[int(template.ReqLevel)]
	if !ok {
		return map[string]interface{}{"f": false}, nil
	}
	quantity = petEquipmentSpecialQuantity(template, quantity, 4, 2)
	if material.TemplateID != petEquipmentBindMaterialID || material.SlotType != domainitem.SlotTypeBag || material.StackCount < quantity {
		return map[string]interface{}{"f": false}, nil
	}
	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil || char == nil || !charHasMoney(char, petEquipmentBindMoneyCost) {
		return map[string]interface{}{"f": false}, nil
	}

	bindMax := int(template.BindPropNum)
	if bindMax <= 0 {
		return map[string]interface{}{"f": false}, nil
	}
	newBind1 := h.petEquipmentRoll(bindMax)
	newBind2 := h.petEquipmentRoll(bindMax)
	consumed, deleted, err := h.itemService.ConsumeItemStack(ctx.Context, characterID, material.ID, quantity)
	if err != nil {
		return map[string]interface{}{"f": false}, nil
	}
	h.sendPetEquipmentMaterialChange(ctx, consumed, deleted)
	charDeductMoney(char, petEquipmentBindMoneyCost)
	if err := h.charService.Update(ctx.Context, char); err != nil {
		return map[string]interface{}{"f": false}, nil
	}
	_ = h.sendCallback(ctx.Connection, "onUPP", charMoneyUpdatePayload(char))

	h.setPendingPetEquipmentBind(characterID, &pendingPetEquipmentBind{
		equipmentID: equipment.ID,
		bindNum1:    newBind1,
		bindNum2:    newBind2,
	})

	return map[string]interface{}{
		"f":         true,
		"ii":        material.ID,
		"n":         consumed.StackCount,
		"b1":        petEquipmentPropertyInt(equipment.Properties, "bindMainPropNum1"),
		"b11":       newBind1,
		"b2":        petEquipmentPropertyInt(equipment.Properties, "bindMainPropNum2"),
		"b12":       newBind2,
		"mainProp1": int(template.MainProp1),
		"mainProp2": int(template.MainProp2),
	}, nil
}

func (h *Handler) SurePetEquModBind(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, pkgerrors.ErrInvalidArgs
	}
	decision, ok := parseFlexibleInt64(args[0])
	if !ok {
		return nil, pkgerrors.ErrInvalidArgs
	}
	characterID, err := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if err != nil {
		return nil, pkgerrors.ErrUnauthorized
	}
	pending := h.takePendingPetEquipmentBind(characterID)
	if pending == nil || decision != 1 {
		return map[string]interface{}{"f": "false"}, nil
	}

	equipment, err := h.itemService.GetItemByID(ctx.Context, characterID, pending.equipmentID)
	if err != nil || equipment == nil || !isPetEquipmentTemplateForModify(h.itemService.GetEquipmentTemplate(equipment.TemplateID)) {
		return map[string]interface{}{"f": "false"}, nil
	}
	updated, err := h.itemService.SetPetEquipmentBindProps(ctx.Context, characterID, pending.equipmentID, pending.bindNum1, pending.bindNum2)
	if err != nil {
		return map[string]interface{}{"f": "false"}, nil
	}
	h.refreshPetEquipmentOwner(ctx, characterID, updated)

	return map[string]interface{}{"f": "sure", "b1": pending.bindNum1, "b2": pending.bindNum2}, nil
}

func (h *Handler) petEquipmentRequest(ctx *rtmp.RPCContext, args []interface{}) (int64, *domainitem.Item, *domainitem.Item, *models.EquiptTemplateTemplate, bool) {
	if len(args) < 2 || h.itemService == nil || h.charService == nil {
		return 0, nil, nil, nil, false
	}
	equipmentID, equipmentOK := parseFlexibleInt64(args[0])
	materialID, materialOK := parseFlexibleInt64(args[1])
	characterID, characterErr := strconv.ParseInt(ctx.CharacterID, 10, 64)
	if !equipmentOK || !materialOK || characterErr != nil || equipmentID <= 0 || materialID <= 0 || equipmentID == materialID {
		return 0, nil, nil, nil, false
	}
	equipment, err := h.itemService.GetItemByID(ctx.Context, characterID, equipmentID)
	if err != nil || equipment == nil || equipment.ItemType != domainitem.ItemTypeEquipment {
		return 0, nil, nil, nil, false
	}
	template := h.itemService.GetEquipmentTemplate(equipment.TemplateID)
	if !isPetEquipmentTemplateForModify(template) {
		return 0, nil, nil, nil, false
	}
	material, err := h.itemService.GetItemByID(ctx.Context, characterID, materialID)
	if err != nil || material == nil {
		return 0, nil, nil, nil, false
	}
	return characterID, equipment, material, template, true
}

func isPetEquipmentTemplateForModify(template *models.EquiptTemplateTemplate) bool {
	if template == nil {
		return false
	}
	position := int(template.Position)
	return int(template.Kind) == 9 && int(template.UseType) == 2 && position >= 50 && position <= 57
}

func compatiblePetEquipmentUpgrade(current *models.EquiptTemplateTemplate, next *models.EquiptTemplateTemplate) bool {
	if !isPetEquipmentTemplateForModify(current) || !isPetEquipmentTemplateForModify(next) {
		return false
	}
	return int(current.NextEquTid) == int(next.ID) &&
		current.Position == next.Position &&
		current.Type == next.Type &&
		next.ReqLevel > current.ReqLevel &&
		current.ReqClass == next.ReqClass &&
		current.ReqClassID == next.ReqClassID
}

func petEquipmentSpecialQuantity(template *models.EquiptTemplateTemplate, quantity int, wrappedMultiplier int, otherMultiplier float64) int {
	if template == nil || int(template.Position) < 56 || int(template.Position) > 57 {
		return quantity
	}
	requiredClassID := template.ReqClassID
	if strings.HasPrefix(requiredClassID, "|") && strings.HasSuffix(requiredClassID, "|") {
		return quantity * wrappedMultiplier
	}
	return int(math.Round(float64(quantity) * otherMultiplier))
}

func petEquipmentMoneyCost(template *models.EquiptTemplateTemplate) int64 {
	if template == nil {
		return 0
	}
	level := int64(template.ReqLevel)
	return level * level * 10
}

func petEquipmentDisplayColor(equipment *domainitem.Item) int {
	if equipment == nil {
		return -1
	}
	if equipment.Properties != nil {
		if value, exists := equipment.Properties["color"]; exists {
			if parsed, ok := parseFlexibleInt64(value); ok && parsed >= 0 && parsed <= 4 {
				color := int(parsed)
				return color
			}
		}
	}
	return domainitem.EquipmentDisplayColorFromColorCode(equipment.ColorCode)
}

func petEquipmentPropertyInt(properties map[string]interface{}, key string) int {
	if properties == nil {
		return 0
	}
	value, ok := parseFlexibleInt64(properties[key])
	if !ok {
		return 0
	}
	return int(value)
}

func petEquipmentLevelFailure(remaining int) map[string]interface{} {
	return map[string]interface{}{"flag": false, "num": remaining}
}

func petEquipmentColorFailure(consumed int) map[string]interface{} {
	return map[string]interface{}{"flag": false, "num": consumed}
}

func (h *Handler) sendPetEquipmentMaterialChange(ctx *rtmp.RPCContext, consumed *domainitem.Item, deleted bool) {
	if consumed == nil {
		return
	}
	if deleted {
		_ = h.sendCallback(ctx.Connection, "onDelCharactorSlot", float64(consumed.ID), float64(consumed.CalculateSID()))
		return
	}
	_ = h.sendCallback(ctx.Connection, "onAddCharactorSlot", h.itemDTO(consumed))
}

func (h *Handler) refreshPetEquipmentOwner(ctx *rtmp.RPCContext, characterID int64, equipment *domainitem.Item) {
	if equipment == nil || equipment.SlotType != domainitem.SlotTypePetEquipped || h.petService == nil {
		return
	}
	pet, err := h.petService.RefreshPetForEquipmentItem(ctx.Context, characterID, equipment.ID)
	if err != nil || pet == nil {
		return
	}
	petDTO := pet.ToDTO()
	data, ok := petDTO["data"].(map[string]interface{})
	if !ok {
		return
	}
	property, ok := data["property"]
	if !ok {
		return
	}
	_ = h.sendCallback(ctx.Connection, "onRefreshPetProp", map[string]interface{}{"id": pet.ID, "s": property})
}

func (h *Handler) setPendingPetEquipmentBind(characterID int64, pending *pendingPetEquipmentBind) {
	h.pendingPetEquipBindsMu.Lock()
	defer h.pendingPetEquipBindsMu.Unlock()
	if pending == nil {
		delete(h.pendingPetEquipBinds, characterID)
		return
	}
	copy := *pending
	h.pendingPetEquipBinds[characterID] = &copy
}

func (h *Handler) takePendingPetEquipmentBind(characterID int64) *pendingPetEquipmentBind {
	h.pendingPetEquipBindsMu.Lock()
	defer h.pendingPetEquipBindsMu.Unlock()
	pending := h.pendingPetEquipBinds[characterID]
	delete(h.pendingPetEquipBinds, characterID)
	if pending == nil {
		return nil
	}
	copy := *pending
	return &copy
}
