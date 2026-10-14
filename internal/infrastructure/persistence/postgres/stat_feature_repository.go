// Open-sourced by BaoLT

// PostgreSQL stat feature repository persists generic character and pet feature state alongside progression snapshots.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/statfeature"
	pkgerrors "mcgame-server/pkg/errors"

	"github.com/jackc/pgx/v5"
)

type StatFeatureRepository struct {
	db *Database
}

func NewStatFeatureRepository(db *Database) *StatFeatureRepository {
	return &StatFeatureRepository{db: db}
}

func (r *StatFeatureRepository) GetCharacterProgression(ctx context.Context, charID int64) (*statfeature.CharacterProgression, error) {
	progression := &statfeature.CharacterProgression{}
	err := r.db.pool.QueryRow(ctx, "select * from player.get_character_progression($1)", charID).Scan(
		&progression.CharacterID,
		&progression.AwakenLevel,
		&progression.AwakenPoints,
		&progression.AwakenPointsUsed,
		&progression.SoulLevel,
		&progression.SoulExp,
		&progression.SoulPoints,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, pkgerrors.Wrap(err, "failed to load character progression")
	}

	return progression, nil
}

func (r *StatFeatureRepository) UpsertCharacterSoulProgression(ctx context.Context, charID int64, soulLevel int, soulExp int64) error {
	var stored bool
	if err := r.db.pool.QueryRow(
		ctx,
		"select player.upsert_character_soul_progression($1, $2, $3)",
		charID,
		soulLevel,
		soulExp,
	).Scan(&stored); err != nil {
		return pkgerrors.Wrap(err, "failed to upsert character soul progression")
	}
	return nil
}

func (r *StatFeatureRepository) ListCharacterFeatureStates(ctx context.Context, charID int64) ([]*statfeature.CharacterFeatureState, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.list_character_feature_states($1)", charID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to list character stat features")
	}
	defer rows.Close()

	states := make([]*statfeature.CharacterFeatureState, 0)
	for rows.Next() {
		state := &statfeature.CharacterFeatureState{CharacterID: charID}
		var raw []byte
		if err := rows.Scan(&state.FeatureKey, &raw); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan character stat feature")
		}
		state.State = unmarshalFeatureState(raw)
		states = append(states, state)
	}

	return states, nil
}

func (r *StatFeatureRepository) UpsertCharacterFeatureState(ctx context.Context, state *statfeature.CharacterFeatureState) error {
	if state == nil {
		return nil
	}
	if state.FeatureKey == character.CreationPackageFeatureKey {
		return fmt.Errorf("creation package snapshots are immutable")
	}

	raw, err := json.Marshal(normalizeFeatureState(state.State))
	if err != nil {
		return pkgerrors.Wrap(err, "failed to marshal character stat feature state")
	}

	var stored bool
	if err := r.db.pool.QueryRow(
		ctx,
		"select player.upsert_character_feature_state($1, $2, $3::jsonb)",
		state.CharacterID,
		state.FeatureKey,
		raw,
	).Scan(&stored); err != nil {
		return pkgerrors.Wrap(err, "failed to upsert character stat feature state")
	}

	return nil
}

func (r *StatFeatureRepository) CommitBloodBagPurchase(ctx context.Context, charID int64, newGold int64, state map[string]interface{}) error {
	raw, err := json.Marshal(normalizeFeatureState(state))
	if err != nil {
		return pkgerrors.Wrap(err, "failed to marshal blood bag state")
	}
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to begin blood bag purchase")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var storedGold int64
	if err := tx.QueryRow(ctx, "update player.character_wallet set gold = $2 where character_id = $1 returning gold", charID, newGold).Scan(&storedGold); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pkgerrors.ErrCharacterNotFound
		}
		return pkgerrors.Wrap(err, "failed to update blood bag gold")
	}
	var stored bool
	if err := tx.QueryRow(ctx, "select player.upsert_character_feature_state($1, $2, $3::jsonb)", charID, statfeature.FeatureBloodBag, raw).Scan(&stored); err != nil {
		return pkgerrors.Wrap(err, "failed to update blood bag state")
	}
	if err := tx.Commit(ctx); err != nil {
		return pkgerrors.Wrap(err, "failed to commit blood bag purchase")
	}
	return nil
}

