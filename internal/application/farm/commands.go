// Farm command methods expose only atomic mutation paths to presentation handlers.
package farm

import (
	"context"

	domainchar "mcgame-server/internal/domain/character"
	domainfarm "mcgame-server/internal/domain/farm"
)

const (
	ripenImmediatelyGoldCost int64 = 9
	uprootGoldCost           int64 = 1
)

type RipenAndHarvestResult struct {
	Harvest     *HarvestResult
	Character   *domainchar.Character
	GoldCharged int64
}

type UprootResult struct {
	Plot      *domainfarm.Plot
	Character *domainchar.Character
}

func (s *Service) RipenAndHarvestPlot(ctx context.Context, characterID int64, plotNPCID int) (*RipenAndHarvestResult, error) {
	if s.commands == nil {
		return nil, ErrFarmUnavailable
	}
	committed, err := s.commands.RipenAndHarvestAtomic(ctx, domainfarm.RipenAndHarvestCommand{
		CharacterID: characterID,
		PlotNPCID:   plotNPCID,
		GoldCost:    ripenImmediatelyGoldCost,
	})
	if err != nil {
		return nil, err
	}
	if committed == nil || committed.Plot == nil || committed.Character == nil {
		return nil, ErrFarmUnavailable
	}
	return &RipenAndHarvestResult{
		Character:   committed.Character,
		GoldCharged: committed.GoldCharged,
		Harvest: &HarvestResult{
			Plot:                 committed.Plot,
			OwnerCharacterID:     characterID,
			HarvesterCharacterID: characterID,
			RewardTemplateID:     committed.RewardTemplateID,
			RewardCount:          committed.RewardCount,
			RewardName:           s.getItemName(committed.RewardTemplateID),
			AddedInventoryItem:   committed.Reward,
			PlotDeleted:          true,
		},
	}, nil
}

func (s *Service) UprootPlotAtomic(ctx context.Context, characterID int64, plotNPCID int) (*UprootResult, error) {
	if s.commands == nil {
		return nil, ErrFarmUnavailable
	}
	committed, err := s.commands.UprootAtomic(ctx, domainfarm.UprootCommand{
		CharacterID: characterID,
		PlotNPCID:   plotNPCID,
		GoldCost:    uprootGoldCost,
	})
	if err != nil {
		return nil, err
	}
	if committed == nil || committed.Plot == nil || committed.Character == nil {
		return nil, ErrFarmUnavailable
	}
	return &UprootResult{Plot: committed.Plot, Character: committed.Character}, nil
}
