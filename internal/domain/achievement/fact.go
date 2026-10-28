// Open-sourced by BaoLT

// Achievement facts are durable business events waiting for idempotent evaluation.
package achievement

import (
	"encoding/json"
	"time"
)

const (
	FactKindQuestCompleted              = "quest-completed"
	FactKindPetCaptured                 = "pet-captured"
	FactKindWishPetDrawn                = "wish-pet-drawn"
	FactKindCharacterRenamed            = "character-renamed"
	FactKindMysteryMerchantPetPurchased = "mystery-merchant-pet-purchased"
)

type PetCapturedFactPayload struct {
	PetID              int64 `json:"pet_id"`
	CreatureTemplateID int64 `json:"creature_template_id"`
}

type WishPetDrawnFactPayload struct {
	DrawID             string  `json:"draw_id"`
	Ordinal            int     `json:"ordinal"`
	PlanID             int64   `json:"plan_id"`
	CreatureTemplateID int64   `json:"creature_template_id"`
	GrowRate           float64 `json:"grow_rate"`
	ColorTier          int     `json:"color_tier"`
}

type CharacterRenamedFactPayload struct {
	OperationID string `json:"operation_id"`
	CardItemID  int64  `json:"card_item_id"`
	NewName     string `json:"new_name"`
}

type MysteryMerchantPetPurchasedFactPayload struct {
	PurchaseID         string         `json:"purchase_id"`
	PetID              int64          `json:"pet_id"`
	CreatureTemplateID int64          `json:"creature_template_id"`
	GrowRate           float64        `json:"grow_rate"`
	ColorTier          int            `json:"color_tier"`
	MerchantID         int64          `json:"merchant_id"`
	OfferID            int64          `json:"offer_id"`
	Price              map[string]int `json:"price"`
}

type Fact struct {
	FactID      string
	CharacterID int64
	Kind        string
	Payload     json.RawMessage
	OccurredAt  time.Time
}
