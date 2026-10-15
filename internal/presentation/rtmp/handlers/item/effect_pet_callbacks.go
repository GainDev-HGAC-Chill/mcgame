// Open-sourced by BaoLT

package item

import (
	"strconv"

	appitem "mcgame-server/internal/application/item"
	domainpet "mcgame-server/internal/domain/pet"
)

type itemCallbackSender interface {
	SendCallback(method string, args ...interface{}) error
}

func sendPetProgressCallbacks(sender itemCallbackSender, progress []appitem.PetProgressResult) {
	if sender == nil {
		return
	}
	for _, update := range progress {
		pet := update.Pet
		if pet == nil {
			continue
		}
		_ = sender.SendCallback("onUpdatePet", float64(pet.ID), "exp", strconv.FormatInt(pet.ClientExperience(), 10))
		if update.LeveledUp {
			_ = sender.SendCallback("onUpdatePet", float64(pet.ID), "level", strconv.Itoa(pet.Level))
		}
		petDTO := pet.ToDTO()
		data, ok := petDTO["data"].(map[string]interface{})
		if !ok {
			continue
		}
		property, ok := data["property"]
		if !ok {
			continue
		}
		_ = sender.SendCallback("onRefreshPetProp", map[string]interface{}{
			"id": pet.ID,
			"s":  property,
		})
	}
}

func sendUpdatedPetCallbacks(sender itemCallbackSender, pets []*domainpet.Pet) {
	if sender == nil {
		return
	}
	for _, pet := range pets {
		if pet == nil {
			continue
		}
		_ = sender.SendCallback("onUpdatePet", float64(pet.ID), "life", strconv.Itoa(pet.Life))
	}
}
