// Open-sourced by BaoLT

// Monster Heart application service: panel state load, slot placement, and slot removal.
package monsterheart

import (
	"context"
	"errors"
	"fmt"
	"sync"

	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

var (
	ErrInvalidBox       = errors.New("invalid box index")
	ErrInvalidHole      = errors.New("invalid hole index")
	ErrItemNotInBag     = errors.New("item not in monster-heart bag")
	ErrSlotAlreadyFull  = errors.New("target hole already occupied")
	ErrSlotEmpty        = errors.New("target hole already empty")
	ErrCatalogEntry     = errors.New("invalid monster-heart catalog entry")
	ErrHeartType        = errors.New("monster-heart type mismatch")
	ErrHoleClosed       = errors.New("target hole is closed")
	ErrHeartUnavailable = errors.New("monster-heart ownership count exhausted")
)

type Catalog interface {
	GetCreatureHeart(id int) *models.CreaturehHeartTemplate
	GetCreatureHeartPoint(id int) *models.CreaturehPointTemplate
}

type Service struct {
	repo          domainfeature.Repository
	logger        *zap.Logger
	catalog       Catalog
	mutationLocks [64]sync.Mutex
}

func (s *Service) SetCatalog(catalog Catalog) {
	s.catalog = catalog
}

func NewService(repo domainfeature.Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) Load(ctx context.Context, charID int64) (*State, error) {
	if s == nil || s.repo == nil {
		return NewState(), nil
	}
	states, err := s.repo.ListCharacterFeatureStates(ctx, charID)
	if err != nil {
		return nil, fmt.Errorf("monsterheart: load feature state: %w", err)
	}
	for _, st := range states {
		if st == nil || st.FeatureKey != domainfeature.FeatureMonsterHeart {
			continue
		}
		return DecodeState(st.State), nil
	}
	return NewState(), nil
}

func (s *Service) saveState(ctx context.Context, charID int64, state *State) error {
	if s == nil || s.repo == nil {
		return nil
	}
	encoded := state.Encode()
	return s.repo.UpsertCharacterFeatureState(ctx, &domainfeature.CharacterFeatureState{
		CharacterID: charID,
		FeatureKey:  domainfeature.FeatureMonsterHeart,
		State:       encoded,
	})
}

func (s *Service) SetSlot(ctx context.Context, charID int64, bagEntryID, holePos, box, claimedType int) (*State, error) {
	if !IsValidBox(box) {
		return nil, ErrInvalidBox
	}
	if !IsValidHole(holePos) {
		return nil, ErrInvalidHole
	}

	lock := &s.mutationLocks[uint64(charID)%uint64(len(s.mutationLocks))]
	lock.Lock()
	defer lock.Unlock()

	state, err := s.Load(ctx, charID)
	if err != nil {
		return nil, err
	}

	category, entry, ok := state.BagEntry(bagEntryID)
	if !ok {
		return nil, ErrItemNotInBag
	}
	if s.catalog == nil {
		return nil, ErrCatalogEntry
	}
	heart := s.catalog.GetCreatureHeart(entry.ItemID)
	point := s.catalog.GetCreatureHeartPoint(box*10 + holePos)
	if heart == nil || point == nil {
		return nil, ErrCatalogEntry
	}
	heartType := int(heart.Type)
	if heartType != claimedType || category != categoryForHeartType(heartType) {
		return nil, ErrHeartType
	}
	if int(point.Type) > 0 && int(point.Type) != heartType {
		return nil, ErrHeartType
	}

	existing := state.GetHoleItem(box, holePos)
	if existing == 0 {
		return nil, ErrHoleClosed
	}
	if existing != EmptySlot {
		return nil, ErrSlotAlreadyFull
	}
	if state.EquippedCount(entry.ItemID) >= state.BagItemCount(entry.ItemID) {
		return nil, ErrHeartUnavailable
	}

	state.PlaceInHole(box, holePos, entry.ItemID)

	if err := s.saveState(ctx, charID, state); err != nil {
		return nil, fmt.Errorf("monsterheart: save after set: %w", err)
	}

	s.logger.Info("monsterheart: slot set",
		zap.Int64("char_id", charID),
		zap.Int("bag_entry_id", bagEntryID),
		zap.Int("box", box),
		zap.Int("hole_pos", holePos))

	return state, nil
}

func (s *Service) RemoveSlot(ctx context.Context, charID int64, holePos, box int) (*State, error) {
	if !IsValidBox(box) {
		return nil, ErrInvalidBox
	}
	if !IsValidHole(holePos) {
		return nil, ErrInvalidHole
	}

	lock := &s.mutationLocks[uint64(charID)%uint64(len(s.mutationLocks))]
	lock.Lock()
	defer lock.Unlock()

	state, err := s.Load(ctx, charID)
	if err != nil {
		return nil, err
	}

	existing := state.GetHoleItem(box, holePos)
	if existing == EmptySlot || existing == 0 {
		return nil, ErrSlotEmpty
	}

	state.ClearHole(box, holePos)

	if err := s.saveState(ctx, charID, state); err != nil {
		return nil, fmt.Errorf("monsterheart: save after remove: %w", err)
	}

	s.logger.Info("monsterheart: slot removed",
		zap.Int64("char_id", charID),
		zap.Int("box", box),
		zap.Int("hole_pos", holePos))

	return state, nil
}

func categoryForHeartType(heartType int) string {
	categories := [...]string{"", "ren", "shou", "zhi", "ji", "mo", "lon", "te"}
	if heartType < 1 || heartType >= len(categories) {
		return ""
	}
	return categories[heartType]
}
