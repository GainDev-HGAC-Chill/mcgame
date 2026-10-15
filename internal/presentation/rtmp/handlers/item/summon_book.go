// Open-sourced by BaoLT

package item

import (
	"context"
	"fmt"
	"sort"

	appitem "mcgame-server/internal/application/item"
	domainpet "mcgame-server/internal/domain/pet"
	"mcgame-server/internal/infrastructure/rtmp"
)

func (h *Handler) prepareSummonBookPet(ctx context.Context, characterID int64, templateID int) (*domainpet.Pet, error) {
	config, ok := appitem.GetSummonBookConfig(templateID)
	if !ok {
		return nil, nil
	}
	if h.petService == nil {
		return nil, fmt.Errorf("summon book pet service unavailable")
	}

	pets, err := h.petService.GetPetList(ctx, characterID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(pets, func(i, j int) bool {
		if pets[i] == nil {
			return false
		}
		if pets[j] == nil {
			return true
		}
		return pets[i].ID < pets[j].ID
	})

	var fallback *domainpet.Pet
	for _, candidate := range pets {
		if !summonBookPetEligible(candidate, config.MinQuality) {
			continue
		}
		if candidate.IsFollowing || candidate.ClientState() == int(domainpet.PetStateBattle) {
			return candidate, nil
		}
		if fallback == nil {
			fallback = candidate
		}
	}
	return fallback, nil
}

func summonBookPetEligible(p *domainpet.Pet, minQuality int) bool {
	if p == nil || !p.IsBound() || p.CreatureData == nil {
		return false
	}
	quality, ok := parseFlexibleInt64(p.CreatureData["qLevel"])
	return ok && int(quality) >= minQuality
}

func (h *Handler) activateSummonBookPet(ctx *rtmp.RPCContext, characterID int64, selected *domainpet.Pet) (*domainpet.Pet, error) {
	if selected == nil || h.petService == nil {
		return nil, fmt.Errorf("summon book has no eligible pet")
	}

	currentID := int64(0)
	current, err := h.petService.GetFollowingPet(ctx.Context, characterID)
	if err != nil {
		return nil, err
	}
	if current != nil {
		currentID = current.ID
	}

	following, err := h.petService.StartFollow(ctx.Context, characterID, selected.ID, currentID)
	if err != nil {
		return nil, err
	}
	return following, nil
}

func (h *Handler) sendSummonBookPetCallbacks(ctx *rtmp.RPCContext, p *domainpet.Pet) {
	if ctx == nil || ctx.Connection == nil || p == nil {
		return
	}

	characterID := p.CharacterID
	mapID, _ := ctx.Connection.GetSceneInfo()
	channelID := ctx.Connection.GetChannelID()
	createPayload := map[string]interface{}{
		"cid":     float64(characterID),
		"petData": p.ToSceneDTO(),
	}
	if h.sceneManager != nil && channelID > 0 && mapID > 0 {
		h.sceneManager.BroadcastToScene(channelID, mapID, 0, "onCancelPet", float64(characterID))
		h.sceneManager.BroadcastToScene(channelID, mapID, 0, "onCreatePet", createPayload)
	} else {
		_ = ctx.Connection.SendCallback("onCancelPet", float64(characterID))
		_ = ctx.Connection.SendCallback("onCreatePet", createPayload)
	}

	dto := p.ToDTO()
	if data, ok := dto["data"].(map[string]interface{}); ok {
		if property, exists := data["property"]; exists {
			_ = ctx.Connection.SendCallback("onRefreshPetProp", map[string]interface{}{"id": p.ID, "s": property})
		}
	}
}
