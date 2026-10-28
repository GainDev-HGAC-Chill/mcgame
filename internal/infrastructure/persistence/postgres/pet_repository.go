// Open-sourced by BaoLT

// PostgreSQL implementation of pet repository.
// Handles pet CRUD with JSON property storage for skills and cultivation.
// Supports follow state management and character pet queries.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	domainachievement "mcgame-server/internal/domain/achievement"
	domainitem "mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/jackc/pgx/v5"
)

type PetRepository struct {
	db *Database
}

type petRowScanner interface {
	Scan(dest ...any) error
}

func NewPetRepository(db *Database) *PetRepository {
	return &PetRepository{db: db}
}

func (r *PetRepository) Save(ctx context.Context, p *pet.Pet) error {
	if p.ID == 0 {
		return r.create(ctx, p)
	}
	return r.update(ctx, p)
}

func (r *PetRepository) SaveCaptured(ctx context.Context, p *pet.Pet) error {
	if p == nil || p.ID != 0 || p.CharacterID <= 0 || p.TemplateID <= 0 {
		return pkgerrors.ErrInvalidInput
	}
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to begin captured pet transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := createPet(ctx, tx, p); err != nil {
		return err
	}
	payload, err := json.Marshal(domainachievement.PetCapturedFactPayload{
		PetID:              p.ID,
		CreatureTemplateID: int64(p.TemplateID),
	})
	if err != nil {
		return pkgerrors.Wrap(err, "failed to marshal captured pet fact")
	}
	fact := domainachievement.Fact{
		FactID:      fmt.Sprintf("pet-captured:v1:%d", p.ID),
		CharacterID: p.CharacterID,
		Kind:        domainachievement.FactKindPetCaptured,
		Payload:     payload,
		OccurredAt:  p.CreatedAt.UTC(),
	}
	if err := appendAchievementFact(ctx, tx, fact); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return pkgerrors.Wrap(err, "failed to commit captured pet transaction")
	}
	return nil
}

func (r *PetRepository) create(ctx context.Context, p *pet.Pet) error {
	return createPet(ctx, r.db.pool, p)
}

func createPet(ctx context.Context, queryer creationQueryer, p *pet.Pet) error {
	propertyJSON, err := json.Marshal(p.Property)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to marshal new pet properties")
	}

	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	err = queryer.QueryRow(ctx,
		"select player.create_character_pet($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27::jsonb, $28, $29, $30, $31, $32, $33)",
		p.CharacterID, p.TemplateID, p.Name, p.Level, p.Experience,
		p.AptStrength, p.AptAgility, p.AptStamina, p.AptIntelligence, p.AptEnergy,
		p.AptStrengthEx, p.AptAgilityEx, p.AptStaminaEx, p.AptIntelligenceEx, p.AptEnergyEx,
		p.GrowRate, p.GrowRateAdd, p.UpgradeNum, p.EvolutionLv, p.Element,
		p.CurrentHP, p.CurrentMP, p.MaxHP, p.MaxMP,
		p.IsFollowing, p.IsMounting, propertyJSON, p.CreatedAt, p.UpdatedAt,
		p.AttrPoints, p.MaxAttrPoints, p.DistributedAttrPoints, p.Life,
	).Scan(&p.ID)

	if err != nil {
		return pkgerrors.Wrap(err, "failed to create pet")
	}

	return nil
}

func (r *PetRepository) update(ctx context.Context, p *pet.Pet) error {
	return updatePet(ctx, r.db.pool, p)
}

