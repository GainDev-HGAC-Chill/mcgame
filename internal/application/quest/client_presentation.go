// Quest presentation compatibility keeps server authority on canonical template types while enabling supported client actions.
package quest

import "mcgame-server/internal/gamedata/predef"

const clientQuestTypeMain = 2

func clientQuestType(templateType int) int {
	if templateType == predef.QuestTypeNewbie {
		return clientQuestTypeMain
	}
	return templateType
}
