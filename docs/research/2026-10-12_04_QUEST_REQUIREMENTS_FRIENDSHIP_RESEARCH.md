# Quest requirement audit: empty grow-up objectives and friendship

## Confirmed live failure (read-only evidence)

User reported friendship quests complete with an empty friends list. Compose Supabase CLI inspection found character 2 (GainTest): zero rows in player.friends; quests 2695/2696 completed with objectives=[]; quest 2697 active with objectives=[]. Runtime logs show successful rewards and UI state104 for 2697. No assistant acceptance/completion/grants or player-data repair.

Live templates: 211 type10 (client QUEST_TYPE_GROWUP) quests, 137 lack data_tbl_quest_require rows, including all39 friendship quests2695–2733. Requirement table globally has kind1=2181, kind2=1239, kind3=1344 and one null-kind row. These counts are inventory evidence, not proof all requirements are implemented.

Important seed-reading trap: data_tbl_quest_require tuple `(2695,48,1,1,-1,2994,29)` means requirement id2695, item48, quest2994 — NOT a requirement for quest2695. Columns are id,item_id,kind,num,q,qid,type. Live query confirms no requirements for friendship quests2695–2698. Do not grant or require item2994 for friendship.

## Cause

GetInitialObjectives returns [] when no ordinary require rows exist. Domain QuestProgress.IsComplete returns true for an empty slice; this is legitimate for ordinary NPC conversation quests, but not script-driven grow-up quests. No friendship relationship verification was wired into quest service. Client does not calculate friendship eligibility; it trusts server state103/104.

## Scoped correction

- Runtime-only CompletionBlocked (excluded from JSON/cache serialization) participates in domain IsComplete/CanComplete. Reset clears it; completed historical records are not re-evaluated or reverted.
- Quest service syncSpecialRequirements runs for active quest getters/completion through syncCollectObjectives and before EnrichQuestDTO. Thus NPC states, quest-manager snapshots and reward gate share verification; UI cannot authorize rewards on its own.
- Type10 active quests with no objectives fail closed until a specialized verifier exists. Other types keep ordinary empty-objective conversation behavior. Type10 quests with ordinary requirements retain the ordinary evaluator, except the explicit friendship chain also requires social verification.
- For the catalog-confirmed chain2695–2733, validate exact expected name `Kết giao N người bạn`, where N=id-2694. Read real FindFriends via the same PostgreSQL relationship repository used by social UI. Require N distinct non-self friend IDs and intimacy>=50 for **all** listed real friends. Nil/self/foreign-owner/non-friend records do not count; duplicates cannot satisfy count. Missing repository, name mismatch or read error never enables completion. Recheck each time, so deletion/intimacy loss clears eligibility.
- Number/intimacy rule comes from explicit template name and info (`tăng độ thân mật của tất cả bạn bè lên 50`). It is a description-backed local implementation, NOT a recovered original script. Whether the original rule used a different predicate/new-friends-since-acceptance cannot be established. No client fields/progress callbacks were invented; ordinary info plus state103/104 are used.
- No migration/seed/configuration/player-state writes by assistant. Existing completed rewards/progression on Gain and GainTest are preserved. Ordinary live quest reads may persist existing collect/pet objective sync as before, but this special flag is ephemeral.

## Client evidence (delegated read-only)

Client researcher verified:

- docs/client/object/Player.as:1824–1910: acceptance checks, not friendship predicate.
- docs/client/predef/GamePredef.as:336–359: state103=taken,104=canfinish; type10=growup.
- docs/client/view/view/compDragable/QuestPanel.as:910–930: Finish enabled by server state.
- QuestCanvas.as:1004–1202: ordinary info/item/pet/kill rendering, no dedicated verified friendship progress field.
- Actual served CallBack.as:11373–11378: onAddChaQuest envelope d/pos/clsData; no friendship calculation.

## Validation and limits

PASS: quest application, quest/NPC handlers, cache tests; vet and host build before final runtime-only JSON tag adjustment, then affected tests/vet rerun PASS. Container build/deploy completed for mcgame-server only; startup08:52:08, RTMP started, running/restart_count0. Latest startup loads itemAwards71 (not modified by this task); no award seed overwritten. Added blacklist/foreign-owner regression cases also PASS.

Tests cover zero friends; too few;49/50 intimacy boundary; extra low-intimacy friend; duplicate/self records; missing dependency/read failure; UI103/104; reward rejection; deletion/stale-state reversal; JSON exclusion; completed history untouched; ordinary talk quests unchanged; unsupported empty grow-up quest blocked.

NOT TESTED: actual Flash retest and successful real friendship quest reward. Assistant did not add friends or replay rewards to test. Other script-driven type10 quests without conditions remain NOT IMPLEMENTED, now blocked rather than falsely completed. No claim of a full quest-system audit or all-quest gameplay PASS.

## Separate security finding

Local CLI reported RLS disabled on public.game_tuning, public.gateway_config, public.gm_settings, public.server_settings. This is outside the quest fix; no RLS changes auto-applied. Keep the stack localhost-only pending a separately authorized policy audit.
