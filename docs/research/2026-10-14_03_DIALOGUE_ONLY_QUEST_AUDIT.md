# Dialogue-Only Quest Catalog Audit

**Date:** 2026-10-14  
**Scope:** Read-only classification of all 5,530 live quest templates. No player data, database mutation, Flash edit, deployment, or reward replay.

## Confirmed wire contract

- Flash has no dedicated conversation requirement kind. Requirement kinds are item, creature, and pet; quest category is independent.
- Flash displays active state `103` and ready-to-turn-in state `104`, but does not derive readiness from missing requirement rows. The server is authoritative.
- `startNpc` controls the offering NPC and `finishNpc` controls the turn-in NPC. Positive finish NPCs are listed by the NPC panel for states 103/104.
- Ordinary completion calls `finishQuest({i,s,n})`; choose-one reward completion calls `finishQuest({i,s})`. Success needs `{flag:true,qid,qn}` and failure needs `{flag:false,info}`.
- Missing ordinary requirement rows are compatible with a dialogue quest but are not sufficient proof because hidden/scripted conditions also exist.

Primary client evidence: `docs/client/predef/GamePredef.as:336-359`, `docs/client/data/DataManager.as:626-693`, `docs/client/view/view/compDragable/QuestPanel.as:441-503,778-836,910-930`, `docs/client/view/view/compDragable/NpcFuncPanel.as:277-326`, and `docs/client/view/view/comp/QuestCanvas.as:1040-1124`.

## Current server behavior

`internal/application/quest/requirement_policy.go` recognizes only an explicit type-1 conversation allowlist. Current source has 67 entries. A conversation is ready only near its positive finish NPC; final turn-in independently checks the same NPC context. Empty quests outside an implemented special policy remain blocked.

Quest 46 (`Chế Tạo Huy Hiệu`) is confirmed by the operator as dialogue-only. It has no requirement rows, `rt=rn=0`, start/finish NPC 430, and is already in the current 67-entry registry. The older 66-entry artifact and the statement that quest 46 is unsupported crafting are stale and must not be restored.

## Exhaustive result

Of 1,684 live quests without ordinary requirement rows, 1,015 had dialogue/report-shaped metadata and were examined in detail:

| Class | Count | Decision |
|---|---:|---|
| High-confidence | 227 | 67 current reviewed IDs plus 160 new type-2/type-4 candidates |
| Ambiguous | 711 | Keep blocked pending manual/runtime evidence |
| Contradicted/blocked | 77 | Keep blocked due currency metadata, unresolved NPCs, missing dialogue, or competing conditions |
| Empty but not dialogue-shaped | 669 | Existing special policy or unconfigured behavior remains unchanged |

Every high-confidence candidate has zero ordinary requirement rows, `rt=0`, `rn=0`, valid positive start and finish NPCs, non-empty start/complete dialogue, and explicit dialogue/report/listen/ask semantics or current reviewed-positive status. `qtest` is empty for all 227. Quest 46 and 161 contain only the known abandonment cleanup `delBuffBid(cid, 313);`; it is not a completion condition.

### Current reviewed type-1 IDs (67)

`1-4, 6, 15, 23-29, 31-46, 54, 58, 161, 163, 165, 167, 169, 172-173, 382-386, 1710-1713, 2563-2568, 2571, 2578, 3191-3201, 5189`

### New high-confidence type-2 IDs (67)

`196, 218, 223, 242, 254, 1046, 1126, 1135, 1283, 1299, 1307, 1310, 1364, 1842, 1845-1846, 3052, 3059, 3076, 3081, 3099, 3125, 3326, 3351, 3378, 3381, 3395, 3400, 3405, 3694-3695, 3699-3700, 3704-3705, 3712, 3722, 3730, 3735, 3738, 3750, 3752-3754, 3757, 3765, 4602, 4612, 4614, 4644-4645, 4647-4648, 4756, 4778, 4783, 4786, 4790-4791, 5375, 5387-5388, 5393, 31590, 31596-31597, 31614`

### New high-confidence type-4 IDs (93)

`71, 86, 115, 134, 1030, 1148, 1174, 1193, 1198-1199, 1203-1204, 1209, 1218, 1450, 1452, 1937-1938, 2493, 2499-2500, 2510, 2521, 2542-2544, 2546, 3010, 3016, 3023, 3032, 3037, 3049, 3219, 3235, 3237, 3243-3245, 3249, 3251, 3256-3257, 3266, 3279, 3292, 3295, 3305, 3313, 3623, 3648, 3651, 3654, 3657, 3659, 3676, 3679, 3682, 3685, 4533, 4538, 4550, 4558, 4562, 4571-4572, 4581, 4583, 4586, 4719, 5030, 5038, 5041, 5053, 5059, 5061, 5065, 5067, 5145-5146, 5148-5149, 5153, 5159, 5360, 5362, 5365, 5369, 31610-31611, 31613, 50367, 50370`

## Important exclusions

- The 711 ambiguous candidates are not an implementation allowlist. This includes type-3 class visits `6001-6018`, type-7 loop rows, type-11 guild rows, and type-10 special chains whose lifecycle or hidden condition cannot be reduced to ordinary dialogue.
- The 77 contradicted candidates remain blocked. Fifty-four carry `rt/rn` or explicit competing conditions; twenty-three have missing/unresolved NPC or dialogue structure.
- Actual NPC-click persistence is not currently part of conversation readiness. Runtime uses server-owned map/position within the local 300-unit policy. This audit does not introduce a new interaction-state table.
- Start-NPC acceptance authorization, generic NPC-click proximity, currency requirement semantics, and Flash catalog publication are separate issues.

## Reproduction

Inputs: `docs/database/game_data/TBL_QUEST.json`, `TBL_QUEST_REQUIRE.json`, `TBL_NPC.json`, `.local/research/quest-catalog-audit-20261012.json`, and the current conversation registry.

The pass restricts IDs to the 5,530-row live snapshot, excludes every qid with an ordinary requirement, joins start/finish NPCs, normalizes Vietnamese text, and then separates explicit dialogue/report wording from competing `rt/rn`, lifecycle, NPC, and dialogue-shape evidence. `cmd/questaudit` remains the baseline policy classifier; the implementation plan adds registry/NPC drift checks rather than deriving runtime authorization from text heuristics.
