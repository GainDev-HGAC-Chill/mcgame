// Reconstruct sorted bag records through supported callbacks; all old-position deletes precede adds.
package item

import domainitem "mcgame-server/internal/domain/item"

func (h *Handler) refreshSortedBag(sender interface {
	SendCallback(string, ...interface{}) error
}, oldSIDs map[int64]int, itemsAfter []*domainitem.Item) error {
	for itemID, oldSID := range oldSIDs {
		if err := sender.SendCallback("onDelCharactorSlot", float64(itemID), oldSID); err != nil {
			return err
		}
	}
	for _, it := range itemsAfter {
		if err := sender.SendCallback("onAddCharactorSlot", h.itemDTO(it)); err != nil {
			return err
		}
	}
	return sender.SendCallback("onItemSort", []interface{}{})
}
