// Guild-loop completion commands describe one atomic consume, reward, and advance operation.
package quest

import "context"

type GuildLoopItemConsumption struct {
	ItemID     int64 `json:"itemId"`
	TemplateID int   `json:"templateId"`
	Count      int   `json:"count"`
}

type GuildLoopItemState struct {
	ItemID    int64 `json:"itemId"`
	Remaining int   `json:"remaining"`
}

type GuildLoopCompletionCommand struct {
	CharacterID  int64
	ProgressID   int64
	QuestID      int
	ExpectedFt   int
	Money        int64
	MoneyBind    int64
	Contribution int64
	GuildExp     int64
	Items        []GuildLoopItemConsumption
	PetIDs       []int64
	NextQuest    *QuestProgress
}

type GuildLoopCompletionResult struct {
	GuildID            int64
	Money              int64
	MoneyBind          int64
	Gold               int64
	GuildContrib       int64
	DonateContrib      int64
	GuildRestoreNormal int64
	GuildRestoreDonate int64
	MemberNormal       int64
	MemberTotal        int64
	MemberWeekly       int64
	GuildExperience    int64
	Ft                 int
	Active             bool
	ItemStates         []GuildLoopItemState
	ConsumedPetIDs     []int64
	NextQuest          *QuestProgress
	DiaryTaskCount     int
	DiaryVitality      int
	DiaryVitalityDelta int
}

type GuildLoopCommandRepository interface {
	CompleteGuildLoopAtomic(ctx context.Context, command GuildLoopCompletionCommand) (*GuildLoopCompletionResult, error)
}