func (r *StatFeatureRepository) ListPetFeatureStates(ctx context.Context, petID int64) ([]*statfeature.PetFeatureState, error) {
	rows, err := r.db.pool.Query(ctx, "select * from player.list_pet_feature_states($1)", petID)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to list pet stat features")
	}
	defer rows.Close()

	states := make([]*statfeature.PetFeatureState, 0)
	for rows.Next() {
		state := &statfeature.PetFeatureState{PetID: petID}
		var raw []byte
		if err := rows.Scan(&state.FeatureKey, &raw); err != nil {
			return nil, pkgerrors.Wrap(err, "failed to scan pet stat feature")
		}
		state.State = unmarshalFeatureState(raw)
		states = append(states, state)
	}

	return states, nil
}

func (r *StatFeatureRepository) UpsertPetFeatureState(ctx context.Context, state *statfeature.PetFeatureState) error {
	if state == nil {
		return nil
	}

	raw, err := json.Marshal(normalizeFeatureState(state.State))
	if err != nil {
		return pkgerrors.Wrap(err, "failed to marshal pet stat feature state")
	}

	var stored bool
	if err := r.db.pool.QueryRow(
		ctx,
		"select player.upsert_pet_feature_state($1, $2, $3::jsonb)",
		state.PetID,
		state.FeatureKey,
		raw,
	).Scan(&stored); err != nil {
		return pkgerrors.Wrap(err, "failed to upsert pet stat feature state")
	}

	return nil
}

func (r *StatFeatureRepository) PPVEChallengeNextFloor(ctx context.Context, charID int64, maxFloor int, freeChallenges int, today string) (*statfeature.PPVEChallengeResult, error) {
	var raw []byte
	err := r.db.pool.QueryRow(
		ctx,
		"select player.ppve_challenge_next_floor($1, $2, $3, $4)",
		charID,
		maxFloor,
		freeChallenges,
		today,
	).Scan(&raw)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "ppve_challenge_next_floor: db call failed")
	}

	result := unmarshalFeatureState(raw)

	if errVal, hasErr := result["error"]; hasErr {
		errStr, _ := errVal.(string)
		switch errStr {
		case "max_floor":
			return nil, statfeature.ErrPPVEMaxFloor
		case "no_free_challenges":
			return nil, statfeature.ErrPPVENoFreeChallenges
		}
		return nil, pkgerrors.Wrap(fmt.Errorf("ppve: unknown db error: %s", errStr), "ppve_challenge_next_floor")
	}

	floor, _ := toInt64(result["ppvefloor"])
	freeTime, _ := toInt64(result["freeTime"])
	todayFloor, _ := toInt64(result["todayFloor"])
	lastReset, _ := result["lastResetDay"].(string)

	return &statfeature.PPVEChallengeResult{
		NewFloor:     int(floor),
		FreeTime:     int(freeTime),
		TodayFloor:   int(todayFloor),
		LastResetDay: lastReset,
	}, nil
}

func (r *StatFeatureRepository) PPVEGetRank(ctx context.Context, charID int64, limit int) (*statfeature.PPVERankResult, error) {
	var raw []byte
	err := r.db.pool.QueryRow(ctx, "select player.get_ppve_rank($1, $2)", charID, limit).Scan(&raw)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "get_ppve_rank: db call failed")
	}

	var payload struct {
		Rank []struct {
			CID      string `json:"cid"`
			Name     string `json:"name"`
			ClassID  string `json:"classId"`
			Level    int    `json:"level"`
			FloorNum int    `json:"floorNum"`
		} `json:"rank"`
		MyRank int `json:"myRank"`
	}
	payload.MyRank = -1
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, pkgerrors.Wrap(err, "get_ppve_rank: decode failed")
		}
	}

	entries := make([]statfeature.PPVERankRow, 0, len(payload.Rank))
	for _, e := range payload.Rank {
		entries = append(entries, statfeature.PPVERankRow{
			CID:      e.CID,
			Name:     e.Name,
			ClassID:  e.ClassID,
			Level:    e.Level,
			FloorNum: e.FloorNum,
		})
	}
	return &statfeature.PPVERankResult{Entries: entries, MyRank: payload.MyRank}, nil
}

