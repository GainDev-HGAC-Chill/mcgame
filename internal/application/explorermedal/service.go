// Open-sourced by BaoLT

// Explorer medal progression persists level and current progress in the feature state.
package explorermedal

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	domainchar "mcgame-server/internal/domain/character"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"
)

const MaxMedalLevel = 100

var (
	ErrAtMaxLevel        = errors.New("explorermedal: already at max level")
	ErrInsufficientScore = errors.New("explorermedal: insufficient explorer points")
	ErrInsufficientGold  = errors.New("explorermedal: insufficient gold")
	ErrInvalidGoldType   = errors.New("explorermedal: invalid gold upgrade type")
	ErrCharacterNotFound = errors.New("explorermedal: character not found")
)

type StateAccessor interface {
	ListCharacterFeatureStates(ctx context.Context, charID int64) ([]*domainfeature.CharacterFeatureState, error)
	UpsertCharacterFeatureState(ctx context.Context, state *domainfeature.CharacterFeatureState) error
}

type CharacterRepository interface {
	FindByID(ctx context.Context, id int64) (*domainchar.Character, error)
	Update(ctx context.Context, character *domainchar.Character) error
}

type LevelUpResult struct {
	Level        int
	ExplorerPnt  int
	Progress     int
	Gold         int64
	PropMap      map[string]interface{}
	WalletSpent  int
	GoldSpent    int64
	LevelChanged bool
}

type Service struct {
	repo          StateAccessor
	charRepo      CharacterRepository
	gameData      *gamedata.Manager
	mutationLocks [64]sync.Mutex
}

func NewService(repo StateAccessor, charRepo CharacterRepository, gameData *gamedata.Manager) *Service {
	return &Service{repo: repo, charRepo: charRepo, gameData: gameData}
}

func (s *Service) GetState(ctx context.Context, charID int64) (level int, progress int, err error) {
	states, err := s.repo.ListCharacterFeatureStates(ctx, charID)
	if err != nil {
		return 0, 0, fmt.Errorf("explorermedal: list feature states: %w", err)
	}
	for _, st := range states {
		if st != nil && st.FeatureKey == domainfeature.FeatureExplorerMedal {
			level, progress = parseInfoField(st.State)
			return level, progress, nil
		}
	}
	return 0, 0, nil
}

func (s *Service) GetPanelResponse(ctx context.Context, charID int64) ([]interface{}, error) {
	level, progress, err := s.GetState(ctx, charID)
	if err != nil {
		return nil, err
	}
	char, err := s.charRepo.FindByID(ctx, charID)
	if err != nil {
		return nil, fmt.Errorf("explorermedal: load character: %w", err)
	}
	if char == nil {
		return nil, ErrCharacterNotFound
	}
	return s.BuildPanelResponse(level, char.ShowTimePoint, progress), nil
}

func (s *Service) BuildPanelResponse(level, explorerPnt, progress int) []interface{} {
	return []interface{}{level, s.buildCumulativePropMap(level), explorerPnt, progress}
}

func (s *Service) LevelUp(ctx context.Context, charID int64) (LevelUpResult, error) {
	unlock := s.lockCharacter(charID)
	defer unlock()

	level, progress, char, err := s.loadMutationState(ctx, charID)
	if err != nil {
		return LevelUpResult{}, err
	}
	if level >= MaxMedalLevel {
		return LevelUpResult{}, ErrAtMaxLevel
	}
	required, err := s.levelCost(level + 1)
	if err != nil {
		return LevelUpResult{}, err
	}
	needed := required - progress
	if needed < 0 {
		needed = 0
	}
	spend := min(needed, char.ShowTimePoint)
	if spend <= 0 && progress < required {
		return LevelUpResult{}, ErrInsufficientScore
	}
	oldWallet := char.ShowTimePoint
	char.ShowTimePoint -= spend
	if err := s.charRepo.Update(ctx, char); err != nil {
		char.ShowTimePoint = oldWallet
		return LevelUpResult{}, fmt.Errorf("explorermedal: spend explorer points: %w", err)
	}
	newLevel, newProgress := s.applyProgress(level, progress+spend)
	if err := s.persistState(ctx, charID, newLevel, newProgress); err != nil {
		char.ShowTimePoint = oldWallet
		if rollbackErr := s.charRepo.Update(ctx, char); rollbackErr != nil {
			return LevelUpResult{}, fmt.Errorf("%w; rollback explorer points: %v", err, rollbackErr)
		}
		return LevelUpResult{}, err
	}
	return s.result(newLevel, newProgress, char, spend, 0, newLevel != level), nil
}

