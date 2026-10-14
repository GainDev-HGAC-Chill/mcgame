package bloodbag

type ItemConfig struct {
	BagType int
	Amount  int64
}

func GetItemConfig(templateID int) (ItemConfig, bool) {
	switch templateID {
	case 2038:
		return ItemConfig{1, 1_000_000}, true
	case 2037:
		return ItemConfig{2, 1_000_000}, true
	case 2847:
		return ItemConfig{1, 50_000}, true
	case 2846:
		return ItemConfig{2, 50_000}, true
	default:
		return ItemConfig{}, false
	}
}