func updatePet(ctx context.Context, queryer creationQueryer, p *pet.Pet) error {
	propertyJSON, err := json.Marshal(p.Property)
	if err != nil {
		propertyJSON = []byte("{}")
	}

	p.UpdatedAt = time.Now()

	var updated bool
	err = queryer.QueryRow(ctx,
		"select player.update_character_pet($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27::jsonb, $28, $29, $30, $31, $32)",
		p.ID,
		p.TemplateID, p.Name, p.Level, p.Experience,
		p.AptStrength, p.AptAgility, p.AptStamina, p.AptIntelligence, p.AptEnergy,
		p.AptStrengthEx, p.AptAgilityEx, p.AptStaminaEx, p.AptIntelligenceEx, p.AptEnergyEx,
		p.GrowRate, p.GrowRateAdd, p.UpgradeNum, p.EvolutionLv, p.Element,
		p.CurrentHP, p.CurrentMP, p.MaxHP, p.MaxMP,
		p.IsFollowing, p.IsMounting, propertyJSON, p.UpdatedAt,
		p.AttrPoints, p.MaxAttrPoints, p.DistributedAttrPoints, p.Life,
	).Scan(&updated)

	if err != nil {
		return pkgerrors.Wrap(err, "failed to update pet")
	}

	if !updated {
		return pkgerrors.ErrNotFound
	}

	return nil
}

func (r *PetRepository) ApplyFusion(ctx context.Context, mainPet *pet.Pet, consumedPetIDs []int64) error {
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to begin pet fusion")
	}
	defer tx.Rollback(ctx)

	if mainPet != nil {
		if err := updatePet(ctx, tx, mainPet); err != nil {
			return err
		}
	}
	for _, petID := range consumedPetIDs {
		var deleted bool
		if err := tx.QueryRow(ctx, "select player.delete_character_pet($1)", petID).Scan(&deleted); err != nil {
			return pkgerrors.Wrap(err, "failed to delete fusion material pet")
		}
		if !deleted {
			return pkgerrors.ErrNotFound
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return pkgerrors.Wrap(err, "failed to commit pet fusion")
	}
	return nil
}

func (r *PetRepository) ApplyIllusion(ctx context.Context, mainPet *pet.Pet, consumedPetID, luckItemID int64, luckItemCount int, updateMain bool) (int, error) {
	if mainPet == nil || consumedPetID <= 0 || luckItemCount < 0 || luckItemCount > 0 && luckItemID <= 0 {
		return 0, pkgerrors.ErrInvalidInput
	}
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return 0, pkgerrors.Wrap(err, "failed to begin pet illusion")
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, "select id from player.character_pets where character_id=$1 and id = any($2::bigint[]) order by id for update", mainPet.CharacterID, []int64{mainPet.ID, consumedPetID})
	if err != nil {
		return 0, pkgerrors.Wrap(err, "failed to lock illusion pets")
	}
	lockedPets := 0
	for rows.Next() {
		lockedPets++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, pkgerrors.Wrap(err, "failed to read locked illusion pets")
	}
	if lockedPets != 2 {
		return 0, pkgerrors.ErrNotFound
	}

	remainingLuck := 0
	if luckItemCount > 0 {
		var currentStack int
		err = tx.QueryRow(ctx, "select stack_count from player.character_items where id=$1 and character_id=$2 and slot_type=$3 for update", luckItemID, mainPet.CharacterID, domainitem.SlotTypeBag).Scan(&currentStack)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, pkgerrors.ErrItemNotOwned
		}
		if err != nil {
			return 0, pkgerrors.Wrap(err, "failed to lock illusion luck item")
		}
		remainingLuck = currentStack - luckItemCount
		if remainingLuck < 0 {
			return 0, pkgerrors.ErrInvalidInput
		}
	}

	if updateMain {
		if err := updatePet(ctx, tx, mainPet); err != nil {
			return 0, err
		}
	}
	var deleted bool
	if err := tx.QueryRow(ctx, "select player.delete_character_pet($1)", consumedPetID).Scan(&deleted); err != nil {
		return 0, pkgerrors.Wrap(err, "failed to delete illusion material pet")
	}
	if !deleted {
		return 0, pkgerrors.ErrNotFound
	}
	if luckItemCount > 0 {
		if remainingLuck == 0 {
			if err := tx.QueryRow(ctx, "select player.delete_character_item($1)", luckItemID).Scan(&deleted); err != nil {
				return 0, pkgerrors.Wrap(err, "failed to delete illusion luck item")
			}
		} else {
			var updated bool
			if err := tx.QueryRow(ctx, "select player.update_character_item_stack($1, $2, $3)", luckItemID, remainingLuck, time.Now()).Scan(&updated); err != nil {
				return 0, pkgerrors.Wrap(err, "failed to update illusion luck item")
			}
			deleted = updated
		}
		if !deleted {
			return 0, pkgerrors.ErrItemNotFound
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, pkgerrors.Wrap(err, "failed to commit pet illusion")
	}
	return remainingLuck, nil
}

