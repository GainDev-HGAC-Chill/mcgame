# Bag sorting and unconfigured event panels

## Report and evidence

The operator reported ghost/incorrectly positioned items after sorting, `item not found` on dragging, missing methods for server-new/system gifts, and GameIntroPanel.onGetPoint Error1010 on recharge events.

Runtime logs: bagSort at22:56:53 sorted11 items; subsequent moveItem(2116,2115) at22:57:18 rejected missing source at bag index15. Later dragging from2103 worked. Read-only DB showed11 final bag records at indices0–10. This establishes client/server position mismatch, not item loss. Missing methods include getActAwardInfo(0) and serchForGift(). No real inventory mutations/reward claims were performed by the assistant.

## Client contracts

Research delegated to the Flash client researcher. Existing actual SWF CallBack export under `.local/research/mcgame-callback-as-research` verifies callbacks; panel/DataManager/Slot details come from partial `docs/client`, not newly verified panel exports.

- Actual onItemSort takes rows oldId/newId as **SIDs**, not persistent IDs, resolves all sources before modifying positions, then restores all unlocked bag slots. It does not remove merged records.
- Existing server sort already emitted the correct deletion-before-sort and mapping protocol. Its failure mechanism remains unconfirmed. bag_slots persisted1; actual client bagSlotNum and callback completion were not observed. Do not claim sequential sort moves caused this specific report.
- onDelCharactorSlot(recordID,oldSID) cleans/deletes by SID; deleting after remapping can remove a survivor. Never send old-position deletions after final adds. Omitted SID/-2 is not record-only deletion.
- onAddCharactorSlot uses complete normal DTO with persistent id distinct from sid. Supported full reconstruction: all pre-sort deletes, then all final adds, then onItemSort([]) for source-free repaint. Added `bag_sort_refresh.go` uses this recovery strategy; application sort/merge rules unchanged. This is ordered reconstruction, not an atomic UI replacement; more callbacks than mapping-only. Unknown stale client-only records outside the original snapshot are not guaranteed repaired. Relogin is necessary to clear existing corrupted UI state.
- serchForGift is intentional spelling, Boolean Responder; false is no available gift. searchGameGift is separate push onGetItemIdAward with JSON text "[]".
- getPointAward/getShopAward Responder expects a top-level reward dictionary. Numeric keys are valid; existing made-up rows lacked itemType or referenced invalid table/template combinations and can cause unguarded GameData.d[row.itemType][row.itemId].name dereferences. Supported empty dictionary is {}, not fabricated valid-template replacement.
- Consume previews: onGetConsumeAwardList({},0); stage preview: onGetStageConsumeAwardList({stageNum:0},{}). getConsume empty object is supported.
- getActAwardInfo has no Responder; onUpdateActAwardInfo would require real type/actInfo/myInfo and event dates. Missing/zero timestamps render current dates in this client, so fake empty schedules are misleading. Register the method with argument validation and explicit onSystemSay that local new-server event is not configured; no invented ranking/eligibility or schedule callback. Full event functionality remains **BLOCKED by absent configuration**, not implemented by this compatibility fix.

## Validation / rollout

- PASS: item application/handler and activity suites, vet, host server build, focused diff whitespace check.
- PASS: isolated sort preserves quantities/binding separation, final slots readable and movable. Refresh regression ensures every delete precedes adds, final SIDs and record IDs retained, empty mapping repaint. Real Gain inventory untouched by tests.
- PASS: unconfigured read response tests verify no synthetic gifts/rewards/stages/totals; authentication enforced. New methods remain logged.
- PASS: Compose rebuilt/recreated only mcgame-server, command reported Started; PostgreSQL/Redis not reset or redeployed. No schema/data changes.
- BLOCKED: post-restart health check cannot connect to Docker Desktop Linux engine named pipe; engine unavailable after harness/server restart. Do not repeat deployment or infer current runtime health from earlier Started output.
- NOT TESTED: actual Flash sort/drag, new-server notification, gift/no-gift and recharge/shop rendering. Full gameplay acceptance remains pending operator retest after Docker/stack is running.
