# NPC & Quest System

## Reviewed dialogue registry expansion (2026-10-14)

Dialogue-only completion is now authorized by a 227-entry fingerprint registry: 67 type-1 quests, 67 type-2 quests, and 93 type-4 quests. Each entry records the exact checked `TBL_QUEST.json` ID, name, type, start NPC, and finish NPC. Runtime also requires no ordinary requirement rows, zero `rt/rn`, non-empty start and complete dialogue, and a positive finish NPC. Any fingerprint or dialogue drift fails closed as unconfigured; no Vietnamese text scanning or broad empty-quest inference occurs. Operator-confirmed quest 46 (`Chế Tạo Huy Hiệu`, NPC 430) remains authorized.

The existing lifecycle is unchanged: canonical objectives stay empty, readiness changes from state 103 to 104 only within the server-owned 300-unit finish-NPC map/proximity gate, and final turn-in checks location again before one-shot rewards. The 711 ambiguous candidates, 77 contradicted candidates, and 669 other empty definitions remain unauthorized by this change.

`cmd/questaudit` now joins the compose snapshot with checked quest/NPC catalogs and reports registry counts, fingerprint drift, missing finish NPCs, ordinary-requirement or `rt/rn` overlap, missing dialogue, and the exact policy partition. The checked snapshot reports 227 reviewed entries with a 67/67/93 split and zero registry diagnostics. Current partition: 3,682 catalog, 227 reviewed conversations, 90 implemented specials, 4 missing-target, 215 unsupported currency, and 1,312 unconfigured definitions. Focused application/quest-handler/NPC-handler/audit tests, targeted vet, and both `questaudit`/`gameserver` builds pass. Broad internal regression has only the known four Pet stat/equipment/skill and two chat-stat failures; quest packages pass. No migration, player-data change, deployment, or restart was performed.

## Capture quest refresh (2026-10-13)

Successful pet capture now triggers `QuestNotifyHelper.PushQuestProgress` from the combat turn handler when a new pet reward belongs to the current character. This refreshes submit-pet objectives immediately instead of leaving the Flash quest panel at `0/1` until a new quest snapshot is requested. The change is tested and built locally but has not been deployed because the user chose not to restart `mcgame-server`.

Latest deployment17:34:09 (2026-10-12): sort-field and reward-lock repairs included; game running/restart_count0,5530 quests loaded. Prior12:59 dependency-startup blocker is historical, not current status. Flash post-deploy validation NOT TESTED; no old quest/reward replay. [Batch results](../research/2026-10-12_09_PANEL_PROTOCOL_ECONOMY_TRIAGE_RESEARCH.md).

## QuestManager missing leading sort field repair (2026-10-12)

The earlier shop envelope repair was necessary but incomplete. `EnrichQuestDTO` supplied `data` without `color`; `DataManager.getChaQuestFullData` overlays every supplied property onto catalog full data, so that partial map replaces the client template, erasing its color. A second quest in the same type/category makes Flex's sorted insertion throw **Find criteria must contain all sort fields leading up to sort2**. This affects both Khiêu chiến chiêu tài bảo progress and mall-purchase refreshes.

Read-only served-SWF exports: existing `.local/research/mcgame-callback-as-research/scripts/com/qeedoo/game/system/CallBack.as:11373–11378` and newly exported complete classes under `C:/Users/gaingain/AppData/Local/Temp/opencode/mcgame-quest-sort-research/scripts/` from `.local/research/mcgame-main-research.swf`. Exact active-tree fields (`com/qeedoo/ui/view/compDragable/QuestManager.as:661–672`): `sort1 = quest.data.color`, then `sort2 = quest.state`; both use `new SortField(name,true,true)` (caseInsensitive=true, descending=true, numeric omitted/null). `mx/collections/SortField.as:291–306` therefore selects string comparison even though the payload values are numeric AMF0 Numbers. The available-quest tree instead uses template `color` then `minLevel`. `mx/collections/Sort.as:263–279` checks `values[field] !== undefined` and rejects a defined second field following an undefined first field. Zero color is valid and must be explicitly present. Do not add top-level `sort1`/`sort2` keys or replace catalog color with callboard `c`.

