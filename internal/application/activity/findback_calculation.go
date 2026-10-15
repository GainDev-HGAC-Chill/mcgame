package activity

import (
	"math"
	"mcgame-server/internal/gamedata/predef"
)

// FindbackRewardBase80 stores UI-calibrated reward EXP at level 80.
// These values are proposed balance constants, NOT discovered original-server rules.
// Class reward is the sum for TWO missed class quests.
var FindbackRewardBase80 = map[string]int64{
	"xiuxing": 4998510, "xueyuan": 3366000, "chongwu": 351399,
	"panjun": 2360407, "feimo": 2082712,
}

// FindbackProposedExperience scales the screenshot's level-80 reward by the
// server's BasicGetExp table. Returned EXP is before the 70%/100% choice.
// IMPORTANT: This is a proposal only. A separate authoritative tracker must
// establish missed activity counts before any actual claim can be offered.
func FindbackProposedExperience(kind string, level int) int64 {
	if level < 1 || level > 160 {
		return 0
	}
	if kind == "xuanshang" {
		return predef.CalcCallboardExp(level, 0)
	}
	base, ok := FindbackRewardBase80[kind]
	if !ok {
		return 0
	}
	anchor := predef.BasicGetExp[80]
	if anchor <= 0 {
		return 0
	}
	return int64(math.Round(float64(base) * float64(predef.BasicGetExp[level]) / float64(anchor)))
}

// FindbackAwardedExperience calculates a mode-specific XP amount.
// mode 1: 70% (silver); mode 2: 100% (gold).
func FindbackAwardedExperience(base int64, mode int) int64 {
	if base <= 0 {
		return 0
	}
	switch mode {
	case 1:
		return int64(math.Floor(float64(base) * 0.7))
	case 2:
		return base
	}
	return 0
}

// FindbackCost computes the exact silver/gold cost based on original SWF constants.
// Available missing attempts must be calculated from authoritative server records.
func FindbackCost(kind string, mode, missing int) int64 {
	if missing <= 0 {
		return 0
	}
	silver := map[string]int64{"xiuxing": 200000, "xueyuan": 100000, "chongwu": 400000, "panjun": 300000, "feimo": 300000, "xuanshang": 0, "xuanshang_banner": 100000}
	gold := map[string]int64{"xiuxing": 10, "xueyuan": 5, "chongwu": 10, "panjun": 30, "feimo": 30, "xuanshang": 5, "xuanshang_banner": 5}
	if mode == 1 {
		return silver[kind] * int64(missing)
	}
	if mode == 2 {
		return gold[kind] * int64(missing)
	}
	return 0
}
