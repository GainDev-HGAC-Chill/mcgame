# Guild System Full Audit

**Date**: 2026-10-27  
**Scope**: Flash client contract, static game data, current Go server, PostgreSQL schema/functions, and gameplay-reference sites. Research only: no Go/schema change, migration, restart, deployment, or player-data mutation was performed.

## 1. Authority and confidence

1. The decompiled Flash client is authoritative for RPC names, argument shapes, callback names, callback arity, and payload keys.
2. `TBL_BUILDING`, `TBL_EXTEND_POSITION`, `TBL_SKILL`, `TBL_QUEST_LOOP`, `TBL_QUEST`, and `TBL_NPC` are authoritative for catalogs, costs, prerequisites, map positions, and quest/NPC relationships.
3. PostgreSQL schema and functions are authoritative for currently durable state and transaction boundaries.
4. Website articles are gameplay context only. They are version-dependent and are not evidence for RPC or database contracts.
5. A mechanic is safe to implement only when at least two independent sources agree, or one primary source completely defines the contract. Unknown callback ordering and absent economy rules remain unresolved rather than guessed.

## 2. Confirmed gameplay model

- The inspected client requires character level 50 and 1,000,000 silver to create a guild. The server currently applies the same level and amount, preferring bound silver when either wallet alone has the full amount (`GuildPanel.as`; `membership.go`).
- The old 4399 overview says level 30 for joining, not creating, and an old guide mentions creating for 100,000 silver. Those values conflict with the inspected client version and must not override it.
- Guild map 49 contains 11 extension positions. `TBL_NPC` places guild quest NPC 1069 and guild-building NPCs 1095–1105 on that map.
- The building catalog has 18 rows: one empty extension point, one construction placeholder, and four functional families: hall `2/6/7/11/12/16`, house `3/8/9/13/14/17`, warehouse `5/15`, academy `4/18`. Rows define `preBuilding`, `postBuilding`, level, count limits, and costs in guild money, guild experience, general material 2026, rare material 2027, and special material 2039.
- Public gameplay documentation agrees that the hall gates guild/building progression, houses raise member capacity, the warehouse stores guild resources, and the academy develops guild skills. This supports the catalog model but does not define wire behavior.
- `TBL_SKILL` contains 130 guild-development rows (`useEnv=4`): 13 families and levels 1–10. The client initializes one level-1 row per family and derives the next row by matching the current `codeName` and `level+1`.
- Every family changes `codeName` to a `NO` suffix at level 7. Therefore the client-visible chain demonstrably reaches only level 6. Levels 7–10 must remain unavailable until another authoritative transition rule is found.
- Guild loop quest `lid=2` is associated with NPC 1069, supports levels 30–160, and has 15 rounds in current data. Existing quest code checks guild membership and cancels guild quest state on departure; no replacement quest implementation is justified by this audit.
- Public documentation agrees that guild quests, donations, construction, and skills are core guild activities, and that leaving clears contribution in the historical version. The current server's 90% restore rule is repository-specific behavior and is not established by those articles.

## 3. Flash RPC matrix

All 32 registered server methods are reachable by the inspected client or its scripted building/NPC path. `Responder` means the client directly consumes the RPC return; otherwise the UI depends on a push callback or later refresh.

| Group | RPCs | Confirmed request/response behavior | Current assessment |
|---|---|---|---|
| Panel reads | `initViewGuildP`, `getGuildMemberList`, `initGuildList` | Panel-init and list/member refresh; `getGuildMemberList` uses a `Responder` | Core shapes exist; callback audit still requires exact push tests |
| Membership | `addGuild`, `addGuildMember`, `quitGuild`, `confirmGuildApply`, `refuseGuildMember`, `kickGuildMember`, `demiseTo`, `delGuild` | Object/ID requests; panel handlers consume `onAddGuild`, `onAddGuildMember`, `onConfirmGuildApply`, `onDelGuildMember`, `onDemise`, `onDelGuild` | Mutations are not end-to-end atomic |
| Management | `updateGuildMember`, `updateGuildRank`, `updateGuildNotice`, `changeGuildName`, `guildLevelUp`, `addGuildBankSlotNum`, `addMaxGuildMemberNum` | Rank update sends one object; level/page/capacity use fire-and-forget calls | Multiple confirmed callback-name/arity defects; costs and limits incomplete |
| Contribution | `contribMoney`, `contribMaterial` | Money sends `{s,g}`; material sends `{slotId,num}` and consumes responder result | Current writes can lose currency/items on later failure |
| Warehouse | `getInitGuildSlot`, `moveGuildItem`, `dropGuildItem` | Initial load uses a `Responder`; move sends source/target SIDs; drop sends slot ID | Move/merge is non-atomic; server ignores configured `canSlot` authority |
| Skills | `getGuildPrivateSkillData`, `developSkill` | Read uses a `Responder`; develop sends `{gid,sid}` for exact selected next row | No cost, authority, environment, or chain enforcement |
| Buildings | `clickBuild`, `execBuildFunc`, `constructBuild`, `finishCreate`, `finishUpgrade`, `resetConstruction`, `createGuildBuildings` | `constructBuild(extendID,targetTID,mode,mapID)`; finish/reset use build ID; scene asks for buildings on map load | State is process memory, not guild-scoped durable state; catalog costs/prerequisites are not enforced |