Server fix: enriched `data.color = int(questTpl.Color)` alongside `data.type`. Existing numeric `state` (103/104) and catalog `c` remain unchanged. Shared service enrichment reaches initial manager snapshots, acceptance, combat notifications and `QuestNotifyHelper.PushQuestProgress` without inventing objectives/rewards or changing callback method names. Complete `QuestManager.onAddChaQuest:545–585` deep-copies then assigns `player.questList[data.id] = quest`: repeated adds are replacement/upsert, not duplicate list entries, and retain the existing guide entry. Complete callback export has no `onUpdateChaQuest`; `onUpdateQuestKill` is creature-delta progress routed to the tracker, not a replacement full quest DTO. Keep full refreshes on `onAddChaQuest` with `d`, `qn`, `pos`, `clsData`; acquisition messages from repeated adds are existing client behavior, not a new update API.

Regressions: `presentation_sort_test.go` covers template colors 0/1/4 and quest types 1/9/10; `notify_sort_test.go` covers acceptance and repeated shop-style notifier refreshes with two same-type quests, updated completion state, replacement semantics, and initial snapshots. `go test ./internal/application/quest ./internal/presentation/rtmp/handlers/quest ./internal/presentation/rtmp/handlers/combat ./internal/presentation/rtmp/handlers/item -count=1` and scoped `go vet` PASS. No AS/client edits, player actions, DB changes, deployment or commit. Live Flash retest remains NOT TESTED; already-running clients may still hold old partial quest data until a fixed full refresh/relogin.

## Reward-item re-entry freeze repair (2026-10-12)

Live log: character3 finishQuest6 at12:41:11 reaches item758 reward creation without returning. Quest CompleteQuest holds a character mutation mutex; item AddItemWithBindAndColor synchronously calls quest.OnItemCollected through the wired item service, reacquiring the same non-reentrant mutex. Earlier reward unit tests omitted that back-reference. CompleteQuest now queues its own synchronous item notifications in a private context scope and flushes them after releasing the lock, for normal and loop completion; other character/services and external item events are not suppressed. In-memory bidirectional-service regression verifies bounded completion, progress of another active collect quest, and no repeated reward. OnLevelUp now uses non-persisting authoritative sync, not caller-provided level or stale active snapshot writeback. Rejected finish responses use Flash info (not msg); awaited finishQuest rate-limit/dispatch errors return flag=false/info instead of disappearing or faulting. Remaining quests receive standard progress envelopes after reward-slot updates. No historical reward repair/replay; the already interrupted quest may have partial reward writes and needs separate diagnosis before any recovery. Focused tests/vet/build PASS; rebuilt game image, but startup BLOCKED because dependencies were already stopped at12:42. Stopped only the new failing game container; await permission to start existing stack. Flash retest NOT TESTED. [Follow-up](../research/2026-10-12_08_GAMEPLAY_FAILURES_FOLLOWUP_RESEARCH.md).

## Full-catalog completion authority repair (2026-10-12)

Supersedes the type10-only guard. Audited all5530 templates:3846 have ordinary rows,1684 do not. Final policy:3682 ordinary valid-target;66 reviewed type1 NPC conversations;14 character-level,14 pet-level,23 pet-count,39 friend conditions;1493 unclassified,195 currency and4 missing-target templates remain **blocked/not implemented**, not automatically completed. Per-ID artifact `.local/research/quest-completion-policy-20261012.json`; reproducible `cmd/questaudit` plus `supabase/data/audit_quest_requirements.sql`.

