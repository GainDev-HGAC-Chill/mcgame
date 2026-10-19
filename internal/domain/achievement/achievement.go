// Open-sourced by BaoLT

// Achievement domain models represent per-character progress and login snapshots.
package achievement

import "time"

type Progress struct {
	ID            int64
	CharacterID   int64
	AchievementID int64
	Progress      int
	Target        int
	IsCompleted   bool
	IsClaimed     bool
	CompletedAt   *time.Time
	ClaimedAt     *time.Time
}

type CompletionEntry struct {
	AchievementID int64 `json:"achievement_id"`
	DoneAt        int64 `json:"done_at"`
}

type RequirementProgress struct {
	CharacterID   int64     `json:"character_id"`
	RequirementID int64     `json:"requirement_id"`
	AchievementID int64     `json:"achievement_id"`
	Done          bool      `json:"done"`
	Progress      int       `json:"progress"`
	Target        int       `json:"target"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type RequirementUpdate struct {
	RequirementID int64 `json:"requirement_id"`
	AchievementID int64 `json:"achievement_id"`
	Delta         int   `json:"delta,omitempty"`
	Absolute      *int  `json:"absolute,omitempty"`
	Target        int   `json:"target"`
	Done          bool  `json:"done,omitempty"`
}

type ProgressUpdate struct {
	Requirements []*RequirementProgress `json:"requirements"`
	Completed    []CompletionEntry      `json:"completed"`
}

type Snapshot struct {
	AchieveLog          map[string]interface{}
	AchieveReqLog       map[string]interface{}
	TakeAchieveAwardLog map[string]interface{}
	AchievementPoints   int
}
