// Open-sourced by BaoLT

// CachedPetRepository wraps pet.Repository with caching.
// Save writes to DB for new pets (needs ID), updates cache for existing.
// Delete tracks deleted IDs for batch persistence.
package cache

import (
	"context"

	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"
)

type CachedPetRepository struct {
	cache    *PlayerCache
	delegate pet.Repository
}

func NewCachedPetRepository(cache *PlayerCache, delegate pet.Repository) *CachedPetRepository {
	return &CachedPetRepository{cache: cache, delegate: delegate}
}

func (r *CachedPetRepository) Save(ctx context.Context, p *pet.Pet) error {
	if p.ID == 0 {
		if err := r.delegate.Save(ctx, p); err != nil {
			return err
		}
		if data := r.cache.GetPlayer(p.CharacterID); data != nil {
			data.AddPetWithoutDirty(p)
		}
		r.upsertColdPet(ctx, p)
		return nil
	}
	if data := r.cache.GetPlayer(p.CharacterID); data != nil {
		data.SetPet(p)
		r.upsertColdPet(ctx, p)
		return nil
	}
	if err := r.delegate.Save(ctx, p); err != nil {
		return err
	}
	r.upsertColdPet(ctx, p)
	return nil
}

func (r *CachedPetRepository) FindByID(ctx context.Context, id int64) (*pet.Pet, error) {
	r.cache.mu.RLock()
	for _, data := range r.cache.activeSessions {
		if p := data.GetPet(id); p != nil {
			r.cache.mu.RUnlock()
			return p, nil
		}
	}
	r.cache.mu.RUnlock()
	return r.delegate.FindByID(ctx, id)
}

func (r *CachedPetRepository) FindByCharacterID(ctx context.Context, characterID int64) ([]*pet.Pet, error) {
	if data := r.cache.GetPlayer(characterID); data != nil {
		return data.GetPets(), nil
	}

	if r.cache.coldLoader != nil {
		petMap, err := r.cache.coldLoader.GetPets(ctx, characterID)
		if err == nil && petMap != nil {
			result := make([]*pet.Pet, 0, len(petMap))
			for _, p := range petMap {
				result = append(result, p)
			}
			return result, nil
		}
	}
	return r.delegate.FindByCharacterID(ctx, characterID)
}

func (r *CachedPetRepository) FindFollowingPet(ctx context.Context, characterID int64) (*pet.Pet, error) {
	if data := r.cache.GetPlayer(characterID); data != nil {
		pets := data.GetPets()
		for _, p := range pets {
			if p.IsFollowing {
				return p, nil
			}
		}
		return nil, nil
	}

	if r.cache.coldLoader != nil {
		petMap, err := r.cache.coldLoader.GetPets(ctx, characterID)
		if err == nil && petMap != nil {
			for _, p := range petMap {
				if p.IsFollowing {
					return p, nil
				}
			}
			return nil, nil
		}
	}
	return r.delegate.FindFollowingPet(ctx, characterID)
}

func (r *CachedPetRepository) Delete(ctx context.Context, id int64) error {
	r.cache.mu.RLock()
	for _, data := range r.cache.activeSessions {
		if p := data.GetPet(id); p != nil {
			r.cache.mu.RUnlock()
			data.DeletePet(id)
			r.deleteColdPet(ctx, p.CharacterID, id)
			return nil
		}
	}
	r.cache.mu.RUnlock()

	stored, _ := r.delegate.FindByID(ctx, id)
	if err := r.delegate.Delete(ctx, id); err != nil {
		return err
	}
	if stored != nil {
		r.deleteColdPet(ctx, stored.CharacterID, id)
	}
	return nil
}

func (r *CachedPetRepository) ApplyFusion(ctx context.Context, mainPet *pet.Pet, consumedPetIDs []int64) error {
	var characterID int64
	if mainPet != nil {
		characterID = mainPet.CharacterID
	} else if len(consumedPetIDs) > 0 {
		if material, err := r.FindByID(ctx, consumedPetIDs[0]); err == nil {
			characterID = material.CharacterID
		}
	}
	fusionRepo, ok := r.delegate.(pet.FusionRepository)
	if !ok {
		return r.applyFusionFallback(ctx, mainPet, consumedPetIDs)
	}
	if err := fusionRepo.ApplyFusion(ctx, mainPet, consumedPetIDs); err != nil {
		return err
	}
	if data := r.cache.GetPlayer(characterID); data != nil {
		data.ApplyPersistedPetFusion(mainPet, consumedPetIDs)
	}
	if mainPet != nil {
		r.upsertColdPet(ctx, mainPet)
	}
	for _, petID := range consumedPetIDs {
		r.deleteColdPet(ctx, characterID, petID)
	}
	return nil
}