func (r *PetRepository) ApplySkillBook(ctx context.Context, mutation pet.SkillBookMutation) (int, error) {
	if mutation.Pet == nil || mutation.BookItemID <= 0 || mutation.ExpectedBookStack <= 0 || len(mutation.ExpectedSkillSlots) != pet.TotalPetSkillSlots {
		return 0, pkgerrors.ErrInvalidInput
	}
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return 0, pkgerrors.Wrap(err, "failed to begin pet skill-book mutation")
	}
	defer tx.Rollback(ctx)

	var propertyJSON []byte
	err = tx.QueryRow(ctx, "select property from player.character_pets where id=$1 and character_id=$2 for update", mutation.Pet.ID, mutation.Pet.CharacterID).Scan(&propertyJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, pkgerrors.ErrNotFound
	}
	if err != nil {
		return 0, pkgerrors.Wrap(err, "failed to lock pet for skill-book mutation")
	}
	if !petSkillSnapshotMatches(propertyJSON, mutation.ExpectedSkillSlots) {
		return 0, pkgerrors.ErrInvalidInput
	}

	var currentStack int
	err = tx.QueryRow(ctx, "select stack_count from player.character_items where id=$1 and character_id=$2 for update", mutation.BookItemID, mutation.Pet.CharacterID).Scan(&currentStack)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, pkgerrors.ErrItemNotOwned
	}
	if err != nil {
		return 0, pkgerrors.Wrap(err, "failed to lock pet skill book")
	}
	if currentStack != mutation.ExpectedBookStack {
		return 0, pkgerrors.ErrInvalidInput
	}

	if err := updatePet(ctx, tx, mutation.Pet); err != nil {
		return 0, err
	}
	remaining := currentStack - 1
	var changed bool
	if remaining == 0 {
		err = tx.QueryRow(ctx, "select player.delete_character_item($1)", mutation.BookItemID).Scan(&changed)
	} else {
		err = tx.QueryRow(ctx, "select player.update_character_item_stack($1, $2, $3)", mutation.BookItemID, remaining, time.Now()).Scan(&changed)
	}
	if err != nil {
		return 0, pkgerrors.Wrap(err, "failed to consume pet skill book")
	}
	if !changed {
		return 0, pkgerrors.ErrItemNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, pkgerrors.Wrap(err, "failed to commit pet skill-book mutation")
	}
	return remaining, nil
}

func petSkillSnapshotMatches(propertyJSON []byte, expected []int) bool {
	var properties map[string]interface{}
	if len(propertyJSON) == 0 {
		properties = make(map[string]interface{})
	} else if err := json.Unmarshal(propertyJSON, &properties); err != nil {
		return false
	}
	for index, expectedSkillID := range expected {
		slot := index + 1
		actual := petSkillPropertyValue(properties[fmt.Sprintf("skill%d", slot)], slot)
		if actual != expectedSkillID {
			return false
		}
	}
	return true
}

func petSkillPropertyValue(value interface{}, slot int) int {
	defaultValue := -1
	if slot <= 5 {
		defaultValue = 0
	}
	switch typed := value.(type) {
	case float64:
		if int(typed) < 0 {
			return defaultValue
		}
		return int(typed)
	case int:
		if typed < 0 {
			return defaultValue
		}
		return typed
	default:
		return defaultValue
	}
}

