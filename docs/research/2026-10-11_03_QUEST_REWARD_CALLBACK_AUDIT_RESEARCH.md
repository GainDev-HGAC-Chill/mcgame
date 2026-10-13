# Quest reward callback compatibility audit

## Report and scope

Flash reported `onUpdateAchieve` #2095/#1063: expected 3 arguments, received 0 after completing a quest and taking its reward. Audited the entire `FinishQuest` push sequence and shared EXP/pet reward helpers against the recovered matching served client, not the incomplete checked-in CallBack stub.

Complete served callback source: `C:/Users/gaingain/AppData/Local/Temp/opencode/mcgame-callback-as-research/scripts/com/qeedoo/game/system/CallBack.as`. Client tracing was independently delegated read-only. No `.as`, served SWF, database state, or reward policy was modified for diagnosis.

## Confirmed defects and repair

1. `CallBack.as:3125–3154`: `onUpdateAchieve(type:int, id:int, data:Object)` updates achievement completion (`type=1`, `data.done`) or requirement progress (`type=2`, `data.done/progress`). It is not a generic refresh. `quest/lifecycle.go` sent it with zero arguments after every successful completion. Removed this placeholder; no fake achievement IDs, progress or completion values are sent. Existing valid farm achievement-delta callback remains unchanged.
2. `CallBack.as:12365–12369`: `onUpdateSafeBattleMid(mapIDs:Object)` updates the safe-battle map collection. The adjacent quest call also had zero arguments. Removed the placeholder rather than sending an empty collection that could erase client state. No authoritative safe-map changes are produced by this reward flow.
3. `CallBack.as:8621–8644`: `onDelCharactorSlot(itemID:Number, sid:int=-2)` expects item identity first; when SID is omitted it reads `sList[itemID].sid`. Quest turn-in incorrectly passed only SID. The consumed-item result now includes the original instance ID, and the handler sends `(itemID, sid)` explicitly. Partial consumption still sends the updated item DTO.

Archived `quest_handler.txt` contains historical copies of the bad placeholder calls; it is not compiled Go and was left untouched. Production searches show no remaining zero-argument SendCallback calls in the quest Go package.

## Other reward callbacks

Shared EXP `onUPP`/`onPlayerLevelUp`, currency/item grants, loop notifications and pet callbacks match client argument counts. Some client UI updates depend on normal in-scene initialization; the audit does not prove every UI branch safe. No speculative callback or gameplay modifications were added.

## Tests

`TestFinishQuestCallbackContractsAndConsumedInventory` drives actual TakeQuest → FinishQuest with in-memory repositories for full and partial item consumption. It verifies persisted quest completion, exact deletion `(instance ID, SID)`, remaining stack, absence of placeholder achievement/safe-map/`lvUp` pushes, and no callbacks on a repeated claim. Instance ID differs from SID to expose the old bug.

Focused regression, complete quest handler/application tests, NPC/utils/RTMP tests, affected vet and host build PASS. Live next-quest reward handling after deployment remains NOT TESTED until the operator verifies it. Do not repeat or reset a completed quest on Gain for testing.
