// Equipment inventory reconstruction is owner-only and independent of scene appearance callbacks.
package item

import domainitem "mcgame-server/internal/domain/item"

func (h *Handler) refreshEquipmentSlots(sender interface {
	SendCallback(string, ...interface{}) error
}, oldSIDs map[int64]int, items []*domainitem.Item) error {
	for id, sid := range oldSIDs {
		if err := sender.SendCallback("onDelCharactorSlot", float64(id), sid); err != nil {
			return err
		}
	}
	for _, it := range items {
		dto := h.itemDTO(it)
		if err := sender.SendCallback("onChangeEquiptInst", dto); err != nil {
			return err
		}
		if err := sender.SendCallback("onAddCharactorSlot", dto); err != nil {
			return err
		}
	}
	return nil
}
