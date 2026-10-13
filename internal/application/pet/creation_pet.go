package pet

import (
	"mcgame-server/internal/domain/pet"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
)

func PrepareCreationPet(manager *gamedata.Manager, template *models.CreatureTemplate, growth float64, bound bool) *pet.Pet {
	p := pet.NewPetFromCreature(0, template)
	applyPetGrowRate(p, growth)
	p.SetBinded(bound)
	ensurePetSkillSlots(p)
	s := &Service{gameDataManager: manager}
	s.seedDefaultPetSkills(p)
	p.CurrentHP = p.MaxHP
	p.CurrentMP = p.MaxMP
	return p
}