Acceptance ignores client objective/current data; callboard uses catalog rather than invented defaults. Active canonical definitions are rebuilt, exact matching kill progress retained; reviewed specials discard old synthetic objectives. Dependencies, targets, expiry and real guild membership fail closed. UI sync does not persist/overwrite completed status. FinishQuest requires actual server map/proximity at positive finishNPC; reviewed conversation readiness has the same NPC check (local radius300). Mutations serialized per character. Kill cleanup validates membership and inflight claims; existing objective JSON persists per-enemy `killCreditEvents` for prefix-safe retries. Combat callbacks use shared BuildAddQuestPayload; fake npcScript test quest removed. No player/history reset or reward replay. Tests/vet/build/deploy PASS; startup12:34:08 running/restart_count0,5530 quests/itemAwards71. Full suite retains pet/auth/chat failures, race tests blocked, Flash retest NOT TESTED. [Full audit and limitations](../research/2026-10-12_06_ALL_QUEST_REQUIREMENTS_AUDIT_RESEARCH.md).

## Friendship and empty grow-up requirements (2026-10-12)

Confirmed GainTest completed2695/2696 with objectives=[] and zero friends;2697 active also empty. Type10=client growup, not an ordinary talk quest. Live211 growup templates,137 without ordinary requirement rows. Quest service now fails closed for active empty growup quests and explicitly verifies catalog chain2695–2733 using real distinct friends and all intimacy>=50, based on name/info (original script unavailable). Same verification affects domain completion, enriched UI state and CompleteQuest reward gate. Dependency errors never allow rewards; historical completed quests/player rewards untouched. Runtime CompletionBlocked excluded from JSON/DB; ordinary NPC conversation semantics unchanged. Other empty growup subsystems remain unsupported, not auto-completed. Tests/vet/build/deploy PASS; startup08:52:08 running/restart_count0. Flash retest remains NOT TESTED. [Research](../research/2026-10-12_04_QUEST_REQUIREMENTS_FRIENDSHIP_RESEARCH.md).

## Shop-triggered quest progress envelope (2026-10-12)

QuestNotifyHelper.PushQuestProgress formerly sent raw enriched DTO through onAddChaQuest. Actual client unconditionally dereferences payload.d.pos/clsData, causing Error1010 after successful shop purchase. It now reuses the quest-acceptance envelope builder (qid/st/d/qn plus original fields); no synthetic quests/rewards/progress. Shared builder tests and quest/item suites/vet/build PASS; live purchase callback retest pending. Existing completed purchases must not be replayed. [Research](../research/2026-10-12_03_AUTO_EQUIP_SHOP_QUEST_MISSING_RPC_RESEARCH.md).

## NPC & Boss System

NPC handlers are in their own `npc` package (`internal/presentation/rtmp/handlers/npc/`) with dedicated files: `handler.go`, `npc_click.go`, `npc_interaction.go`, `npc_script.go`, `npc_func_other.go`.

NPC Classification (4 legacy types): NPCTypeNormal=1 (attackable), NPCTypeShop=2 (shop), NPCTypeQuest=3 (quest giver), NPCTypeBoss=4 (boss). Full NPCType enum has 29 types (see `architecture-protocol.md`).

Spawn Mechanisms: Static spawn (loaded from `player.npcs` table at map entry, sent via `createNpcs`). Dynamic spawn (boss by respawn timer, creatures by spawn rate from `TBL_MAP_CREATURE`).

MapCreatureTemplate fields: ID, Mid (map ID), Cid (creature ID), Level, BossFlag (1=boss), StartRate/EndRate (spawn percentage range), Exp, ExpMulti, BattleMode, UniqueFlag, QLevel.

CreatureTemplate fields: ID, Name, ClassID, QLevel, UseLv, attributes, aptitudes, combat properties, Skills (pipe-delimited), Catchable, GrowBase, Life, visual codes.

Combat Flow: Click NPC (`clickNpc` RPC) -> server checks type -> `hitNpc` RPC -> server creates battle -> turn-based processing -> rewards on victory.

## NPC Handler RPCs

`clickNpc`, `npcScript`, `npcFuncClick`, `npcFuncOther`. NPC handlers were separated from the combat package into their own `npc` package.

## NPC Healing System