func (r *PetRepository) FindByID(ctx context.Context, id int64) (*pet.Pet, error) {
	p := &pet.Pet{}
	err := scanPetRow(r.db.pool.QueryRow(ctx, "select * from player.get_character_pet_by_id($1)", id), p)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pkgerrors.ErrNotFound
		}
		return nil, pkgerrors.Wrap(err, "failed to find pet by ID")
	}

	return p, nil
}

func (r *PetRepository) FindByCharacterID(ctx context.Context, characterID int64) ([]*pet.Pet, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.get_character_pets_by_character($1)", characterID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to find pets by character ID")
	}
	defer rows.Close()

	return r.scanPets(rows)
}

func (r *PetRepository) FindFollowingPet(ctx context.Context, characterID int64) (*pet.Pet, error) {
	p := &pet.Pet{}
	err := scanPetRow(r.db.pool.QueryRow(ctx, "select * from player.get_following_character_pet($1)", characterID), p)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, pkgerrors.Wrap(err, "failed to find following pet")
	}

	return p, nil
}

func (r *PetRepository) Delete(ctx context.Context, id int64) error {
	var deleted bool
	if err := r.db.pool.QueryRow(ctx, "select player.delete_character_pet($1)", id).Scan(&deleted); err != nil {
		return pkgerrors.Wrap(err, "failed to delete pet")
	}

	if !deleted {
		return pkgerrors.ErrNotFound
	}

	return nil
}

func (r *PetRepository) Count(ctx context.Context, characterID int64) (int, error) {
	var count int
	err := r.db.pool.QueryRow(ctx, "select player.count_character_pets($1)", characterID).Scan(&count)
	if err != nil {
		return 0, pkgerrors.Wrap(err, "failed to count pets")
	}

	return count, nil
}

func (r *PetRepository) UpdateFollowState(ctx context.Context, petID int64, isFollowing bool) error {
	var updated bool
	if err := r.db.pool.QueryRow(ctx, "select player.set_pet_follow_state($1, $2, $3)", petID, isFollowing, time.Now()).Scan(&updated); err != nil {
		return pkgerrors.Wrap(err, "failed to update pet follow state")
	}

	if !updated {
		return pkgerrors.ErrNotFound
	}

	return nil
}

func (r *PetRepository) ClearFollowing(ctx context.Context, characterID int64) error {
	_, err := r.db.pool.Exec(ctx, "select player.clear_character_following_pets($1, $2)", characterID, time.Now())
	if err != nil {
		return pkgerrors.Wrap(err, "failed to clear following pets")
	}

	return nil
}

func (r *PetRepository) scanPets(rows pgx.Rows) ([]*pet.Pet, error) {
	var pets []*pet.Pet
	for rows.Next() {
		p := &pet.Pet{}
		if err := scanPetRow(rows, p); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan pet row")
		}
		pets = append(pets, p)
	}

	return pets, nil
}

func scanPetRow(scanner petRowScanner, p *pet.Pet) error {
	var propertyJSON []byte
	if err := scanner.Scan(
		&p.ID, &p.CharacterID, &p.TemplateID, &p.Name, &p.Level, &p.Experience,
		&p.AptStrength, &p.AptAgility, &p.AptStamina, &p.AptIntelligence, &p.AptEnergy,
		&p.AptStrengthEx, &p.AptAgilityEx, &p.AptStaminaEx, &p.AptIntelligenceEx, &p.AptEnergyEx,
		&p.GrowRate, &p.GrowRateAdd, &p.UpgradeNum, &p.EvolutionLv, &p.Element,
		&p.CurrentHP, &p.CurrentMP, &p.MaxHP, &p.MaxMP,
		&p.IsFollowing, &p.IsMounting, &propertyJSON, &p.CreatedAt, &p.UpdatedAt,
		&p.AttrPoints, &p.DistributedAttrPoints, &p.MaxAttrPoints, &p.Life,
	); err != nil {
		return err
	}
	if len(propertyJSON) > 0 {
		if err := json.Unmarshal(propertyJSON, &p.Property); err == nil {
			return nil
		}
	}
	p.Property = make(map[string]interface{})
	return nil
}
