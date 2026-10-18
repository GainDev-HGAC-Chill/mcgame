// Open-sourced by BaoLT

package petarena

import (
	"context"
	"testing"

	domainpet "mcgame-server/internal/domain/pet"
	domainpetarena "mcgame-server/internal/domain/petarena"

	"go.uber.org/zap"
)

type stubPetArenaRepo struct {
	configs  map[string]*domainpetarena.FightConfig
	entries  map[string]*domainpetarena.Entry
	rankings map[int64]*domainpetarena.ArenaRanking
	battle   *domainpetarena.ArenaBattle
}

func (r *stubPetArenaRepo) configKey(characterID int64, key string) string {
	return string(rune(characterID)) + ":" + key
}

func (r *stubPetArenaRepo) GetRanking(ctx context.Context, characterID int64, season int) (*domainpetarena.ArenaRanking, error) {
	return nil, nil
}
func (r *stubPetArenaRepo) GetOrCreateRanking(ctx context.Context, characterID int64, season int) (*domainpetarena.ArenaRanking, error) {
	if r.rankings == nil {
		r.rankings = map[int64]*domainpetarena.ArenaRanking{}
	}
	if r.rankings[characterID] == nil {
		r.rankings[characterID] = domainpetarena.NewArenaRanking(characterID, season)
	}
	return r.rankings[characterID], nil
}
func (r *stubPetArenaRepo) SaveRanking(ctx context.Context, ranking *domainpetarena.ArenaRanking) error {
	if r.rankings == nil {
		r.rankings = map[int64]*domainpetarena.ArenaRanking{}
	}
	r.rankings[ranking.CharacterID] = ranking
	return nil
}
func (r *stubPetArenaRepo) GetTopRankings(ctx context.Context, season int, limit, offset int) ([]*domainpetarena.ArenaRanking, error) {
	return nil, nil
}
func (r *stubPetArenaRepo) GetRank(ctx context.Context, characterID int64, season int) (int, error) {
	return 0, nil
}
func (r *stubPetArenaRepo) SaveBattle(ctx context.Context, battle *domainpetarena.ArenaBattle) error {
	r.battle = battle
	return nil
}
func (r *stubPetArenaRepo) GetBattle(ctx context.Context, battleID int64) (*domainpetarena.ArenaBattle, error) {
	return nil, nil
}
func (r *stubPetArenaRepo) GetBattleHistory(ctx context.Context, characterID int64, limit int) ([]*domainpetarena.ArenaBattle, error) {
	return nil, nil
}
func (r *stubPetArenaRepo) GetReward(ctx context.Context, characterID int64, season int) (*domainpetarena.ArenaReward, error) {
	return nil, nil
}
func (r *stubPetArenaRepo) SaveReward(ctx context.Context, reward *domainpetarena.ArenaReward) error {
	return nil
}
func (r *stubPetArenaRepo) FindOpponents(ctx context.Context, characterID int64, rating int, season int, limit int) ([]*domainpetarena.ArenaRanking, error) {
	return nil, nil
}
func (r *stubPetArenaRepo) GetFightConfig(ctx context.Context, characterID int64, configKey string) (*domainpetarena.FightConfig, error) {
	if r.configs == nil {
		return nil, nil
	}
	return r.configs[r.configKey(characterID, configKey)], nil
}
func (r *stubPetArenaRepo) SaveFightConfig(ctx context.Context, config *domainpetarena.FightConfig) error {
	if r.configs == nil {
		r.configs = map[string]*domainpetarena.FightConfig{}
	}
	r.configs[r.configKey(config.CharacterID, config.ConfigKey)] = config
	return nil
}
func (r *stubPetArenaRepo) GetEntry(ctx context.Context, characterID int64, season int) (*domainpetarena.Entry, error) {
	if r.entries == nil {
		return nil, nil
	}
	return r.entries[r.configKey(characterID, string(rune(season)))], nil
}
func (r *stubPetArenaRepo) SaveEntry(ctx context.Context, entry *domainpetarena.Entry) error {
	if r.entries == nil {
		r.entries = map[string]*domainpetarena.Entry{}
	}
	r.entries[r.configKey(entry.CharacterID, string(rune(entry.Season)))] = entry
	return nil
}
func (r *stubPetArenaRepo) ListEntries(ctx context.Context, season int, limit int) ([]*domainpetarena.Entry, error) {
	result := make([]*domainpetarena.Entry, 0)
	for _, entry := range r.entries {
		if entry.Season == season {
			result = append(result, entry)
		}
	}
	return result, nil
}

