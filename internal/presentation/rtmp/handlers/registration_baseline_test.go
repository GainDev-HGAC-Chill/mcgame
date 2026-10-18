package handlers

import (
	"context"
	"testing"

	appdress "mcgame-server/internal/application/dress"
	appmonsterheart "mcgame-server/internal/application/monsterheart"
	apppetarena "mcgame-server/internal/application/petarena"
	appprs "mcgame-server/internal/application/prs"
	domainfeature "mcgame-server/internal/domain/statfeature"
	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/infrastructure/rtmp"
	"mcgame-server/internal/presentation/rtmp/handlers/activity"
	dresshandler "mcgame-server/internal/presentation/rtmp/handlers/dress"
	farmhandler "mcgame-server/internal/presentation/rtmp/handlers/farm"
	monsterhearthandler "mcgame-server/internal/presentation/rtmp/handlers/monsterheart"
	petarenahandler "mcgame-server/internal/presentation/rtmp/handlers/petarena"
	prshandler "mcgame-server/internal/presentation/rtmp/handlers/prs"
	pkgerrors "mcgame-server/pkg/errors"

	"go.uber.org/zap"
)

type registrationFeatureRepo struct{}

func (registrationFeatureRepo) ListCharacterFeatureStates(context.Context, int64) ([]*domainfeature.CharacterFeatureState, error) {
	return nil, nil
}

func (registrationFeatureRepo) UpsertCharacterFeatureState(context.Context, *domainfeature.CharacterFeatureState) error {
	return nil
}

