# Actual auto-equip route, shop quest envelope and missing economic RPCs

## Runtime report

Operator: unequip visible, equip still invisible until relog; shop purchase invokes onAddChaQuest with undefined-property failure; exchange and convenience-bag install missing methods. No assistant purchase/use/exchange/equip or player-data repair performed.

Logs confirm:

-00:36:41/00:38:39/00:41:39: client calls `useItem(1,-1,8)` for equipment756, not equipOn. Service equips actual slot9, handler logs auto-equipped via use.
-00:35:26 buySystemItemClient("3494",1) creates instance39/template2262;00:42:42 purchase("3874",1) creates instance40/template2809. These purchases already succeeded before UI error. Do not replay/refund/regrant them automatically.
-exchangeAwardToGoldBind(1000) and setTempBag(2112,80) are missing methods.

## Confirmed auto-equip defect and correction

Previous patch covered explicit equipOn/equipOff, not the actual useItem equipment path. Application UseItem fetched a detached item, invoked EquipItem (which separately fetches/moves another snapshot), then returned the old bag snapshot as if it were equipped. Handler consequently used old bag SID as destination. Detached repository/cache semantics exposed this; alias-returning tests missed it.

Application now reloads item after EquipItem and returns the actual final equipment record. Presentation useItem reuses `refreshEquipmentSlots`: captured old bag SID and displaced old actual equipment SID; delete all originals before updating instance cache/adding final slots. Existing stat/element/active-list/appearance logic retained and HP/MP synchronization aligned with explicit equip. No item recreation, consumption, grant or template-rule change.

Regression tests: application detached repository returns equipmentSID10 rather than old2116; handler detached fixture exercises actual useItem(1,-1,instanceID), empty/replaced destination, ordering and final identity/SID. Original explicit equipment and unequip tests retained.

## Shop quest callback defect

Shop checkQuestUpdates invokes QuestNotifyHelper.PushQuestProgress, which sent raw EnrichQuestDTO as onAddChaQuest. Actual served callback export `.local/research/mcgame-callback-as-research/scripts/com/qeedoo/game/system/CallBack.as:11373–11378` unconditionally reads `param1.d.pos` and `param1.d.clsData`; raw DTO has no d.

Extracted existing quest-acceptance envelope builder and reused it for progress push: original enriched fields plus qid/string, st/state, d/enriched body, qn/active count. Root pos/clsData remain available for the callback's assignments. No invented quests/objectives/completion or rewards. Tests verify nested body and no recursive/mutated DTO. Shop purchase itself is not repeated to validate this fix.

## Remaining economic/container functionality — explicitly not implemented

Registered the two reported RPCs to produce onSystemSay **unavailable** notices after auth/argument validation; no success/award/bag-update callback and no persistence mutation. This replaces missing-method errors with an honest unsupported state, not a completed function.

-exchangeAwardToGoldBind: docs/client ExchangePanel:690–705 shows10 account award points per1 goldBind; actual callback onExAwardPointToGoldBind subtracts the received spent-point amount from UI. Existing gp handler reads account.Gold, shared across account characters. AccountRepository offers only generic Update, no coordinated atomic account-debit/character-credit operation. Need safe insufficient-balance, ownership, concurrency, rollback and write-behind cache integration before enabling. Never debit then separately credit and pretend transactional success. Do not send onExAwardPointToGoldBind on rejection.
-setTempBag: docs/client Slot:1146–1184 expects source regular-bag SID, destination installationSID80–84 and unlocked slot; source template must type517. Live source template2830 has t43200. Dedicated installed bags have item identity/capacity/expiry/replacement semantics, distinct from moving item contents to SlotTypeTempBag. Current BuildTBagPayload returns empty normal/mx tempBag metadata; counts alone are insufficient. Actual updateTempBags replaces full tBag object. Need persisted installation descriptors, duration/expiry and safe replacement/capacity behavior before consuming the item. No updateTempBags snapshot is forged.

Client researcher delegation failed due usage limit; parent directly cross-checked existing actual export and relevant partial panels for the above concrete contracts. No .as edits or new exports.

## Validation / deployment

-PASS: affected item application/handler, quest application/handler, exchange tests; shop compiles (no existing tests). Affected vet and host build.
-PASS: missing-RPC handlers reject invalid/authless requests; no repository calls, no fabricated success/credit/container state.
-PASS: deployed only mcgame-server; startup07:25:39 confirms itemAwards67 and RTMP started, running/restart_count0. No schema/seed/player changes or PostgreSQL/Redis redeploy/reset. Immediate startup is not sustained/gameplay acceptance.
-NOT TESTED: real Flash auto-equip and purchase callback after fix.
-BLOCKED/NOT IMPLEMENTED: actual voucher exchange and bag installation, despite compatibility handlers now registered. Do not mark these gameplayPASS.