type stubPetRepo struct {
	pets map[int64]*domainpet.Pet
}

func (r *stubPetRepo) Save(ctx context.Context, pet *domainpet.Pet) error { return nil }
func (r *stubPetRepo) FindByID(ctx context.Context, id int64) (*domainpet.Pet, error) {
	return r.pets[id], nil
}
func (r *stubPetRepo) FindByCharacterID(ctx context.Context, characterID int64) ([]*domainpet.Pet, error) {
	result := make([]*domainpet.Pet, 0)
	for _, p := range r.pets {
		if p.CharacterID == characterID {
			result = append(result, p)
		}
	}
	return result, nil
}
func (r *stubPetRepo) FindFollowingPet(ctx context.Context, characterID int64) (*domainpet.Pet, error) {
	return nil, nil
}
func (r *stubPetRepo) Delete(ctx context.Context, id int64) error { return nil }
func (r *stubPetRepo) Count(ctx context.Context, characterID int64) (int, error) {
	return 0, nil
}
func (r *stubPetRepo) UpdateFollowState(ctx context.Context, petID int64, isFollowing bool) error {
	return nil
}
func (r *stubPetRepo) ClearFollowing(ctx context.Context, characterID int64) error { return nil }

func TestSaveAndLoadPetFightConfig(t *testing.T) {
	repo := &stubPetArenaRepo{}
	petRepo := &stubPetRepo{pets: map[int64]*domainpet.Pet{
		4733382: {ID: 4733382, CharacterID: 35644, Property: map[string]interface{}{"skill1": 5345}},
	}}
	service := NewService(repo, petRepo, zap.NewNop())

	conf := map[string]interface{}{
		"conf1": map[string]interface{}{
			"5": map[string]interface{}{
				"pid": 4733382,
				"pos": 5,
				"cmdList": []interface{}{
					map[string]interface{}{
						"id":     "5345",
						"level":  "3",
						"action": -50,
						"t":      "54",
					},
				},
			},
		},
	}

	if err := service.SavePetFightConfig(context.Background(), 35644, "farm", conf); err != nil {
		t.Fatalf("SavePetFightConfig() error = %v", err)
	}

	loaded, err := service.GetPetFightConfig(context.Background(), 35644, "farm")
	if err != nil {
		t.Fatalf("GetPetFightConfig() error = %v", err)
	}

	conf1, ok := loaded["conf1"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected conf1 map in loaded config")
	}
	slot5, ok := conf1["5"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected slot 5 entry in loaded config")
	}
	if slot5["pos"] != 5 {
		t.Fatalf("slot5 pos = %#v, expected 5", slot5["pos"])
	}
}

func TestPetFightConfigContextsAreIndependent(t *testing.T) {
	repo := &stubPetArenaRepo{}
	petRepo := &stubPetRepo{pets: map[int64]*domainpet.Pet{
		11: {ID: 11, CharacterID: 7, Property: map[string]interface{}{}},
		12: {ID: 12, CharacterID: 7, Property: map[string]interface{}{}},
	}}
	service := NewService(repo, petRepo, zap.NewNop())
	farm := fightConfigPayload(singlePetLineup(11, 5, nil))
	hmtxls := fightConfigPayload(singlePetLineup(12, 6, nil))

	if err := service.SavePetFightConfig(context.Background(), 7, ConfigContextFarm, farm); err != nil {
		t.Fatalf("save farm: %v", err)
	}
	if err := service.SavePetFightConfig(context.Background(), 7, ConfigContextHMTXLS, hmtxls); err != nil {
		t.Fatalf("save hmtxls: %v", err)
	}
	loadedFarm, err := service.GetPetFightConfig(context.Background(), 7, ConfigContextFarm)
	if err != nil {
		t.Fatalf("load farm: %v", err)
	}
	loadedHMTXLS, err := service.GetPetFightConfig(context.Background(), 7, ConfigContextHMTXLS)
	if err != nil {
		t.Fatalf("load hmtxls: %v", err)
	}
	if firstConfiguredPetID(t, loadedFarm) != 11 || firstConfiguredPetID(t, loadedHMTXLS) != 12 {
		t.Fatalf("contexts overwritten: farm=%v hmtxls=%v", loadedFarm, loadedHMTXLS)
	}
}

