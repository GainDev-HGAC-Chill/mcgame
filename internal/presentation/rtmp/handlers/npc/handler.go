// Open-sourced by BaoLT

// NPC RTMP handlers.
package npc

import (
	"context"
	"sync"

	appchar "mcgame-server/internal/application/character"
	appcombat "mcgame-server/internal/application/combat"
	appfairy "mcgame-server/internal/application/fairy"
	appfarm "mcgame-server/internal/application/farm"
	appgathering "mcgame-server/internal/application/gathering"
	appgroup "mcgame-server/internal/application/group"
	appguild "mcgame-server/internal/application/guild"
	appitem "mcgame-server/internal/application/item"
	apppet "mcgame-server/internal/application/pet"
	appquest "mcgame-server/internal/application/quest"
	appscene "mcgame-server/internal/application/scene"
	appskill "mcgame-server/internal/application/skill"
	"mcgame-server/internal/domain/auth"
	"mcgame-server/internal/gamedata"
	redisstore "mcgame-server/internal/infrastructure/persistence/redis"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type Handler struct {
	sceneService        *appscene.Service
	charService         *appchar.Service
	combatService       *appcombat.Service
	questService        *appquest.Service
	farmService         *appfarm.Service
	gatheringService    *appgathering.Service
	sceneManager        *rtmp.SceneManager
	rtmpServer          *rtmp.Server
	presenceStore       *redisstore.SocialPresenceStore
	gameData            *gamedata.Manager
	guildService        *appguild.Service
	groupService        *appgroup.Service
	itemService         *appitem.Service
	petService          *apppet.Service
	skillService        *appskill.Service
	fairyService        *appfairy.Service
	accountRepo         auth.AccountRepository
	farmHarvestMu       sync.Mutex
	farmHarvests        map[uint32]*farmHarvestTask
	fishingTaskMu       sync.Mutex
	fishingTasks        map[uint32]*fishingTask
	herbTaskMu          sync.Mutex
	herbTasks           map[uint32]*herbTask
	groupStateRefresher func(ctx context.Context, groupID int64)
	scheduleBossClick   rtmp.HandlerFunc
	weddingBookProvider weddingBookProvider
	logger              *zap.Logger
}

type weddingBookProvider interface {
	SendWeddingBookList(ctx context.Context, connection *rtmp.Connection) error
}

func NewHandler(sceneService *appscene.Service, charService *appchar.Service, questService *appquest.Service, sceneManager *rtmp.SceneManager, accountRepo auth.AccountRepository, logger *zap.Logger) *Handler {
	return &Handler{
		sceneService: sceneService,
		charService:  charService,
		questService: questService,
		sceneManager: sceneManager,
		accountRepo:  accountRepo,
		farmHarvests: make(map[uint32]*farmHarvestTask),
		fishingTasks: make(map[uint32]*fishingTask),
		herbTasks:    make(map[uint32]*herbTask),
		logger:       logger,
	}
}

func (h *Handler) SetGameDataManager(manager *gamedata.Manager) {
	h.gameData = manager
}

func (h *Handler) SetCombatService(combatService *appcombat.Service) {
	h.combatService = combatService
}

func (h *Handler) SetGroupService(groupService *appgroup.Service) {
	h.groupService = groupService
}

func (h *Handler) SetGroupStateRefresher(fn func(ctx context.Context, groupID int64)) {
	h.groupStateRefresher = fn
}

func (h *Handler) SetGuildService(guildService *appguild.Service) {
	h.guildService = guildService
}

func (h *Handler) SetItemService(itemService *appitem.Service) {
	h.itemService = itemService
}

func (h *Handler) SetPetService(petService *apppet.Service) {
	h.petService = petService
}

func (h *Handler) SetSkillService(skillService *appskill.Service) {
	h.skillService = skillService
}

func (h *Handler) SetFarmService(farmService *appfarm.Service) {
	h.farmService = farmService
}

func (h *Handler) SetGatheringService(gatheringService *appgathering.Service) {
	h.gatheringService = gatheringService
}

func (h *Handler) SetFairyService(fairyService *appfairy.Service) {
	h.fairyService = fairyService
}

func (h *Handler) SetRTMPServer(rtmpServer *rtmp.Server) {
	h.rtmpServer = rtmpServer
}

func (h *Handler) SetPresenceStore(store *redisstore.SocialPresenceStore) {
	h.presenceStore = store
}

func (h *Handler) SetScheduleBossClickHandler(handler rtmp.HandlerFunc) {
	h.scheduleBossClick = handler
}

func (h *Handler) SetWeddingBookProvider(provider weddingBookProvider) {
	h.weddingBookProvider = provider
}

func (h *Handler) sendSceneResetCallbacks(conn *rtmp.Connection) error {
	if err := conn.SendCallback("onCreateNpcs", map[string]interface{}{}); err != nil {
		return err
	}
	return conn.SendCallback("onCreateBoss", map[string]interface{}{})
}

func (h *Handler) RegisterHandlers(dispatcher *rtmp.RPCDispatcher) {
	dispatcher.Register("clickNpc", h.ClickNpc)
	dispatcher.Register("npcScript", h.NpcScript)
	dispatcher.Register("npcFuncClick", h.NpcFuncClick)
	dispatcher.Register("npcFuncOther", h.NpcFuncOther)
	dispatcher.Register("doDoubleHarvest", h.DoDoubleHarvest)
	dispatcher.Register("ripenImmediately", h.RipenImmediately)
	dispatcher.Register("uproot", h.Uproot)
	dispatcher.Register("onProductReady", h.OnProductReady)
	dispatcher.Register("onProductCancel", h.OnProductCancel)
	dispatcher.Register("queryFishPool", h.QueryFishPool)
	dispatcher.Register("queryHerb", h.QueryHerb)
	dispatcher.Register("queryGather", h.QueryGather)
	dispatcher.Register("quitGatheringClinet", h.QuitGatheringClinet)
}
