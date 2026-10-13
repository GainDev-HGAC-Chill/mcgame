package item

import (
	"go.uber.org/zap"
	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/gamedata/models"
)

func PrepareCreationEquipment(it *item.Item, template *models.EquiptTemplateTemplate) {
	s := &Service{logger: zap.NewNop()}
	s.populateEquipmentProperties(it, template)
}