Implemented `npcFuncOther` RPC handler for NPC type 8 (Heal).

Flow: Player clicks NPC -> `clickNpc` -> server sends `npcFuncInit` with `npcType: 8` -> client opens NpcFuncOther panel with options: ID 1=Heal character, ID 2=Heal pet (stub), ID 3=Heal both -> player confirms -> `npcFuncOther(npcId, funcId)` -> server validates, calculates cost, deducts money, restores HP/MP.

Cost Formula: `costMultiplier = min(BASIC_GET_MONEY[level] / 70000, 1)`, `healCost = ceil((missingHP + missingMP * 1.3) * costMultiplier)`. Uses `moneyBind` (ngan phieu) as default currency.

## Quest System

### Active quest abandonment compatibility (2026-10-16)

Accepted type-1 quests are now cancellable without editing the Flash client. The canonical template remains type 1 on the server, but `EnrichQuestDTO` publishes presentation type 2 for active type-1 progress because the served QuestManager and QuestPanel hard-disable Cancel when `data.type == 1`. `AbandonQuest` no longer rejects type 1 and serializes lookup/status/delete under the per-character quest mutation lock. Existing ownership, active-status and callback contracts remain unchanged; no completion history or rewards are created. The visual tradeoff is that accepted type-1 entries are grouped under main quests, while available catalog entries and all server-side semantics retain their original type.

### Quest reward callback correction (2026-10-11)

`FinishQuest` must not send zero-argument `onUpdateAchieve` or `onUpdateSafeBattleMid`. These are data updates, not refresh signals: served Flash expects `(type,id,data)` and `(mapIDs)` respectively. Removed both placeholders without inventing achievement progress or clearing safe maps. Consumed-item results carry instance `id` plus `sid`; full deletion sends `onDelCharactorSlot(id,sid)`, not SID as an item ID. Partial stack updates are unchanged. End-to-end in-memory completion tests cover both consumption paths and rejected repeated claims. Evidence: [Quest reward callback audit](../research/2026-10-11_03_QUEST_REWARD_CALLBACK_AUDIT_RESEARCH.md).

QuestTemplate fields: ID, Name, Type (1=tutorial,2=main,3=class,4=world,6=activity,7=loop,9=callboard,10=growup,11=guild,12=lucky), MinLevel, MaxLevel, StartNPC, FinishNPC, PreQuestType, Info, AwardExpRe, MoneyNum, MoneyType. Use actual Flash enums; “daily” is existing server shorthand for type6, not a type2 enum.

Objective Types (server `internal/domain/quest/objective.go`): KillMonster=1, CollectItem=2, TalkToNPC=3, ReachLevel=4, ExploreArea=5, UseItem=6, WinBattle=7, **SubmitPet=8**.

Server objective IDs intentionally differ from Flash `QUEST_REQUIRE_*` kinds (1=item / 2=creature / 3=pet). `GetInitialObjectives` maps TBL_QUEST_REQUIRE `kind` to the server enum and stores the row's `type` (TBL_ITEM_TEMPLATE=29 or TBL_EQUIPT_TEMPLATE=19) on `Objective.ItemTableType` and `q` on `Objective.Quality`. `Objective.ClientKind()` / `ClientTableType()` recover the Flash-side values for DTOs. See `internal/domain/quest/objective.go`.

Quest Flow: Click quest NPC -> GetNPCInteraction -> takeQuest RPC -> create QuestProgress. Progress updates: OnMonsterKilled, OnItemCollected, OnNPCInteracted. Completion: finishQuest RPC -> check all objectives -> give rewards.

Class Quest (type=3) dynamic rewards:

```text
completionIdx = completionCount % 10
EXP   = round((BASIC_GET_EXP[playerLevel] * CLASS_QUEST_MONEY_EXP_NUM[completionIdx] * 10) / 31.5)
Money = round((BASIC_GET_MONEY[playerLevel] * CLASS_QUEST_MONEY_EXP_NUM[completionIdx] * 3) / 31.5)
```