## 4. Confirmed callback contract defects

### 4.1 Rank settings

- Initial truncated-source inspection incorrectly treated the GuildPanel method as the global callback signature.
- Complete shipped trait recovery proves `CallBack.onUpdateGuildRank(param1:Object)` and forwards `param1.gid, param1.rankData` to GuildPanel.
- The NetConnection callback must therefore carry one wrapper object `{gid,rankData}`.

### 4.2 Guild property and numeric deltas

- Client panel implements `onUpdateGuildProperty(Object)` and `onUpdateNumProp(Object)`, but these are not global callback traits.
- Complete shipped traits prove the global wrappers `onGuildPropertyRefresh(Object)`, `onUpdateGuildNumProp(Object)`, and `onUpdatePerNumProp(Object)`.
- `onUpdateNumProp` consumes `type` and `num`: type 1 donation contribution, type 2 normal contribution, type 3 guild money, and type 4 guild experience.
- Sending panel-only names through NetConnection causes Error #2095/#1069. Server broadcasts must use the proven global wrappers; panel-independent authoritative refresh remains preferable when the panel may not be initialized.

### 4.3 Guild info updates

- Client panel implements `onUpdateGuildInfo(Object)` and `onUpdateGuild(gid, property, value)`.
- Complete shipped trait recovery proves global `updateGuildInfo(Object)`, which forwards to the panel method. Global `onUpdateGuildInfo` does not exist.

### 4.4 Unresolved callbacks

- The repository copy of `CallBack.as` is truncated. Complete shipped trait recovery is required whenever a callback is absent there; warehouse and construction callbacks used by the repair have since been checked against the shipped trait.
- `DataManager` has matching guild-slot mutation methods and `BuildingView` has `reloadBuild`, which is supporting evidence, not proof of callback name/ordering.
- No client source handler was found for `onGuildSkillDeveloped`.
- Do not rename or remove unresolved pushes based only on absence from a truncated decompilation. Require complete shipped-trait or runtime evidence.

### 4.5 Shipped construction-entry callback recovery

- Follow-up on 2026-10-28 inspected the shipped encrypted main asset `frontend/s/sv/6a41a5ba5c0006ab4e0c1321a2b05030`. The shipped loader decrypts each byte with `byte XOR ((index + 1) % 3)`; the resulting SWF has SHA-256 `FD7B5538376A63C8D4701DB43515B92B7390F00093B9D935F7817D42436D43AC`.
- The complete `CallBack` trait establishes `onCreateGuildBuildings(Object)` → `sceneCreateGuildBuildings(Object)` for map hydration and `onShowNewBuildManager(int)` → `_core.view.getE(extendId).showNewBuildManager()` for opening the create panel at an empty extension.
- NPC 1098 selects script function `startConstruction` through a fire-and-forget `npcScript(func)` call, so an RPC return value is ignored. Its stored extension context is ID 10, matching the catalog empty point at `(245,407)`. The server must emit `onShowNewBuildManager(10)`; there is no `CallBack.startConstruction` trait.
- This resolves the construction entry method, arity, and argument without changing ActionScript or inferring a callback name. Finish/reset callback ordering remains governed by the separately recovered traits and runtime verification.

## 5. Persistence and transaction gaps