func TestLegacyFeatureCombinedDispatcherRegistrationBaseline(t *testing.T) {
	logger := zap.NewNop()
	dispatcher := rtmp.NewRPCDispatcher(logger)

	farmhandler.NewHandler(nil, nil, nil, nil, logger).RegisterHandlers(dispatcher)
	gameData := gamedata.NewManager(nil, logger)
	dressService := appdress.NewService(nil, gameData, logger)
	dressRPCHandler := dresshandler.NewHandler(dressService, nil, nil, nil, logger)
	dressRPCHandler.SetDecoHoleService(appdress.NewDecoHoleService(registrationFeatureRepo{}, nil, gameData, logger))
	dressRPCHandler.RegisterHandlers(dispatcher)
	petArenaService := apppetarena.NewService(nil, nil, logger)
	petarenahandler.NewHandler(petArenaService, logger).RegisterHandlers(dispatcher)
	prshandler.NewHandler(appprs.NewService(nil, nil, nil, logger), logger).RegisterHandlers(dispatcher)
	monsterhearthandler.NewHandler(appmonsterheart.NewService(nil, logger), logger).RegisterHandlers(dispatcher)
	activity.NewHandler(nil, logger).RegisterHandlers(dispatcher)

	registered := map[string][]string{
		"monsterheart": {
			"initMonsterHeartData", "monsterHeartSet", "monsterHeartReMove", "openHoleMonsterHeart",
			"openHoleMonsterHeartJingang", "upgradeMonsterHeartItem", "resolveMonsterHeart", "upgradeMonsterHeartBox",
		},
		"card-stub": {"initCardPlayPanel", "startGame", "changeCard", "getCardAward", "addPlayTime"},
		"sweep-stub": {
			"initTaskSweepPanelByClient", "getTaskBattleTime", "taskSweepSure", "taskSweepCancel",
			"taskSweepEnd", "taskSweepLeftTimeCheck", "taskFinishByGold",
		},
		"farm": {
			"getFarmByCid", "getFriendFarm", "addMineral", "addFarmNum", "farmLvUp",
			"harvestMine", "steelMine", "getFarmBag", "getFarmBagAll", "getFromFarmBag",
			"getFarmLog", "getReplayList", "saveReplay", "delReplay", "queryPlant",
		},
		"dress-deco": {"getDecoSuitProp", "addDecoHoleLevel"},
		"petarena":   {"getPetArenaConf", "getPetConf", "petFightConf", "petArenaTicket"},
		"findback": {
			"getFindBackData", "findBackXiuXing", "findBackXueYuan", "findBackChongWu",
			"findBackPanJun", "findBackFeiMo", "findBackXuanShangFree", "findBackXuanShangByBanner",
		},
		"hmtxls": {"getHMTXLSData", "enterPubRoom", "enterAdvRoom", "endHMTXLRoom", "getHMTAward"},
		"ppve": {
			"initPPVEPanel", "getPPVERank", "challengeNextFloor", "savePPVEConf",
			"increaseChallengeTimeByGold", "exchangeKP", "exchangeKPByGold",
		},
		"prs": {"initPRSPanel", "addSoulTreeLvl", "activePRSShowSpe"},
	}

	for family, methods := range registered {
		for _, method := range methods {
			t.Run(family+"/"+method, func(t *testing.T) {
				_, err := dispatchWithCharacter(dispatcher, method, "", nil)
				if pkgerrors.Is(err, pkgerrors.ErrMethodNotFound) {
					t.Fatalf("%s did not resolve after combined registration", method)
				}
			})
		}
	}

	t.Run("real handlers survive trailing activity registration", func(t *testing.T) {
		monsterResult, err := dispatchWithCharacter(dispatcher, "initMonsterHeartData", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		monsterPanel, ok := monsterResult.(map[string]interface{})
		if !ok || monsterPanel["data"] == nil || monsterPanel["bag"] == nil {
			t.Fatalf("monster heart resolved to unexpected handler result: %#v", monsterResult)
		}

		decoResult, err := dispatchWithCharacter(dispatcher, "getDecoSuitProp", "1", []interface{}{float64(1), float64(0), float64(0)})
		if err != nil {
			t.Fatal(err)
		}
		decoPanel, ok := decoResult.(map[string]interface{})
		if !ok || decoPanel["actFlag"] != false || decoPanel["suitIdObj"] == nil {
			t.Fatalf("decoration resolved to unexpected handler result: %#v", decoResult)
		}

		prsResult, err := dispatchWithCharacter(dispatcher, "initPRSPanel", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		prsPanel, ok := prsResult.(map[string]interface{})
		if !ok || prsPanel["tid"] != "0" || prsPanel["actArr"] == nil {
			t.Fatalf("PRS resolved to unexpected handler result: %#v", prsResult)
		}

		_, err = dispatchWithCharacter(dispatcher, "getFarmByCid", "invalid", nil)
		if !pkgerrors.Is(err, pkgerrors.ErrSystemError) {
			t.Fatalf("getFarmByCid no longer resolves to the farm handler: %v", err)
		}
		_, err = dispatchWithCharacter(dispatcher, "getPetConf", "invalid", nil)
		if !pkgerrors.Is(err, pkgerrors.ErrUnauthorized) {
			t.Fatalf("getPetConf no longer resolves to its authenticated handler: %v", err)
		}
		for _, method := range []string{"getFindBackData", "getHMTXLSData", "initPPVEPanel"} {
			_, err := dispatchWithCharacter(dispatcher, method, "invalid", nil)
			if !pkgerrors.Is(err, pkgerrors.ErrInvalidInput) {
				t.Fatalf("%s no longer resolves to its authenticated handler: %v", method, err)
			}
		}
	})

	t.Run("card remains explicit compatibility stub", func(t *testing.T) {
		result, err := dispatchWithCharacter(dispatcher, "initCardPlayPanel", "7", nil)
		if err != nil {
			t.Fatal(err)
		}
		panel, ok := result.(map[string]interface{})
		if !ok || panel["flag"] != true || panel["max"] != 0 || panel["now"] != 0 || panel["changeTime"] != 0 {
			t.Fatalf("unexpected card starter payload: %#v", result)
		}
	})

	t.Run("sweep RPCs fail closed without method errors", func(t *testing.T) {
		methods := []string{
			"initTaskSweepPanelByClient", "getTaskBattleTime", "taskSweepSure",
			"taskSweepCancel", "taskSweepEnd", "taskSweepLeftTimeCheck", "taskFinishByGold",
		}
		for _, method := range methods {
			result, err := dispatchWithCharacter(dispatcher, method, "7", nil)
			if err != nil || result != nil {
				t.Fatalf("%s result=%#v err=%v, want fail-closed nil", method, result, err)
			}
		}
	})
}

func dispatchWithCharacter(dispatcher *rtmp.RPCDispatcher, method, characterID string, args []interface{}) (interface{}, error) {
	conn := rtmp.NewConnection(1, nil, dispatcher, zap.NewNop())
	if characterID != "" {
		conn.SetCharacter(characterID)
	}
	return dispatcher.Dispatch(conn, method, args)
}
