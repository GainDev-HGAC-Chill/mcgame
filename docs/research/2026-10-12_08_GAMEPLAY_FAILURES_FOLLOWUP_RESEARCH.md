# Gameplay failure follow-up

## Scope and safety

Reported: character deletion unavailable, recheck all quest conditions, VIP panel/purchase AsyncError2095 for onInitPmData, expansion books unusable, freeze on turning in a newly accepted quest. No live deletion, book consumption, VIP purchase, quest reward replay, player repair, reset, or client modification is used to reproduce these issues. Existing unrelated workspace changes remain intact.

## Quests

Live logs show character3 finishing quest6 at12:41:11. Server reaches equipment758 creation, then no completion response. CompleteQuest owns the per-character mutation lock; item reward creation synchronously invokes OnItemCollected through the bidirectional runtime wiring and tries to reacquire the same lock. Earlier isolated tests did not wire item-to-quest. This is a server deadlock, independent of the quest being a valid conversation.

Fix: private completion context queues synchronous self-owned item notifications; release the lock before flushing notifications through the normal authoritative event path. Both normal and loop completion use this scope. No broad unlocking around consumption, completion, or reward grants. Other service/character events retain normal processing. Regression uses both service directions, a real in-memory award, a second active collect quest, bounded completion and rejected duplicate reward.

Level notifications previously wrote stale active snapshots with caller-supplied newLevel. They now evaluate current server-owned conditions without persisting the snapshot; real reads/completion verify actual character level. Regression checks no writes and no completion from a fabricated level999 notification.

Independent review found additional responder defects: wrong/incomplete turn-in failures used msg while Flash reads info; rate-limited awaited finishQuest silently suppressed the error, and generic handler errors produced a transport fault although this panel has no fault responder. Failures now carry flag=false/info; awaited finishQuest dispatch errors (including rate limit) send a terminal result without invoking/granting again. Tests exercise two transactions inside1100ms and invalid/auth handler errors. Other RPC error policies are unchanged. FinishQuest also pushes canonical progress envelopes for remaining quests after reward slot callbacks, so another quest satisfied by the reward does not retain a stale tracker. End-to-end callback-order test includes actual bidirectional wiring. The Flash panel's failure path itself hides the Finish button; reopen the panel after a rejected attempt, not a claim that the client button implementation changed.

Reran the classifier against the existing full5530-row snapshot; unchanged partition:3682 catalog,66 reviewed conversations,14 character-level,14 pet-level,23 pet-count,39 friendship;1493 unconfigured,195 currency semantics and4 missing-target conditions blocked. Follow-up artifact `.local/research/quest-completion-policy-followup-20261012.json`. This is not a fresh database export or proof that all quests are playable. Full evidence/limitations: [whole-catalog audit](2026-10-12_06_ALL_QUEST_REQUIREMENTS_AUDIT_RESEARCH.md).

Already interrupted completion may have persisted completed state and a prefix of rewards before waiting. This patch does not recover/replay that history. Any recovery requires a separate read-only inventory/history diagnosis and explicit, precise remediation decision; never tell the player to repeat purchases or consume items to reproduce it.

## VIP

initPmData and buyPm are panel Responder calls, not global callback methods. Removed nonexistent global onInitPmData/onBuyPm pushes; responder results carry existing payloads. Valid onAddPm/currency/scene pushes remain. Failed purchase returns false instead of nil, ensuring a result for the waiter. Tests verify responder results and no unsupported callback. See `docs/memory/pm-vip-system.md`.

## Character deletion

BLOCKED: unregistered startDeleteChar/restoreChar; hard-delete repository is not a compatible implementation of pending deletion with restore. No persisted deadline supported, original grace-period/finalization rules and success callback order unproven. No deletion is performed or claimed fixed. [Deletion evidence](2026-10-12_07_CHARACTER_DELETION_RESEARCH.md).

## Expansion books

Templates324/325/326 previously went through generic scroll handling: consumed but no capacity effect. Exact-ID implementation now expands bank by one30-slot tab (max5), bag by one30-slot tab (max7), carried pets by one slot (existing domain max180). Synchronizes persisted tab/count aliases; rejects invalid target/template/ownership/level/count/cap before consumption. Multi-use preflights the whole count. Owner receives remaining book stack/original-SID deletion then onUPP capacity update. No currency debit, no pet-bank or pet-equipment-bag expansion. This task does not refund books consumed before the fix. Source/coverage in `docs/memory/item-inventory.md`.

Item-repository delete/stack errors preserve capacity and books, but separate consumption and character persistence are **not atomic**: a later character-write failure can leave a consumed book without durable expansion. Do not claim all storage failures preserve the book; do not automatically rollback/regrant historical uses. Capacity-only single/multi handlers now return an explicit wrapped error before any successful capacity callback on character-write failure; no retry or speculative rollback. Failure tests for all three IDs and both routes PASS; generic unrelated item behavior unchanged.

## Verification and deployment

Focused quest application/handler, activity, item and RTMP tests PASS; reward-lock/level/credit/read, finish responder/callback-order and expansion regressions repeated20 times PASS. Affected vet and host build PASS. Full suite run retains the previously observed four pet-stat, one auth PM EXP and two chat-stat failures; no blanket full-suite PASS. Docker image build/recreate PASS, but runtime startup BLOCKED: database, Redis, edge and the rest of the existing stack had already stopped at12:42, before this redeploy. New game server fails DNS lookup db and restarts. Stopped only this newly recreated game container to prevent the failure loop; did not recreate/restart/delete database or Redis, alter volumes, or repair player data. Await operator approval to start the existing stack. Actual Flash retest NOT TESTED. Ordinary dependency stalls and mid-reward storage failures remain separate nontransactional boundaries; no automatic history replay.