func (r *StatFeatureRepository) GetActiveMount(ctx context.Context, charID int64) (*statfeature.ActiveMount, error) {
	active := &statfeature.ActiveMount{}
	err := r.db.pool.QueryRow(ctx, "select * from player.get_active_mount($1)", charID).Scan(
		&active.CharacterID,
		&active.MountID,
		&active.Level,
		&active.Experience,
		&active.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, pkgerrors.Wrap(err, "failed to load active mount")
	}

	return active, nil
}

func normalizeFeatureState(state map[string]interface{}) map[string]interface{} {
	if state == nil {
		return map[string]interface{}{}
	}
	return state
}

func unmarshalFeatureState(raw []byte) map[string]interface{} {
	if len(raw) == 0 {
		return map[string]interface{}{}
	}

	state := make(map[string]interface{})
	if err := json.Unmarshal(raw, &state); err != nil {
		return map[string]interface{}{}
	}

	return state
}

func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	}
	return 0, false
}
func (r *StatFeatureRepository) ConsumeBloodBagReserve(
	ctx context.Context,
	charID int64,
	bagType int,
	requested int64,
) (map[string]interface{}, int64, error) {
	if charID <= 0 || requested <= 0 ||
		bagType < 1 || bagType > 4 {
		return nil, 0, fmt.Errorf("invalid blood bag consume")
	}

	keys := []string{
		"", "characterHp", "characterMp", "petHp", "petMp",
	}
	key := keys[bagType]

	var raw []byte
	var consumed int64

	err := r.db.pool.QueryRow(ctx, `
        WITH locked AS (
            SELECT character_id,
                   state,
                   LEAST(
                       GREATEST(
                           COALESCE((state ->> $2)::bigint, 0),
                           0
                       ),
                       $3::bigint
                   ) AS amount
            FROM player.character_stat_features
            WHERE character_id = $1
              AND feature_key = 'blood_bag'
            FOR UPDATE
        ),
        updated AS (
            UPDATE player.character_stat_features f
            SET state = jsonb_set(
                    f.state,
                    ARRAY[$2::text],
                    to_jsonb(
                        GREATEST(
                            COALESCE((f.state ->> $2)::bigint, 0)
                            - locked.amount, 0
                        )
                    ),
                    true
                ),
                updated_at = now()
            FROM locked
            WHERE f.character_id = locked.character_id
              AND f.feature_key = 'blood_bag'
            RETURNING f.state, locked.amount
        )
        SELECT state, amount FROM updated
    `, charID, key, requested).Scan(&raw, &consumed)

	if err == pgx.ErrNoRows {
		return map[string]interface{}{}, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}

	var state map[string]interface{}
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, 0, err
	}

	return state, consumed, nil
}

