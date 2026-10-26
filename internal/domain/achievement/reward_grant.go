// Open-sourced by BaoLT

// Achievement reward grants track durable claim and delivery state.
package achievement

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type RewardGrantStatus string

const (
	RewardGrantReserved  RewardGrantStatus = "reserved"
	RewardGrantGranting  RewardGrantStatus = "granting"
	RewardGrantGranted   RewardGrantStatus = "granted"
	RewardGrantFinalized RewardGrantStatus = "finalized"
	RewardGrantFailed    RewardGrantStatus = "failed"
)

type RewardGrant struct {
	CharacterID   int64
	AchievementID int64
	ClaimToken    uuid.UUID
	RewardKind    int
	TemplateID    int
	Quantity      int
	Status        RewardGrantStatus
	Receipt       json.RawMessage
	LastError     string
	LeaseExpires  *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	GrantedAt     *time.Time
	FinalizedAt   *time.Time
}

type RewardReceipt struct {
	ItemID     int64 `json:"item_id,omitempty"`
	TitleID    int   `json:"title_id,omitempty"`
	MedalSlot  int   `json:"medal_slot,omitempty"`
	MedalCount int   `json:"medal_count,omitempty"`
}
