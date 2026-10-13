// Unlinked legacy portals resolve only an exact, unique public-map label.
// Arrival uses the map's authoritative safe point, not an invented reciprocal gate.
package scene

import (
	"math"

	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func (s *Service) resolveUnlinkedGate(gate *models.SceneitemInstanceTemplate) *TeleportDestination {
	template := s.gameDataManager.GetSceneItemTemplate(int(gate.Tid))
	if template == nil || template.Type != 3 || gate.Name == "" {
		return nil
	}
	var destination *models.MapTemplate
	for _, candidate := range s.gameDataManager.GetAllMaps() {
		if candidate.Name != gate.Name {
			continue
		}
		if destination != nil {
			s.logger.Warn("Ambiguous unlinked portal destination", zap.Int64("gate_id", gate.ID), zap.String("name", gate.Name))
			return nil
		}
		destination = candidate
	}
	if destination == nil || destination.ID == int64(gate.PosMapID) || destination.CopyFlag != 0 || destination.Type != 2 {
		return nil
	}
	x, y := destination.SafeX, destination.SafeY
	if math.IsNaN(x) || math.IsInf(x, 0) || math.IsNaN(y) || math.IsInf(y, 0) || x < 0 || y < 0 || (x == 0 && y == 0) || x >= destination.Width || y >= destination.Height {
		s.logger.Warn("Invalid safe point for unlinked portal", zap.Int64("gate_id", gate.ID), zap.Int64("map_id", destination.ID))
		return nil
	}
	s.logger.Info("Resolved unlinked portal to named map safe point", zap.Int64("gate_id", gate.ID), zap.Int64("map_id", destination.ID))
	return &TeleportDestination{MapID: int(destination.ID), X: int(x), Y: int(y)}
}
