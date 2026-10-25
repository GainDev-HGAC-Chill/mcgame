// Open-sourced by BaoLT

// Achievement facts are durable business events waiting for idempotent evaluation.
package achievement

import (
	"encoding/json"
	"time"
)

const FactKindQuestCompleted = "quest-completed"

type Fact struct {
	FactID      string
	CharacterID int64
	Kind        string
	Payload     json.RawMessage
	OccurredAt  time.Time
}
