// The local server starter package is applied before the new character is persisted.
// Loading or selecting existing characters never grants this package again.
package character

import "mcgame-server/internal/domain/character"

const (
	starterLevel = character.DefaultCreationLevel
)

func applyStarterRewards(char *character.Character, defaults character.NewCharacterDefaults) {
	char.RaiseLevelTo(defaults.Level)
	char.Gold = defaults.Gold
	char.GoldBind = defaults.GoldBind
}