func TestSavePetFightConfigRejectsForgedLineupsAndCommands(t *testing.T) {
	petRepo := &stubPetRepo{pets: map[int64]*domainpet.Pet{
		11: {
			ID:          11,
			CharacterID: 7,
			Property: map[string]interface{}{
				"skill1": 5345,
				"skills": []interface{}{map[string]interface{}{"id": float64(5345), "level": float64(2), "slot": float64(1)}},
			},
		},
		12: {ID: 12, CharacterID: 8, Property: map[string]interface{}{}},
	}}

	tests := []struct {
		name   string
		lineup map[string]interface{}
	}{
		{name: "foreign pet", lineup: singlePetLineup(12, 5, nil)},
		{name: "illegal position", lineup: singlePetLineup(11, 4, nil)},
		{name: "position mismatch", lineup: map[string]interface{}{"5": petLineupEntry(11, 6, nil)}},
		{name: "duplicate pet", lineup: map[string]interface{}{"5": petLineupEntry(11, 5, nil), "6": petLineupEntry(11, 6, nil)}},
		{name: "too many commands", lineup: singlePetLineup(11, 5, repeatedCommands(25))},
		{name: "invalid action", lineup: singlePetLineup(11, 5, []interface{}{command(-30, 1000, -1, 1)})},
		{name: "invalid basic level", lineup: singlePetLineup(11, 5, []interface{}{command(petActionAttack, 1000, 1, 1)})},
		{name: "forged skill", lineup: singlePetLineup(11, 5, []interface{}{command(petActionSkill, 9999, 2, 1)})},
		{name: "forged skill level", lineup: singlePetLineup(11, 5, []interface{}{command(petActionSkill, 5345, 3, 1)})},
		{name: "unsafe target", lineup: singlePetLineup(11, 5, []interface{}{command(petActionSkill, 5345, 2, 55)})},
		{name: "fractional pet id", lineup: map[string]interface{}{"5": map[string]interface{}{"pid": 11.5, "pos": 5, "cmdList": []interface{}{}}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(&stubPetArenaRepo{}, petRepo, zap.NewNop())
			if err := service.SavePetFightConfig(context.Background(), 7, ConfigContextFarm, fightConfigPayload(test.lineup)); err == nil {
				t.Fatal("expected invalid config to fail")
			}
		})
	}
}

func TestRegisterArenaEntryRequiresTwoToFivePetsAndPersistsArenaSnapshot(t *testing.T) {
	repo := &stubPetArenaRepo{}
	petRepo := &stubPetRepo{pets: map[int64]*domainpet.Pet{
		11: {ID: 11, CharacterID: 7, Property: map[string]interface{}{}},
		12: {ID: 12, CharacterID: 7, Property: map[string]interface{}{}},
	}}
	service := NewService(repo, petRepo, zap.NewNop())
	if _, err := service.RegisterArenaEntry(context.Background(), 7, "team", singlePetLineup(11, 5, nil)); err == nil {
		t.Fatal("one-pet arena registration should fail")
	}
	lineup := map[string]interface{}{
		"5": petLineupEntry(11, 5, nil),
		"6": petLineupEntry(12, 6, nil),
	}
	entry, err := service.RegisterArenaEntry(context.Background(), 7, "team", lineup)
	if err != nil {
		t.Fatalf("register arena: %v", err)
	}
	loaded, err := service.GetPetFightConfig(context.Background(), 7, ConfigContextArena)
	if err != nil {
		t.Fatalf("load arena config: %v", err)
	}
	if firstConfiguredPetID(t, loaded) != 11 || firstConfiguredPetID(t, entry.ConfData) != 11 {
		t.Fatalf("arena snapshots differ: config=%v entry=%v", loaded, entry.ConfData)
	}
	if repo.rankings[7].PetID == nil || *repo.rankings[7].PetID != 11 {
		t.Fatalf("arena lead pet = %v, want 11", repo.rankings[7].PetID)
	}
}

