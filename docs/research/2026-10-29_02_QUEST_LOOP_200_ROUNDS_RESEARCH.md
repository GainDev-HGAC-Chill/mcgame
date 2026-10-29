# Quest Loops and 200-Round Quest — Research

**Date**: 2026-10-29

## Scope and authority

This research covers the shared type-7 loop lifecycle and loop ID 9, `Nhiệm Vụ 200 Vòng`. Decompiled Flash code is authoritative for RPC and callback behavior. Catalog and seed data are authoritative for IDs and static configuration. Public websites are gameplay corroboration only.

Sources:

- `docs/client/view/view/compDragable/NpcFuncPanel.as`
- `docs/client/view/view/compDragable/QuestManager.as`
- `docs/client/view/view/comp/QuestCanvas.as`
- `docs/database/game_data/TBL_QUEST_LOOP.json`
- `docs/database/game_data/TBL_QUEST.json`
- `docs/database/game_data/TBL_QUEST_REQUIRE.json`
- `supabase/seeds/data_tbl_diary.sql`
- `supabase/seeds/data_tbl_achievement.sql`
- `supabase/seeds/data_tbl_achievement_require.sql`
- http://vuaphapthuat.go.vn/bai-viet/nhiem-vu-200-vong/4255.html
- https://web.4399.com/mlxt/
- https://www.youtube.com/@Magiccampusneo

No 4399 article or Magiccampusneo capture found in this pass supplied an exact reward formula, take fee, milestone item/plan identity, or boss-spawn protocol.

## Confirmed Flash contract

- The NPC panel calls `takeLoop(lid, npcID)`; its fallback without an NPC sends `npcID=-1`.
- The quest manager calls `cancelLoop(lid)` and `loopRepaire(lid)`.
- These methods are fire-and-forget. The client consumes `onTakeLoop`, `onCancelLoopQuest`, and the repaired `onAddChaQuest` push.
- Loop completion consumes `onFinishLoopQuest`. The server must send it before the next round's `onAddChaQuest`.
- Client loop state uses `id`, `qid`, `takeDate`, `ft`, and `finished`; the loop template ID is used for both `id` and `qid`.
- The client only calculates the visible dynamic formula in `QuestCanvas.as` for `QUEST_TYPE_CLASS`, not type 7. The shared `CLASS_QUEST_MONEY_EXP_NUM` factors therefore do not authorize a 200-round formula.

## Loop ID 9 catalog facts

| Property | Value |
|---|---:|
| NPC | 48 |
| Level range | 70–160 |
| Rounds | 200 |
| Team mode | solo (`0`) |
| Cancel cooldown | 259,200 seconds / 72 hours |
| Child pool | `subType="7-9"` |

The loop description and Vua Pháp Thuật article agree that each completion grants EXP, rewards increase within each ten-round cycle, and later ten-round cycles are worth more. They also agree on milestone categories: level-5 material at 50, advanced gem at 100, Nội Đơn Huyền Thú at 150, and an advanced pet skill book at 200. The catalog additionally describes feather-property and feather-fusion-formula rewards.

Neither source identifies a deterministic item template or loot plan. The category examples explicitly permit multiple possible items, so choosing one catalog ID would be invented behavior.

## Child-pool census

- 924 type-7 child templates use `subType="7-9"`.
- 725 requirement rows exist: 251 kind 1, 12 kind 2, and 462 kind 3. The remaining 199 children have no requirement row and are NPC-dialogue rounds.
- All 924 children have `awardExp=0` and `moneyNum=0`.
- No `TBL_QUEST_AWARD` row exists for these children.
- The pool is cumulative by minimum level and capped at 160. The current server's rebirth-aware min/max filtering is consistent with those catalog fields; no authoritative weighting script survives because loop 9 has no `scriptGet`.
- Uniform selection among eligible children is therefore an explicit approximation, not recovered original behavior.

### Special kill rounds

QIDs `14001..14012` form one boss round for each threshold 70, 75, 80, 85, 90, 95, 100, 105, 110, 115, 120, and 125. They target creature IDs `556, 566..576`. None has a `TBL_MAP_CREATURE` spawn. No authoritative NPC pairing or summon protocol was found.

The current server safely excludes these 12 whenever any of the other 912 fightable children are eligible. This preserves completion safety but does not reproduce the original boss encounter.

## Milestone progression facts

- Diary IDs 13, 14, 15, and 16 correspond to rounds 50, 100, 150, and 200. Each grants 45 vitality and has a daily maximum of one.
- Achievement IDs 5038, 5039, 5040, and 5041 correspond to reaching 50, 100, 150, and 200 rounds once.
- Achievement IDs 6027 and 6028 correspond to completing the full 200-round loop 5 and 10 times.
- Achievement requirement kind 16 identifies this progression family, but current server code has no authoritative producer mapping for it.

These rows prove milestone counters but do not prove item identities, callback ordering for awards, or an atomic persistence command.

## Current server defects

1. `TakeLoop` persists an active state before saving the child progress. A child-save failure leaves an active loop with no child.
2. `TakeLoop`, `CancelLoop`, and `RepairLoop` do not use the service's per-character mutation lock.
3. `TakeLoop` ignores the supplied NPC ID and does not validate the catalog NPC or server-side position.
4. Non-guild completion consumes objectives, applies rewards, deletes progress, and advances the loop through separate operations.
5. Reward and progress-delete errors are logged and ignored, allowing progression after a failed reward or failed cleanup.
6. Loop 9 is advertised as takeable even though its 924 children contain no static reward and the server has no dynamic EXP, fee, milestone item, diary, or achievement implementation. Item-requirement rounds can therefore consume player inventory and advance while omitting the defining rewards.

The guild loop already demonstrates the correct safety boundary: fail closed unless an atomic command with calibrated rewards is available.

## Evidence-safe conclusion

The exact 200-round reward command cannot be implemented without inventing economy values and item identities. Until authoritative captures or original server scripts provide those values, loop 9 must fail closed before state creation, objective consumption, reward mutation, progress deletion, or round advance. Cancellation and repair of retained states must remain available so existing data is recoverable.

General lifecycle hardening that does not alter gameplay values is safe now: serialize mutations, validate the take NPC and position, compensate take failures, and propagate completion failures rather than advancing after them.