`CLASS_QUEST_MONEY_EXP_NUM`: `[0.6, 0.7, 0.8, 0.9, 1, 1.1, 1.2, 1.3, 1.4, 1.5]`.

Callboard Quest (type=9) reward multiplier from `CALLBOARD_AWARD_NUM = [1, 2, 8, 12, 20]`: Color index 0=1x, 1=2x, 2=8x, 3=12x, 4=20x. `exp = template.awardExp * CALLBOARD_AWARD_NUM[color]`, `money = template.moneyNum * CALLBOARD_AWARD_NUM[color]`.

Money type mapping: case 0=Money (default), case 1=MoneyBind, case 2=Money, case 3=Gold, case 4=GoldBind.

Quest visibility filter (`CanAcceptQuestWithSnapshot`) mirrors Flash `Player.canTakeQuest` (Player.as:1824–1911):
1. Skip if active or already completed.
2. Effective level uses `RebirthLvl` when quest `is_rebirth > 0`, else `Level`. Reject outside `[min_level, max_level]`.
3. Quest `gender` uses Flash `GamePredef`: `0`=MALE, `1`=FEMALE, `2`=NONE. Reject only when `gender != 2` and `char.Gender != gender` — every seed row currently has `gender=2`, so an inverted check (e.g. `gender > 0 && char.Gender != gender`) silently filters every quest and a brand-new character sees no NPC quest icons or NPC dialog quests.
4. Reject if `req_class` (pipe-delimited) is set and excludes `char.ClassID`.
5. Walk `data_tbl_quest_pre`:
   - `kind=1` (item): require `CountCarriedItemsByTemplateID >= num` (always AND).
   - `kind=2` (quest): branch on `pre_quest_type` — `==1` requires every prereq quest finished (AND); `==2` requires at least one finished (OR). Any other value (including `0`) ignores `kind=2` rows entirely, mirroring the Flash client switch which only handles 1 and 2.