func TestSavePetFightConfigAcceptsSparseTwentyFourSlotSequence(t *testing.T) {
	commands := make([]interface{}, 24)
	commands[23] = command(petActionAttack, 1000, -1, 1)
	service := NewService(&stubPetArenaRepo{}, &stubPetRepo{pets: map[int64]*domainpet.Pet{
		11: {ID: 11, CharacterID: 7, Property: map[string]interface{}{}},
	}}, zap.NewNop())
	if err := service.SavePetFightConfig(context.Background(), 7, ConfigContextFarm, fightConfigPayload(singlePetLineup(11, 5, commands))); err != nil {
		t.Fatalf("sparse 24-slot config rejected: %v", err)
	}
}

func TestFightUsesRegisteredArenaSnapshotInsteadOfStaleRankingPet(t *testing.T) {
	staleAttackerPetID := int64(99)
	staleDefenderPetID := int64(199)
	repo := &stubPetArenaRepo{
		entries: map[string]*domainpetarena.Entry{},
		rankings: map[int64]*domainpetarena.ArenaRanking{
			7: {CharacterID: 7, Season: 1, Rating: 1000, Tickets: 1, MaxTickets: 10, PetID: &staleAttackerPetID},
			8: {CharacterID: 8, Season: 1, Rating: 1000, Tickets: 1, MaxTickets: 10, PetID: &staleDefenderPetID},
		},
	}
	repo.entries[repo.configKey(7, string(rune(1)))] = &domainpetarena.Entry{
		CharacterID: 7,
		Season:      1,
		ConfData: fightConfigPayload(map[string]interface{}{
			"5": petLineupEntry(11, 5, nil),
			"6": petLineupEntry(12, 6, nil),
		}),
	}
	repo.entries[repo.configKey(8, string(rune(1)))] = &domainpetarena.Entry{
		CharacterID: 8,
		Season:      1,
		ConfData: fightConfigPayload(map[string]interface{}{
			"5": petLineupEntry(21, 5, nil),
			"6": petLineupEntry(22, 6, nil),
		}),
	}
	petRepo := &stubPetRepo{pets: map[int64]*domainpet.Pet{
		11: battleTestPet(11, 7),
		12: battleTestPet(12, 7),
		21: battleTestPet(21, 8),
		22: battleTestPet(22, 8),
	}}
	service := NewService(repo, petRepo, zap.NewNop())
	battle, _, err := service.Fight(context.Background(), 7, 8)
	if err != nil {
		t.Fatalf("Fight() error = %v", err)
	}
	if battle.AttackerPetID != 11 || battle.DefenderPetID != 21 {
		t.Fatalf("battle pets = %d,%d, want registered leads 11,21", battle.AttackerPetID, battle.DefenderPetID)
	}
}

func fightConfigPayload(lineup map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{"conf1": lineup}
}

func singlePetLineup(petID int64, position int, commands []interface{}) map[string]interface{} {
	return map[string]interface{}{string(rune('0' + position)): petLineupEntry(petID, position, commands)}
}

func petLineupEntry(petID int64, position int, commands []interface{}) map[string]interface{} {
	if commands == nil {
		commands = []interface{}{}
	}
	return map[string]interface{}{"pid": petID, "pos": position, "cmdList": commands}
}

func command(action, id, level, target int) map[string]interface{} {
	return map[string]interface{}{"action": action, "id": id, "level": level, "t": target}
}

func repeatedCommands(count int) []interface{} {
	commands := make([]interface{}, count)
	for index := range commands {
		commands[index] = command(petActionAttack, 1000, -1, 1)
	}
	return commands
}

func battleTestPet(id, characterID int64) *domainpet.Pet {
	return &domainpet.Pet{
		ID:          id,
		CharacterID: characterID,
		MaxHP:       10,
		Property: map[string]interface{}{
			"finalAttack":   5,
			"finalDefence":  1,
			"finalMAttack":  5,
			"finalMDefence": 1,
			"finalSpeed":    1,
			"finalHit":      100,
		},
	}
}

func firstConfiguredPetID(t *testing.T, payload map[string]interface{}) int64 {
	t.Helper()
	lineup, ok := payload["conf1"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing conf1: %v", payload)
	}
	for position := 5; position <= 9; position++ {
		entry, ok := lineup[string(rune('0'+position))].(map[string]interface{})
		if !ok {
			continue
		}
		petID, ok := exactInt64(entry["pid"])
		if ok {
			return petID
		}
	}
	t.Fatalf("missing configured pet: %v", payload)
	return 0
}
