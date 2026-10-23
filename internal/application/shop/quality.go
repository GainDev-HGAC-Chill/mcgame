// Open-sourced by BaoLT

// Shop quality validation and canonical equipment color mapping.
package shop

import (
	"errors"
	"math"

	domainitem "mcgame-server/internal/domain/item"
)

var errInvalidShopQuality = errors.New("shop quality must be an integer from 0 to 25")

func shopColorCodeFromQuality(quality float64) (int, error) {
	if math.IsNaN(quality) || math.IsInf(quality, 0) || quality != math.Trunc(quality) || quality < 0 || quality > 25 {
		return 0, errInvalidShopQuality
	}
	return domainitem.EquipmentColorCodeFromQuality(int(quality)), nil
}