| Severity | Area | Evidence | Failure mode |
|---|---|---|---|
| Critical | Guild creation | Guild/member creation and wallet debit occur in separate service/cache operations | Free guild or debited wallet with compensating disband failure |
| Critical | Money contribution | Character wallet/contribution is persisted before guild/member contribution | Money can be lost while guild receives nothing |
| Critical | Material contribution | Item stack is consumed before warehouse/contribution writes | Material can be lost while warehouse/member state is unchanged |
| Critical | Warehouse merge | Target update and source delete/update are separate calls | Duplication, loss, or partial merge after one write fails |
| Critical | Construction | `Handler.buildStates` is in-memory and keyed only by build ID | Restart loses work; different guilds collide; unauthorized finish/reset is possible |
| High | Building economy | Completion does not debit `TBL_BUILDING` money/exp/material costs and create/upgrade derives state from guild level | Free progression and fabricated building layout |
| High | Skill economy | `DevelopSkill` only checks membership and upserts the requested skill | Arbitrary guild skill ID, no exact-next check, no cost, no permission |
| High | Rank authority | Settings are JSON in `guild_logs` action 9001 while member permission columns remain independent defaults | UI permissions and server authorization diverge |
| High | Contribution authority | Character currencies 20/21 and `guild_members` totals duplicate the same personal state | Drift after partial writes; login/UI/achievement readers can disagree |
| High | Membership lifecycle | `add_guild_member`/`remove_guild_member` are atomic internally, but restore currencies, quest cancellation, application state, and cache synchronization sit outside | Membership succeeds while restore/cleanup fails |
| Medium | Bank pages | Page count is JSON in `guild_logs` action 9002, with no catalog-backed debit | Durable but weakly modeled, difficult to lock/update safely |
| Medium | Guild level/capacity | Level increments and capacity `+10` have leader checks but no validated cost/prerequisite | Free or unbounded progression |

The existing `add_guild_member` and `remove_guild_member` functions correctly combine member-row, character `guild_id`, and population changes, but they do not check affected-row counts and `remove_guild_member` can decrement population even when no member was deleted. They are not sufficient transaction boundaries for the complete lifecycle.

## 6. Required durable model

### 6.1 Canonical personal contribution

- Make `player.character_currencies` types 20 and 21 the canonical spendable personal balances because the rest of the character/economy system already loads them through `get_character_currencies`.
- Keep `guild_members.contribution_normal`, `contribution_donate`, `contribution_total`, and `contribution_weekly` as guild-membership ledger/leaderboard values, updated in the same transaction as currencies 20/21.
- Add reconciliation diagnostics before any cleanup or backfill. Do not silently choose one side for already divergent live rows.

### 6.2 Rank permissions

- Replace log-encoded current settings with a typed per-guild/per-rank table keyed by `(guild_id, rank)` and six client-visible flags: `can_add`, `can_quest`, `can_slot`, `can_info`, `can_del`, `can_duty`, plus display name.
- Server authorization must read this same row. Legacy member boolean columns may remain compatibility fields initially but must not remain a second authority.
- Seed defaults equivalent to `DefaultRankSettings`; only the leader can edit settings.

### 6.3 Buildings and construction

- Add one durable building row per `(guild_id, extend_id)` with template ID, map/position identity, build state, pending target template, mode, actor, timestamps, and a version/revision.
- Never infer built templates from guild level. `createGuildBuildings` must read persisted rows and use `TBL_EXTEND_POSITION` only for immutable coordinates.
- Start/finish/reset must lock the guild and building row, validate map 49, extension type, actor membership/permission, mode, exact catalog predecessor/successor, `preBuilding`, `numLimit`, and resources.
- Debit catalog costs atomically with the state transition. Because the client exposes start and finish separately but no reliable timer semantics were recovered, the first safe implementation may persist a pending construction and permit explicit finish without inventing a duration; it must not auto-complete or fabricate progress.

### 6.4 Guild skills

- Store canonical learned level per client-visible family rather than trusting arbitrary submitted IDs.
- Validate `useEnv=4`, requested row level, exact-next row, same current `codeName`, and catalog costs. With current evidence, reject every transition beyond level 6 because level 7 changes family key to `*NO`.
- Lock the guild and skill family, debit `guildDevExp`/`guildDevMoney` from guild resources in one transaction, then return the refreshed skill map. `costGuildContrib` appears in catalog/UI but its payer semantics are not established strongly enough for implementation; fail closed for rows requiring it until confirmed, or obtain an original-server trace.

## 7. Safe implementation boundaries

