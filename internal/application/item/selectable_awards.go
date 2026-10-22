// Selectable item awards use persisted item-award rows as ordered Flash choices.
package item

import (
	"context"
	"fmt"
	"sort"

	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/gamedata/models"
)

func buildSelectableGiftBox(awards []*models.ItemAwardTemplate) (*GiftBoxData, bool, error) {
	metadataRows := 0
	for _, award := range awards {
		if award.HasSelectionMetadata() {
			metadataRows++
		}
	}
	if metadataRows == 0 {
		return nil, false, nil
	}
	if metadataRows != len(awards) {
		return nil, true, fmt.Errorf("mixed selectable and ordinary item awards")
	}

	ordered := append([]*models.ItemAwardTemplate(nil), awards...)
	sort.Slice(ordered, func(i, j int) bool {
		left, _ := ordered[i].Selection()
		right, _ := ordered[j].Selection()
		return left.Order < right.Order
	})

	choices := make([]*GiftBoxChoice, len(ordered))
	getNum := 0
	for index, award := range ordered {
		selection, ok := award.Selection()
		if !ok || selection.Order != index {
			return nil, true, fmt.Errorf("invalid selectable item award order")
		}
		if getNum == 0 {
			getNum = selection.GetNum
		} else if getNum != selection.GetNum {
			return nil, true, fmt.Errorf("inconsistent selectable item award count")
		}
		awardType := int(award.Type)
		if awardType != 12 && awardType != 19 && awardType != 29 {
			return nil, true, fmt.Errorf("unsupported selectable item award type: %d", awardType)
		}
		count := int(award.Count)
		if award.ID <= 0 || int(award.AwardID) <= 0 || count <= 0 {
			return nil, true, fmt.Errorf("invalid selectable item award")
		}
		choices[index] = &GiftBoxChoice{
			AwardRowID:  award.ID,
			TableType:   awardType,
			ID:          int(award.AwardID),
			StackNum:    count,
			Quality:     int(award.Quality),
			PreNameType: int(award.PreNameType),
			Payload:     selectableAwardPayload(award.Payload),
		}
	}
	if getNum <= 0 || getNum > len(choices) {
		return nil, true, fmt.Errorf("invalid selectable item award selection count")
	}
	return &GiftBoxData{Choices: choices, GetNum: getNum}, true, nil
}

func selectableAwardPayload(payload map[string]interface{}) map[string]interface{} {
	if len(payload) == 0 {
		return nil
	}
	result := make(map[string]interface{}, len(payload))
	for key, value := range payload {
		if key != models.AwardPayloadSelectionKey {
			result[key] = value
		}
	}
	return result
}

func (s *Service) GrantSelectableAward(ctx context.Context, char *character.Character, choice *GiftBoxChoice) (map[string]interface{}, error) {
	if s == nil || s.awardHandler == nil || char == nil || choice == nil || choice.AwardRowID <= 0 {
		return nil, fmt.Errorf("invalid selectable item award")
	}
	result, err := s.awardHandler.applySelectableChoice(ctx, char, choice)
	if err != nil {
		return nil, err
	}
	return s.effectResultToMap(result), nil
}