func (s *Service) LevelUpByGold(ctx context.Context, charID int64, upgradeType int) (LevelUpResult, error) {
	goldCost, addedProgress, ok := goldUpgrade(upgradeType)
	if !ok {
		return LevelUpResult{}, ErrInvalidGoldType
	}
	unlock := s.lockCharacter(charID)
	defer unlock()

	level, progress, char, err := s.loadMutationState(ctx, charID)
	if err != nil {
		return LevelUpResult{}, err
	}
	if level >= MaxMedalLevel {
		return LevelUpResult{}, ErrAtMaxLevel
	}
	if char.Gold < goldCost {
		return LevelUpResult{}, ErrInsufficientGold
	}
	oldGold := char.Gold
	char.Gold -= goldCost
	if err := s.charRepo.Update(ctx, char); err != nil {
		char.Gold = oldGold
		return LevelUpResult{}, fmt.Errorf("explorermedal: spend gold: %w", err)
	}
	newLevel, newProgress := s.applyProgress(level, progress+addedProgress)
	if err := s.persistState(ctx, charID, newLevel, newProgress); err != nil {
		char.Gold = oldGold
		if rollbackErr := s.charRepo.Update(ctx, char); rollbackErr != nil {
			return LevelUpResult{}, fmt.Errorf("%w; rollback gold: %v", err, rollbackErr)
		}
		return LevelUpResult{}, err
	}
	return s.result(newLevel, newProgress, char, 0, goldCost, newLevel != level), nil
}

func (s *Service) loadMutationState(ctx context.Context, charID int64) (int, int, *domainchar.Character, error) {
	level, progress, err := s.GetState(ctx, charID)
	if err != nil {
		return 0, 0, nil, err
	}
	char, err := s.charRepo.FindByID(ctx, charID)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("explorermedal: load character: %w", err)
	}
	if char == nil {
		return 0, 0, nil, ErrCharacterNotFound
	}
	return level, progress, char, nil
}

func (s *Service) applyProgress(level, progress int) (int, int) {
	for level < MaxMedalLevel {
		cost, err := s.levelCost(level + 1)
		if err != nil || progress < cost {
			break
		}
		progress -= cost
		level++
	}
	if level >= MaxMedalLevel {
		progress = 0
	}
	return level, progress
}

func (s *Service) levelCost(level int) (int, error) {
	template := s.gameData.GetExplorerMedal(level)
	if template == nil || template.Cost <= 0 || template.Cost != float64(int(template.Cost)) {
		return 0, ErrAtMaxLevel
	}
	return int(template.Cost), nil
}

func (s *Service) persistState(ctx context.Context, charID int64, level, progress int) error {
	state := &domainfeature.CharacterFeatureState{
		CharacterID: charID,
		FeatureKey:  domainfeature.FeatureExplorerMedal,
		State:       map[string]interface{}{"info": fmt.Sprintf("%d|%d", level, progress)},
	}
	if err := s.repo.UpsertCharacterFeatureState(ctx, state); err != nil {
		return fmt.Errorf("explorermedal: persist progression: %w", err)
	}
	return nil
}

func (s *Service) result(level, progress int, char *domainchar.Character, walletSpent int, goldSpent int64, levelChanged bool) LevelUpResult {
	return LevelUpResult{
		Level: level, ExplorerPnt: char.ShowTimePoint, Progress: progress, Gold: char.Gold,
		PropMap: s.buildCumulativePropMap(level), WalletSpent: walletSpent, GoldSpent: goldSpent, LevelChanged: levelChanged,
	}
}

func (s *Service) lockCharacter(charID int64) func() {
	lock := &s.mutationLocks[uint64(charID)%uint64(len(s.mutationLocks))]
	lock.Lock()
	return lock.Unlock
}

func goldUpgrade(upgradeType int) (int64, int, bool) {
	switch upgradeType {
	case 0:
		return 50, 50, true
	case 1:
		return 500, 500, true
	default:
		return 0, 0, false
	}
}

func (s *Service) buildCumulativePropMap(level int) map[string]interface{} {
	totals := map[int]float64{}
	for i := 1; i <= level; i++ {
		if template := s.gameData.GetExplorerMedal(i); template != nil {
			addTemplateToPropMap(totals, template)
		}
	}
	result := make(map[string]interface{}, len(totals))
	for key, value := range totals {
		result[strconv.Itoa(key)] = value
	}
	return result
}

func addTemplateToPropMap(totals map[int]float64, template *models.ExplorerMedalTemplate) {
	types := []float64{template.PropType1, template.PropType2, template.PropType3, template.PropType4, template.PropType5, template.PropType6, template.PropType7, template.PropType8}
	values := []float64{template.PropNum1, template.PropNum2, template.PropNum3, template.PropNum4, template.PropNum5, template.PropNum6, template.PropNum7, template.PropNum8}
	for i, propTypeValue := range types {
		propType := int(propTypeValue)
		if propType > 0 {
			totals[propType] += values[i]
		}
	}
}

func parseInfoField(state map[string]interface{}) (level int, progress int) {
	raw, ok := state["info"]
	if !ok {
		return 0, 0
	}
	info, ok := raw.(string)
	if !ok {
		return 0, 0
	}
	parts := strings.SplitN(info, "|", 2)
	level, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
	if len(parts) == 2 {
		progress, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
	}
	if level < 0 || level > MaxMedalLevel {
		level = 0
	}
	if progress < 0 {
		progress = 0
	}
	return level, progress
}
