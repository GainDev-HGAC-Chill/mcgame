// Guild-loop completion planning keeps objective selection outside the PostgreSQL command while DB revalidation remains authoritative.
package quest

import (
	"context"
	"sort"

	appitem "mcgame-server/internal/application/item"
	domainpet "mcgame-server/internal/domain/pet"
	domainquest "mcgame-server/internal/domain/quest"
	"mcgame-server/internal/gamedata/models"
	pkgerrors "mcgame-server/pkg/errors"
)

type guildLoopConsumptionPlan struct {
	items     []domainquest.GuildLoopItemConsumption
	itemPlans []appitem.PlannedConsumption
	petIDs    []int64
}

func (s *Service) planGuildLoopConsumption(ctx context.Context, qp *domainquest.QuestProgress) (*guildLoopConsumptionPlan, error) {
	plan := &guildLoopConsumptionPlan{
		items:     make([]domainquest.GuildLoopItemConsumption, 0),
		itemPlans: make([]appitem.PlannedConsumption, 0),
		petIDs:    make([]int64, 0),
	}
	if qp == nil {
		return plan, nil
	}
	type itemKey struct {
		templateID int
		tableType  int
		colorCode  int
	}
	requiredItems := make(map[itemKey]int)
	for _, objective := range qp.Objectives {
		if objective.Type == domainquest.ObjectiveCollectItem && objective.Required > 0 {
			key := itemKey{templateID: objective.Target, tableType: objective.ItemTableType, colorCode: collectObjectiveColorFilter(&objective)}
			requiredItems[key] += objective.Required
		}
	}
	if len(requiredItems) > 0 && s.itemService == nil {
		return nil, errGuildLoopRewardsUnavailable
	}
	for key, count := range requiredItems {
		planned, err := s.itemService.PlanCarriedItemConsumption(ctx, qp.CharacterID, key.templateID, count, key.tableType, key.colorCode)
		if err != nil {
			return nil, err
		}
		for _, entry := range planned {
			plan.items = append(plan.items, domainquest.GuildLoopItemConsumption{ItemID: entry.Item.ID, TemplateID: entry.Item.TemplateID, Count: entry.Count})
			plan.itemPlans = append(plan.itemPlans, entry)
		}
	}

	hasPetObjective := false
	for _, objective := range qp.Objectives {
		if objective.Type == domainquest.ObjectiveSubmitPet && objective.Required > 0 {
			hasPetObjective = true
			break
		}
	}
	if !hasPetObjective {
		return plan, nil
	}
	if s.petService == nil {
		return nil, errGuildLoopRewardsUnavailable
	}
	pets, err := s.petService.GetPetList(ctx, qp.CharacterID)
	if err != nil {
		return nil, err
	}
	selected := make(map[int64]bool)
	for _, objective := range qp.Objectives {
		if objective.Type != domainquest.ObjectiveSubmitPet || objective.Required <= 0 {
			continue
		}
		name := s.creatureTemplateName(objective.Target)
		candidates := make([]*domainpet.Pet, 0)
		for _, pet := range pets {
			if pet != nil && !selected[pet.ID] && petQualifiesForQuestSubmit(pet, &objective, name) {
				candidates = append(candidates, pet)
			}
		}
		if len(candidates) < objective.Required {
			return nil, pkgerrors.ErrInsufficientFunds
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].GrowRate != candidates[j].GrowRate {
				return candidates[i].GrowRate < candidates[j].GrowRate
			}
			return candidates[i].ID < candidates[j].ID
		})
		for i := 0; i < objective.Required; i++ {
			selected[candidates[i].ID] = true
			plan.petIDs = append(plan.petIDs, candidates[i].ID)
		}
	}
	return plan, nil
}

func guildLoopConsumedItemPayload(plan []appitem.PlannedConsumption) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(plan))
	for _, entry := range plan {
		if entry.Item == nil {
			continue
		}
		itemCopy := *entry.Item
		itemCopy.StackCount = entry.Remaining
		payload := map[string]interface{}{"deleted": entry.Remaining == 0, "id": entry.Item.ID, "sid": entry.Item.CalculateSID()}
		if entry.Remaining > 0 {
			payload["item"] = itemCopy.ToDTO()
		}
		result = append(result, payload)
	}
	return result
}

func (s *Service) completeGuildLoopChildQuest(ctx context.Context, qp *domainquest.QuestProgress, state *domainquest.LoopState, loopTpl *models.QuestLoopTemplate) (*domainquest.QuestProgress, map[string]interface{}, error) {
	commands, ok := s.loopRepo.(domainquest.GuildLoopCommandRepository)
	if !ok {
		return nil, nil, errGuildLoopRewardsUnavailable
	}
	char, err := s.GetCharacter(ctx, qp.CharacterID)
	if err != nil || char == nil {
		return nil, nil, pkgerrors.ErrNotFound
	}
	round := state.Ft + 1
	reward, err := CalculateGuildLoopReward(char.Level, round)
	if err != nil {
		return nil, nil, err
	}
	consumption, err := s.planGuildLoopConsumption(ctx, qp)
	if err != nil {
		return nil, nil, err
	}

	var nextQuest *domainquest.QuestProgress
	if round < int(loopTpl.Num) {
		child := s.pickNextChild(ctx, qp.CharacterID, state.LoopID, qp.QuestID)
		if child == nil {
			return nil, nil, errGuildLoopRewardsUnavailable
		}
		nextQuest = domainquest.NewQuestProgress(qp.CharacterID, int(child.ID), s.GetInitialObjectives(int(child.ID)))
	}
	result, err := commands.CompleteGuildLoopAtomic(ctx, domainquest.GuildLoopCompletionCommand{
		CharacterID:  qp.CharacterID,
		ProgressID:   qp.ID,
		QuestID:      qp.QuestID,
		ExpectedFt:   state.Ft,
		Money:        reward.Money,
		MoneyBind:    reward.MoneyBind,
		Contribution: reward.Contribution,
		GuildExp:     reward.GuildExp,
		Items:        consumption.items,
		PetIDs:       consumption.petIDs,
		NextQuest:    nextQuest,
	})
	if err != nil {
		return nil, nil, err
	}
	char, err = s.GetCharacter(ctx, qp.CharacterID)
	if err != nil {
		return nil, nil, err
	}
	npcIDs := []int{int(loopTpl.Nid)}
	if current := s.gameDataManager.GetQuest(qp.QuestID); current != nil {
		npcIDs = append(npcIDs, int(current.FinishNPC))
	}
	if result.NextQuest != nil {
		if next := s.gameDataManager.GetQuest(result.NextQuest.QuestID); next != nil {
			npcIDs = append(npcIDs, int(next.FinishNPC))
		}
	}
	rewards := map[string]interface{}{
		"char":              char,
		"currencyChanges":   map[string]int64{"money": reward.Money, "moneyBind": reward.MoneyBind},
		"guildContribution": reward.Contribution,
		"guildExperience":   reward.GuildExp,
		"consumedItems":     guildLoopConsumedItemPayload(consumption.itemPlans),
		"consumedPets":      result.ConsumedPetIDs,
		"loop":              map[string]interface{}{"id": state.LoopID, "ft": result.Ft},
		"loopNextQuest":     result.NextQuest,
		"loopNpcIDs":        dedupeInts(npcIDs),
		"diary":             map[string]interface{}{"id": "5", "val": int64(result.DiaryTaskCount), "act": int64(result.DiaryVitality), "num": int64(result.DiaryVitalityDelta)},
	}
	return qp, rewards, nil
}
