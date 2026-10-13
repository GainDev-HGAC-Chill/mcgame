
package combat

import (
    "context"

    domainbuff "mcgame-server/internal/domain/buff"
    "go.uber.org/zap"
)

type ExpBuffProvider interface {
    ListActive(
        ctx context.Context,
        characterID int64,
    ) ([]*domainbuff.Buff, error)
}

func (s *Service) SetExpBuffProvider(p ExpBuffProvider) {
    s.expBuffProvider = p
}

func (s *Service) expMultiplier(
    ctx context.Context,
    charID int64,
    forPet bool,
) int64 {
    multiplier := int64(10)

    if s.expBuffProvider == nil {
        return multiplier
    }

    buffs, err := s.expBuffProvider.ListActive(ctx, charID)
    if err != nil {
        s.logger.Warn(
            "Failed to read EXP buffs",
            zap.Int64("character_id", charID),
            zap.Error(err),
        )
        return multiplier
    }

    for _, b := range buffs {
        if b == nil {
            continue
        }

        var value int64

        switch b.BuffID {
        case 281:
            value = 15
        case 762:
            value = 30
        case 764:
            value = 50
        case 1538:
            if forPet {
                value = 30
            }
        }

        if value > multiplier {
            multiplier = value
        }
    }

    return multiplier
}

func (s *Service) applyExpMultiplier(
    ctx context.Context,
    charID int64,
    base int64,
    forPet bool,
) int64 {
    if base <= 0 {
        return 0
    }

    multiplier := s.expMultiplier(ctx, charID, forPet)

    const maxInt64 = int64(^uint64(0) >> 1)
    if base > maxInt64/multiplier {
        return maxInt64
    }

    return base * multiplier / 10
}