The Flash QuestManager "Can Take" panel filters quests entirely on the client via `Player.canTakeQuest`, which reads the pipe-delimited `player.questLog`. The server therefore injects `data["questLog"] = "|id1|id2|...|"` into the player UPP in `buildShowCharacterInfoPayload` (auth/characters.go), built from `quest_history` via `questService.GetCompletedQuestIDs`. The earlier `completedIds` field on `initQuestManager` is unused by the client (this field is unrelated to the loop system's newer `"l"` field on the same response, added by [Quest Loop System](quest-loop-system.md)).

M3 (2026-05-29): `questLog` is now also injected into `cData` in `buildOnChooseCharactorPayload` (auth/login.go L536) via `questService.BuildQuestLogString(ctx, charID)` — returns `|id1|id2|...|` (leading+trailing pipe) from completed quests. Without this every completed quest re-appears available at login. Top-level `qn` is now an int scalar (count of active quests) — NOT the `questDTOs` slice; the Flash `Player.qn:int` cannot coerce an array. Active quest DTOs are delivered exclusively by `initQuestManager` (quest/manager.go) which re-fetches them on demand.

M9 (2026-07-04): `GetCompletedQuestIDs` moved out of `service.go` into `internal/application/quest/daily_quest.go` and is no longer a plain passthrough to the repository — it now excludes type-7 loop-child ids always and excludes type-6 daily quests once their latest `quest_history` completion is more than a day old (so dailies re-open after their `character_quests` row is deleted on completion). `BuildQuestLogString` and every other quest-availability check (`CanAcceptQuest`, `GetAvailableQuestsForNpc`, `GetAvailableQuestsForCallBoard`) now call this filtered `Service.GetCompletedQuestIDs` instead of hitting `s.questRepo.GetCompletedQuestIDs` directly, so the date-aware exclusion applies everywhere "completed" gates re-take. See [Quest Loop System](quest-loop-system.md) for the full mechanism (an optional `completionDateRepository` interface, not a `quest.Repository` interface change, so no existing repo fakes needed updating).

## Quest Requirement Catalog (TBL_QUEST_REQUIRE)

Authoritative Flash source: `QuestPanel.checkBag/checkEquipt/checkPet` (`docs/client/view/view/compDragable/QuestPanel.as:975–1247`), `QuestCanvas.as:1040–1123`, `DataManager.as:626–693`, `Basic.colorByGrowRate / getColorByQuality`. Constants in `GamePredef.as:340–358` cap requirement kinds at 3 — no gold / achievement / level rows exist in the schema.

| kind | type | Meaning | Server objective | Active progress sync | Turn-in consumption |
|------|------|---------|------------------|----------------------|---------------------|
| 1 | 29 (TBL_ITEM_TEMPLATE) | Collect consumable / quest item | `ObjectiveCollectItem` + `ItemTableType=29` | `syncCollectObjectives` counts bag + quest-bag by template, optionally filtered by exact color when `q ≥ 0` | `ConsumeCarriedItemsByTemplate` (delete or shrink stack) |
| 1 | 19 (TBL_EQUIPT_TEMPLATE) | Collect equipment instance | `ObjectiveCollectItem` + `ItemTableType=19` + `Quality` | Same as above; equipment color filter is **exact match** `getColorByQuality(q)` per `QuestPanel.as:1104` | Same as above; equipment instances are deleted as stacks of 1 |
| 2 | 12 (TBL_CREATURE) | Kill `num` creatures of template `itemId` | `ObjectiveKillMonster` | Incrementally pushed by `OnMonsterKilled`; `questKill` array reports remaining | Not consumable |
| 3 | 12 (TBL_CREATURE) | Surrender owned pets of template `itemId` | `ObjectiveSubmitPet` + `Quality` | `syncCollectObjectives` walks `petRepo`, requires `pet.TemplateID==itemId AND pet.Name==creatureTemplate.Name AND colorByGrowRate(pet.growRate) ≥ colorByGrowRate(q/10)` | Lowest-`growRate` qualifying pet is deleted via `DeletePet` (moves equipped pet gear back to bag), `onDelPet` callback sent per consumed pet |

`q` semantics:
- Collect objectives (kind=1, both equipment and consumable rows): `q < 0` skips the color filter — most consumable rows use `q=-1`. `q >= 0` requires an **exact** colorCode match equal to `EquipmentColorCodeFromQuality(q)` (= `(q+4)/5`), mirroring Flash `Core.getItemNumByColor` and `QuestPanel.checkEquipt`.
- Pet submit (kind=3): threshold `growRate = q / 10` with a **≥** comparison. Most rows use `q=1` (any pet passes); meaningful tiers exist at `q=12` (≥ blue) and `q=15` (≥ blue/purple cutoff). Mapping table in `internal/domain/pet/quality.go::PetGrowRateBands = [1.1, 1.3, 1.6, 2.0, 2.5]`.
- `syncCollectObjectives` lazily loads the full `petRepo` pet list only when at least one active quest has a SubmitPet objective; cost is bounded by typical pet counts (< 50 per char) and is reused across all pet objectives on the quest.

## Quest Prerequisite Catalog (TBL_QUEST_PRE)

Flash source: `Player.canTakeQuest:1824–1911`. Constants: `QUEST_PRE_ITEM=1`, `QUEST_PRE_QUEST=2`.

| kind | type | Meaning |
|------|------|---------|
| 1 | 29 | Player must **carry** `num` copies of TBL_ITEM_TEMPLATE `itemId`. Server respects `pre.Type` when counting: `CountCarriedItemsByTemplate(charID, itemId, 29, -1)` so an equipment instance with the same id cannot satisfy a consumable prereq. |
| 2 | 45 / -1 | Prerequisite quest gate; quest template's `pre_quest_type` decides AND (`==1`) vs OR (`==2`). Flash switches only on `kind`, so `type=-1` is functionally identical to `type=45`. |

## Quest-Battle, Tracker, and Accept-Time Handoff Rules

See `quest-battle-rules.md`.