- Wire fixes, typed permissions, and read-model corrections can be implemented without inventing gameplay.
- Creation, contribution, warehouse, skill, and building writes require PostgreSQL command functions/transactions and failure-injection tests before runtime use.
- Existing guild quest membership behavior should be regression-tested, not rebuilt.
- Guild war, maintenance/daily production, building duration, skill level 7–10 transition, bank-page cost, guild-level cost, member-capacity cost, and original push ordering remain unsupported by sufficient evidence.
- Unknown mechanics must reject safely or stay unavailable. They must not be approximated from guild level, website prose, or in-memory state.

## 8. Source index

### Repository evidence

- Client: `docs/client/view/view/compDragable/GuildPanel.as`, `GuildContribPanel.as`, `GuildWarehousePanel.as`, `GuildBuildProcess.as`, `ConstructionManager.as`; `docs/client/view/view/comp/GuildSkillDevSlot.as`, `BuildSlot.as`; `docs/client/view/view/compGameStage/BuildingView.as`; `docs/client/data/DataManager.as`; `docs/client/predef/GamePredef.as`.
- Game data: `docs/database/game_data/TBL_BUILDING.json`, `TBL_EXTEND_POSITION.json`, `TBL_SKILL.json`, `TBL_QUEST_LOOP.json`, `TBL_QUEST.json`, `TBL_NPC.json`.
- Server: `internal/presentation/rtmp/handlers/guild/`, `internal/application/guild/`, `internal/domain/guild/`, `internal/infrastructure/persistence/postgres/guild_repository*.go`.
- Schema: `supabase/migrations/20260328184527_baseline_mcgame_schema.sql`, `20260410043725_guild_leave_restore_state.sql`, `20260414084416_add_high_risk_repo_db_functions.sql`, `20260417183544_character_db_functions.sql`, `20260417184534_extra_db_functions.sql`, `20260514041806_batch3_social_functions.sql`.

### Gameplay references consulted

- http://web.4399.com/mlxt/new4/guild.html — historical guild overview, map/tasks/donations/buildings/permissions.
- https://web.4399.com/mlxt/yxgl/a1196618.html — historical guild-task guide; version-specific values conflict with the inspected client.
- http://vuaphapthuat.go.vn/bai-viet/he-thong-bang-hoi/4053.html — Vietnamese guild-system reference.
- http://vuaphapthuat.go.vn/bai-viet/he-thong-nhiem-vu/4047.html — Vietnamese quest-system reference.
- http://vuaphapthuat.go.vn/bai-viet/phuc-loi/4347.html — Vietnamese benefits/reference context.

## 8.1 2026-10-28 building and guild-quest follow-up

### Building coverage

- `TBL_BUILDING` contains two infrastructure rows (`1` empty extension, `10` pending placeholder) and 16 playable templates: Hall levels 1–6, House levels 1–6 with family limit 2, Warehouse levels 1–2, and Academy levels 1–2. Map 49 has 11 extension positions.
- The client creation panel offers every type-1 level-1 template. Upgrade selection follows `postBuilding` when present, otherwise exact same-`codeName` level + 1. All current `postBuilding` values are empty, so the four catalog families define every upgrade chain.
- Start validation currently checks level-1 create or exact-next same-family upgrade, `preBuilding`, `numLimit`, extension occupancy, and leader authority. Finish atomically checks and debits guild money, guild experience, and warehouse templates 2026/2027/2039.
- Only Hall level 1 received a live create/reset/finish smoke test. No other template or nonzero-cost finish has been tested through Flash.
- Confirmed functional gap: finish persists the template and debits resources but does not apply catalog properties. Client labels prove property 1 is member limit, property 4 is building level, and property 5 is guild warehouse. The catalog uses these for Hall/House/Warehouse, while the server does not derive or update guild level, maximum population, or bank-page count from completed buildings. Academy access is routed by `funcScript`, but skill development is not gated by a completed Academy.
- House capacity aggregation remains ambiguous: Hall rows also carry property 1 value 20 and two House instances are allowed. Current database default `max_population=50` is not derivable unambiguously from the catalog. Do not guess additive/max semantics.

### Finish semantics

