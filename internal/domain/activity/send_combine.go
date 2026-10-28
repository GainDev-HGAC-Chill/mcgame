// Send Combine commands describe one durable atomic purchase and its committed state.
package activity

import (
	"context"

	"mcgame-server/internal/domain/item"
)

type SendCombineAward struct {
	TemplateID int           `json:"templateId"`
	ItemType   item.ItemType `json:"itemType"`
	Count      int           `json:"count"`
	IsBound    bool          `json:"isBound"`
	ColorCode  int           `json:"colorCode"`
}

type SendCombinePurchaseCommand struct {
	CharacterID int64
	ActivityID  string
	EventEndMs  int64
	Limit       int
	Cost        int64
	Awards      []SendCombineAward
}

type SendCombinePurchaseResult struct {
	BuyCount int
	Gold     int64
	Items    []*item.Item
}

type SendCombineRepository interface {
	PurchaseSendCombine(context.Context, SendCombinePurchaseCommand) (*SendCombinePurchaseResult, error)
	ListSendCombineBuyCounts(context.Context, int64, int64) (map[string]int, error)
}