// RestoreCharacterHPFromBloodBag updates HP and its reserve in one transaction.
// The effective maximum is calculated by the combat layer (including equipment).
func (r *StatFeatureRepository) RestoreCharacterHPFromBloodBag(
	ctx context.Context, charID int64, effectiveMax int,
) (int, int64, map[string]interface{}, error) {
	if charID <= 0 || effectiveMax <= 0 {
		return 0, 0, nil, fmt.Errorf("invalid blood bag recovery")
	}
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return 0, 0, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var hp int
	err = tx.QueryRow(ctx, `SELECT current_hp FROM player.character_attributes WHERE character_id=$1 FOR UPDATE`, charID).Scan(&hp)
	if err != nil {
		return 0, 0, nil, err
	}
	if hp <= 0 || hp >= effectiveMax {
		return hp, 0, nil, nil
	}
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT state FROM player.character_stat_features WHERE character_id=$1 AND feature_key='blood_bag' FOR UPDATE`, charID).Scan(&raw)
	if err == pgx.ErrNoRows {
		return hp, 0, nil, nil
	}
	if err != nil {
		return 0, 0, nil, err
	}
	var state map[string]interface{}
	if err := json.Unmarshal(raw, &state); err != nil {
		return 0, 0, nil, err
	}
	reserve, valid := toInt64(state["characterHp"])
	if !valid || reserve <= 0 {
		return hp, 0, state, nil
	}
	consumed := int64(effectiveMax - hp)
	if consumed > reserve {
		consumed = reserve
	}
	newHP := hp + int(consumed)
	state["characterHp"] = reserve - consumed
	updated, err := json.Marshal(state)
	if err != nil {
		return 0, 0, nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE player.character_attributes SET current_hp=$2 WHERE character_id=$1`, charID, newHP); err != nil {
		return 0, 0, nil, err
	}
	if _, err = tx.Exec(ctx, `UPDATE player.character_stat_features SET state=$2::jsonb, updated_at=now() WHERE character_id=$1 AND feature_key='blood_bag'`, charID, updated); err != nil {
		return 0, 0, nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, 0, nil, err
	}
	return newHP, consumed, state, nil
}
func (r *StatFeatureRepository) RestorePetVitalsFromBloodBag(
	ctx context.Context,
	charID, petID int64,
	effectiveMaxHP, effectiveMaxMP int,
) (int, int, int64, int64, map[string]interface{}, error) {
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var hp, mp, life int
	err = tx.QueryRow(ctx, `
		SELECT current_hp, current_mp, life
		FROM player.character_pets
		WHERE id = $1 AND character_id = $2
		FOR UPDATE
	`, petID, charID).Scan(&hp, &mp, &life)
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}
	if life <= 0 {
		return hp, mp, 0, 0, nil, nil
	}

	var raw []byte
	err = tx.QueryRow(ctx, `
		SELECT state
		FROM player.character_stat_features
		WHERE character_id = $1 AND feature_key = 'blood_bag'
		FOR UPDATE
	`, charID).Scan(&raw)
	if err == pgx.ErrNoRows {
		return hp, mp, 0, 0, nil, nil
	}
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}

	var state map[string]interface{}
	if err := json.Unmarshal(raw, &state); err != nil {
		return 0, 0, 0, 0, nil, err
	}

	hpReserve, _ := toInt64(state["petHp"])
	mpReserve, _ := toInt64(state["petMp"])

	var usedHP, usedMP int64

	if hp > 0 && effectiveMaxHP > hp && hpReserve > 0 {
		usedHP = int64(effectiveMaxHP - hp)
		if usedHP > hpReserve {
			usedHP = hpReserve
		}
	}

	if effectiveMaxMP > mp && mpReserve > 0 {
		usedMP = int64(effectiveMaxMP - mp)
		if usedMP > mpReserve {
			usedMP = mpReserve
		}
	}

	if usedHP == 0 && usedMP == 0 {
		return hp, mp, 0, 0, state, nil
	}

	newHP := hp + int(usedHP)
	newMP := mp + int(usedMP)

	state["petHp"] = hpReserve - usedHP
	state["petMp"] = mpReserve - usedMP

	updated, err := json.Marshal(state)
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE player.character_pets
		SET current_hp = $2,
		    current_mp = $3,
		    updated_at = now()
		WHERE id = $1 AND character_id = $4
	`, petID, newHP, newMP, charID)
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE player.character_stat_features
		SET state = $2::jsonb, updated_at = now()
		WHERE character_id = $1 AND feature_key = 'blood_bag'
	`, charID, updated)
	if err != nil {
		return 0, 0, 0, 0, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, 0, 0, nil, err
	}

	return newHP, newMP, usedHP, usedMP, state, nil
}