- `GuildBuildProcess.checkBtnEnable()` enables the button labelled `Xong` only when the client snapshot has enough guild money, guild experience, normal material, rare material, and special material. `finishBuild()` immediately sends `finishCreate(buildingID)` or `finishUpgrade(buildingID)`.
- No timer, elapsed-duration field, construction percentage, worker count, or quest-completion gate occurs in the shipped panel or persisted server state. The safe server implementation therefore uses explicit resource-gated finish, not an invented timer.
- Hall level 1 and Warehouse level 1 have zero cost in the authoritative catalog, so immediate completion is expected for those two templates. Every other playable template has nonzero catalog costs and cannot finish unless the transaction sees all required resources.

### Guild loop quest and rewards

- `TBL_QUEST_LOOP` ID 2 defines the 15-round guild loop, minimum level 30, NPC 1069, guild membership gate, and text promising character EXP, personal contribution, guild EXP, guild funds, materials, milestone chests, and high-round recipes.
- The pool contains 275 type-7/subtype-`7-2` quests (IDs 30001–31514). Every row has one requirement: 172 Pet submissions (type 12) and 103 item submissions (type 29). There are zero matching `TBL_QUEST_AWARD` rows, and quest rows carry zero/empty fixed EXP and money rewards.
- Current server correctly gates loop ID 2 by guild membership, grants children, validates and consumes item/Pet objectives, advances up to 15 rounds, and cleans guild quests on departure. However, `applyRewards` ignores round count and has no guild economy/contribution command; these 275 rows therefore grant none of the documented guild-loop rewards.
- The Vietnamese and Chinese gameplay references independently describe character EXP, personal contribution, guild EXP/funds, materials, and milestone rewards. They establish that rewards exist, but not exact formulas, probabilities, payload ordering, or transaction amounts. Website prose is insufficient authority for implementation.
- Operator-provided Vietnamese gameplay capture `https://www.youtube.com/watch?v=33WtZ4P7jKY` (`Q Bang 003`, reviewed 2026-10-28) narrows the observed completion reward: a level-160 character receives a level-scaled character-EXP message after each guild-loop completion, plus the separately catalogued 2 diary activity points. The capture does not visibly award personal contribution, guild EXP/funds, or materials. This is stronger runtime evidence than the generic loop description for reward kinds, but one level-160 recording still does not establish the full level lookup/formula or exact rounding for levels 30–159.
- A second operator-provided capture, `https://www.youtube.com/watch?v=MT5gH-7QTkI` (`Vua Pháp Thuật NEO - Hướng dẫn nhiệm vụ bang hội phần 1`, reviewed 2026-10-28), is visibly a different NEO server/version and conflicts with the EXP-only capture. At character level 90 its completion log visibly includes 2 diary activity points, 272,160 silver, 272,160 bound silver, 4,000 of a `Kết Tinh...` reward, 3,042 guild EXP, and 1,521 personal contribution. Because the recording identifies a NEO variant and does not expose protocol/state provenance, these amounts are comparative gameplay evidence, not authority for this shipped client's economy.
- The two runtime captures therefore prove reward behavior varies by server/version. Neither may be generalized into this backend without first identifying which variant matches the shipped SWF and obtaining readable before/after state or backend rules from that same variant.
- Chinese original-version documentation also describes a distinct per-building construction-task system whose completions accelerate a selected building and grant guild resources/special material. No building↔quest mapping, construction progress field, open-task RPC, or exact reward formula is present in the inspected client/data/server. This mechanic must remain unimplemented until shipped/original-server evidence is obtained.

### Follow-up safety conclusion

- The construction state machine and resource gate are valid, but the wider building system is incomplete because building effects are not applied and 15/16 playable templates have not received live coverage.
- The guild-loop route is structurally implemented but its reward side is incomplete and currently consumes objectives without authoritative rewards. Further live mutation should stop until reward authority is recovered or completion is deliberately fail-closed before consumption.
- Required original-server evidence: guild-loop captures from the exact server/version matching the shipped SWF, at additional known character levels, with readable EXP, currency 20/21, silver/bound-silver, guild funds/EXP and warehouse-material before/after state; one construction-task open/complete trace; and member-limit behavior with Hall plus one/two House instances.

## 9. Audit conclusion

The server has broad RPC coverage but not a safe guild authority. The highest-risk defects are non-atomic economy writes and non-persisted construction. The first implementation wave should correct proven wire/permission defects and establish transaction seams; subsequent waves should move economy, warehouse, skills, and buildings to locked PostgreSQL commands. Runtime callback ordering that cannot be recovered from source must remain unresolved pending original-server or Flash trace evidence.