func (r *CachedPetRepository) ApplyIllusion(ctx context.Context, mainPet *pet.Pet, consumedPetID, luckItemID int64, luckItemCount int, updateMain bool) (int, error) {
	illusionRepo, ok := r.delegate.(pet.IllusionRepository)
	if !ok {
		return 0, pkgerrors.ErrInvalidInput
	}
	if err := r.cache.SavePlayer(ctx, mainPet.CharacterID); err != nil {
		return 0, err
	}
	remainingLuck, err := illusionRepo.ApplyIllusion(ctx, mainPet, consumedPetID, luckItemID, luckItemCount, updateMain)
	if err != nil {
		return 0, err
	}
	if data := r.cache.GetPlayer(mainPet.CharacterID); data != nil {
		data.ApplyPersistedPetIllusion(mainPet, consumedPetID, luckItemID, remainingLuck, updateMain)
	}
	if updateMain {
		r.upsertColdPet(ctx, mainPet)
	}
	r.deleteColdPet(ctx, mainPet.CharacterID, consumedPetID)
	if luckItemID > 0 && r.cache.warmLoader != nil {
		if remainingLuck == 0 {
			_ = r.cache.warmLoader.DeleteFromRedis(ctx, mainPet.CharacterID)
		} else if data := r.cache.GetPlayer(mainPet.CharacterID); data != nil {
			if it := data.GetItem(luckItemID); it != nil {
				_ = r.cache.warmLoader.UpdateItem(ctx, mainPet.CharacterID, it.ID, it.SlotType, it.SlotIndex)
			}
		}
	}
	return remainingLuck, nil
}

func (r *CachedPetRepository) applyFusionFallback(ctx context.Context, mainPet *pet.Pet, consumedPetIDs []int64) error {
	if mainPet != nil {
		if err := r.Save(ctx, mainPet); err != nil {
			return err
		}
	}
	for _, petID := range consumedPetIDs {
		if err := r.Delete(ctx, petID); err != nil {
			return err
		}
	}
	return nil
}

func (r *CachedPetRepository) Count(ctx context.Context, characterID int64) (int, error) {
	if data := r.cache.GetPlayer(characterID); data != nil {
		return len(data.GetPets()), nil
	}

	if r.cache.coldLoader != nil {
		petMap, err := r.cache.coldLoader.GetPets(ctx, characterID)
		if err == nil && petMap != nil {
			return len(petMap), nil
		}
	}
	return r.delegate.Count(ctx, characterID)
}

func (r *CachedPetRepository) UpdateFollowState(ctx context.Context, petID int64, isFollowing bool) error {
	var charID int64
	var foundInCache bool

	r.cache.mu.RLock()
	for _, data := range r.cache.activeSessions {
		if p := data.GetPet(petID); p != nil {
			charID = p.CharacterID
			foundInCache = true
			p.IsFollowing = isFollowing
			data.SetPet(p)
			break
		}
	}
	r.cache.mu.RUnlock()

	if foundInCache && r.cache.coldLoader != nil {
		petMap, _ := r.cache.coldLoader.GetPets(ctx, charID)
		if petMap != nil {
			if p := petMap[petID]; p != nil {
				p.IsFollowing = isFollowing
				_ = r.cache.coldLoader.SavePets(ctx, charID, petMap)
			}
		}
	}

	if foundInCache {
		return nil
	}
	return r.delegate.UpdateFollowState(ctx, petID, isFollowing)
}

func (r *CachedPetRepository) ClearFollowing(ctx context.Context, characterID int64) error {
	if data := r.cache.GetPlayer(characterID); data != nil {
		pets := data.GetPets()
		for _, p := range pets {
			if p.IsFollowing {
				p.IsFollowing = false
				data.SetPet(p)
			}
		}

		if r.cache.coldLoader != nil {
			petMap, _ := r.cache.coldLoader.GetPets(ctx, characterID)
			if petMap != nil {
				for _, p := range petMap {
					if p.IsFollowing {
						p.IsFollowing = false
					}
				}
				_ = r.cache.coldLoader.SavePets(ctx, characterID, petMap)
			}
		}

		return nil
	}

	if r.cache.coldLoader != nil {
		petMap, err := r.cache.coldLoader.GetPets(ctx, characterID)
		if err == nil && petMap != nil {
			for _, p := range petMap {
				if p.IsFollowing {
					p.IsFollowing = false
				}
			}
			_ = r.cache.coldLoader.SavePets(ctx, characterID, petMap)
			return nil
		}
	}
	return r.delegate.ClearFollowing(ctx, characterID)
}

func (r *CachedPetRepository) upsertColdPet(ctx context.Context, p *pet.Pet) {
	if r.cache.coldLoader == nil || p == nil {
		return
	}

	petMap, err := r.cache.coldLoader.GetPets(ctx, p.CharacterID)
	if err != nil || petMap == nil {
		return
	}

	petMap[p.ID] = p
	_ = r.cache.coldLoader.SavePets(ctx, p.CharacterID, petMap)
}

func (r *CachedPetRepository) deleteColdPet(ctx context.Context, characterID int64, petID int64) {
	if r.cache.coldLoader == nil || characterID == 0 {
		return
	}

	petMap, err := r.cache.coldLoader.GetPets(ctx, characterID)
	if err != nil || petMap == nil {
		return
	}

	delete(petMap, petID)
	_ = r.cache.coldLoader.SavePets(ctx, characterID, petMap)
}
