#!/usr/bin/env python3
"""MCGame pet skill book 506 fix (Thuoc Kich Thich / Quyen Luyen) and buff IDs."""
import argparse
from pathlib import Path
import shutil
import sys

PET_HELPER = '''// Pet skill books write to the same skill1..skill15 slots used by battles.
package pet

import (
    "context"
    "fmt"
    domainpet "mcgame-server/internal/domain/pet"
    pkgerrors "mcgame-server/pkg/errors"
)

// LearnBookSkill validates ownership, prevents duplicates and uses only opened slots.
func (s *Service) LearnBookSkill(ctx context.Context, charID, petID int64, skillID int) (*domainpet.Pet, error) {
    if s == nil || s.petRepo == nil || s.gameDataManager == nil || petID <= 0 || skillID <= 0 {
        return nil, pkgerrors.ErrCannotUseItem
    }
    if s.gameDataManager.GetSkill(skillID) == nil {
        return nil, pkgerrors.ErrCannotUseItem
    }
    p, err := s.getPetAndValidate(ctx, charID, petID)
    if err != nil { return nil, err }
    if p == nil { return nil, pkgerrors.ErrNotFound }
    ensurePetSkillSlots(p)
    free := 0
    for slot := 1; slot <= petSkillLastOpenSlot; slot++ {
        existing := petSkillSlotValue(p, slot)
        if existing == skillID { return nil, fmt.Errorf("Pet đã học kỹ năng này") }
        if existing == 0 && free == 0 { free = slot }
    }
    if free == 0 { return nil, fmt.Errorf("Pet không còn ô kỹ năng đã mở") }
    setPetSkillSlotValue(p, free, skillID)
    if err := s.petRepo.Save(ctx, p); err != nil { return nil, err }
    return p, nil
}
'''
BOOK_HANDLER = '''// Pet skill books (type 506, use_type 2) grant skills to the selected Pet.
package item

import (
    "context"
    "fmt"
    "strings"
    "mcgame-server/internal/domain/character"
    domainitem "mcgame-server/internal/domain/item"
    domainpet "mcgame-server/internal/domain/pet"
    "mcgame-server/internal/gamedata/models"
    pkgerrors "mcgame-server/pkg/errors"
)

type petSkillBookLearner interface {
    LearnBookSkill(ctx context.Context, charID, petID int64, skillID int) (*domainpet.Pet, error)
}

type creBookEffectHandler struct { itemService *Service }
func (h *creBookEffectHandler) CanHandle(tpl *models.ItemTemplateTemplate) bool { return int(tpl.Type) == 506 }

func (h *creBookEffectHandler) Apply(ctx context.Context, uctx *UseContext, char *character.Character, it *domainitem.Item, tpl *models.ItemTemplateTemplate) (*ItemEffectResult, error) {
    if h.itemService == nil || h.itemService.petService == nil || h.itemService.gameDataRec == nil || tpl == nil || char == nil {
        return nil, pkgerrors.ErrCannotUseItem
    }
    if int(tpl.UseType) != 2 {
        // Retain the existing creature-contract behavior for non-skill books.
        creatureID := int(tpl.I1)
        if creatureID <= 0 { return nil, pkgerrors.ErrCannotUseItem }
        pets, err := h.itemService.petService.ContractPets(ctx, char.ID, []int{creatureID})
        if err != nil { return nil, err }
        result := NewEffectResult()
        result.Pets = append(result.Pets, pets...)
        result.Message = fmt.Sprintf("Đã triệu hồi pet từ %s.", tpl.Name)
        return result, nil
    }
    if uctx == nil || uctx.PetID <= 0 { return nil, pkgerrors.ErrCannotUseItem }
    learner, ok := h.itemService.petService.(petSkillBookLearner)
    if !ok { return nil, pkgerrors.ErrCannotUseItem }
    // Exact mappings for verified books. Other 506 books use the level-1 skill name.
    skillID := 0
    switch int(tpl.ID) {
    case 1622: skillID = 3590 // Thuốc Kích Thích
    case 1623: skillID = 3678 // Quyến Luyến Không Rời
    default:
        for _, candidate := range h.itemService.gameDataRec.GetAllSkills() {
            if candidate == nil || int(candidate.Level) != 1 || int(candidate.Kind) != 1 { continue }
            if !strings.EqualFold(strings.TrimSpace(candidate.Name), strings.TrimSpace(tpl.Name)) { continue }
            if skillID == 0 || int(candidate.ID) < skillID { skillID = int(candidate.ID) }
        }
    }
    if skillID == 0 { return nil, pkgerrors.ErrCannotUseItem }
    pet, err := learner.LearnBookSkill(ctx, char.ID, uctx.PetID, skillID)
    if err != nil { return nil, err }
    result := NewEffectResult()
    result.UpdatedPets = []*domainpet.Pet{pet}
    result.Message = fmt.Sprintf("Pet đã học %s.", tpl.Name)
    return result, nil
}
'''

def plan(root):
    f1=root/'internal/application/pet/book_skill_learning.go'
    f2=root/'internal/application/item/effect_cre_book.go'
    f3=root/'internal/application/combat/pet_active_skill_specials.go'
    for p in (f2,f3):
        if not p.is_file(): raise RuntimeError(f'Missing source: {p}')
    if f1.exists() and f1.read_text(encoding='utf-8') != PET_HELPER:
        raise RuntimeError(f'Refusing to overwrite existing {f1}')
    src=f2.read_text(encoding='utf-8')
    if 'type creBookEffectHandler struct' not in src: raise RuntimeError('Unexpected item handler structure')
    spec=f3.read_text(encoding='utf-8')
    token='Family: "Thuốc Kích Thích", Duration: 3,'
    replace='Family: "Thuốc Kích Thích", BuffID: []int{968, 1015, 1016}[level-1], Duration: 3,'
    if token not in spec and replace not in spec: raise RuntimeError('Cannot locate Thuốc Kích Thích status effect')
    return [(f1,PET_HELPER),(f2,BOOK_HANDLER),(f3,spec.replace(token,replace,1))]

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument('root',type=Path)
    ap.add_argument('--apply',action='store_true')
    ns=ap.parse_args()
    items=plan(ns.root.resolve())
    for p,content in items:
        print(('UPDATE' if p.exists() else 'CREATE'),p)
    if not ns.apply:
        print('DRY RUN: rerun with --apply')
        return
    for p,content in items:
        if p.exists():
            bak=Path(str(p)+'.petbooks.bak')
            if not bak.exists(): shutil.copy2(p,bak)
        p.parent.mkdir(parents=True,exist_ok=True)
        p.write_text(content,encoding='utf-8')
    print('Applied. Run gofmt and go test. DO NOT deploy until tests pass.')

if __name__=='__main__':
    try: main()
    except Exception as e:
        print('ERROR:',e,file=sys.stderr)
        sys.exit(1)
