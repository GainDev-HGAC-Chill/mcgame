// Reviewed class-rank requirements are exact catalog policy data.
package quest

import (
	"strconv"
	"strings"

	"mcgame-server/internal/gamedata/models"
)

type classRankQuestSpec struct {
	ClassID   int
	ClassRank int
	StartNPC  int
	FinishNPC int
	Name      string
}

var reviewedClassRankQuests = map[int]classRankQuestSpec{
	143: {ClassID: 1, ClassRank: 1, StartNPC: 339, FinishNPC: 600, Name: "Thăng Cấp Trung Cấp 3-1"},
	287: {ClassID: 2, ClassRank: 1, StartNPC: 341, FinishNPC: 600, Name: "Thăng Cấp Trung Cấp 3-1"},
	639: {ClassID: 3, ClassRank: 1, StartNPC: 342, FinishNPC: 600, Name: "Thăng Cấp Trung Cấp 3-1"},
	640: {ClassID: 4, ClassRank: 1, StartNPC: 490, FinishNPC: 600, Name: "Thăng Cấp Trung Cấp 3-1"},
	643: {ClassID: 5, ClassRank: 1, StartNPC: 426, FinishNPC: 600, Name: "Thăng Cấp Trung Cấp 3-1"},
	644: {ClassID: 6, ClassRank: 1, StartNPC: 640, FinishNPC: 600, Name: "Thăng Cấp Trung Cấp 3-1"},
}

func classRankQuestFingerprintMatches(tpl *models.QuestTemplate, spec classRankQuestSpec) bool {
	return tpl != nil &&
		tpl.Type == 1 &&
		tpl.StartNPC == float64(spec.StartNPC) &&
		tpl.FinishNPC == float64(spec.FinishNPC) &&
		tpl.Name == spec.Name &&
		tpl.ReqClass == "|"+strconv.Itoa(spec.ClassID)+"|" &&
		strings.TrimSpace(tpl.StartText) != "" &&
		strings.TrimSpace(tpl.CompleteText) != ""
}
