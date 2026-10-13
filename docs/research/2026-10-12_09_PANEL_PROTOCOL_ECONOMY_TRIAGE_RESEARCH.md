# Panel protocol and economy triage

## Scope / safety

User reports quest/mall Flex sort2 crashes, PRS/petPVE/secret-hunt/monster-heart errors, sign-in wallet direction, crafting sale-bag nulls, silver voucher1132, furnace, special training, explorer-medal paid upgrade and disabled character deletion. Also requests target-level and configurable new-character gifts in dashboard. Do not modify ActionScript, reset/repair player balances, replay rewards, use vouchers or delete/grant live characters to reproduce. New dashboard feature plan is Draft, not approved.

Existing Docker stack is now running (DB/Redis healthy); no assistant database/Redis restart/reset was used in this follow-up. Current gameplay log is owner-driven, not assistant reproduction. Rebuilt/recreated only game server; startup17:34:09 loads5530 quests/itemAwards71, RTMP started, running/restart_count0. No post-deploy Flash gameplay was performed by assistant.

## Findings and scoped repairs

| Report | Evidence / finding | Status |
|---|---|---|
|Quest/mall sort2|Served Flash QuestManager sorts sort1=data.color, sort2=state; EnrichQuestDTO omitted color. Repeated add callback replaces by ID, not duplicate append.|Numeric catalog color added; app and callback regressions PASS; needs relog/Flash retest|
|PRS null|Actual13:26:46 initPRSPanel executes activity/backlog_panels.go starter and returns bag[]/data{}, despite real PRS handler registered earlier.|Legacy registration/malformed push removed; combined startup-order integration reproduces failure before fix and verifies real persisted DTO/no writes after|
|PetPVE error|Panel dereferences p.s0–s7; missing keys in empty state.|Panel and login default containers added without progression grants or DB repair; tests pending parent integration|
|Secret-hunt openflag Boolean|sthPlayIsInGroup returned Boolean but caller accesses openflag.|Closed object {openflag:false,group:false}; not an activated hunt. showPanel null remains unresolved/runtime UI availability|
|Monster-heart exp null|Empty heartBoxLev0 does not match template ladder starting1.|Default/read normalization to1, no persisted level repair; handler regressions PASS|
|Sign-in wallet|Gold+GoldBind was sent as gold; inflation on client while bound wallet stale.|Separate absolute gold/goldBind, unsupported goldDiff removed; bound/mixed/unbound tests PASS|
|Sign-in expenditure increases|Client text, seed thresholds and consume-delta persistence define cumulative expenditure.|Increase is expected; wallet decreases. Do not invert cumulative spend or repair history|
|Craft gems/material bags|EquipFuncBag dereferences getTemplateData(row.type,row.itemId,false); Core returns null without requesting missing data in this mode. Static shop36/66/67/70 JSON references all resolve; live hydration may differ.|NOT FIXED; need actual failing shop row/table/cache evidence, no invented template/enable callback. A purchase-handler fix cannot repair pre-purchase static lookup by itself|
|Bạch Ngân Lễ10van|Live useItem1132 returns cannot-use; catalog1132 type550/useType1 has no self-opening award. Description says unbound silver/bound silver, shorter info says gold voucher inconsistently.|Exact binding-aware100000 money/moneyBind rule; item-write failures pre-credit, overflow/count/level/target checks, single/multi callback tests PASS. No payout from shop gold100 field|
|Furnace|Registered paths used placeholder1 crystal per item record/score unit; client uses recycling table values and stack counts; separate writes risk lost inputs.|Registered RPCs now false+alert, no dependency/consumption/credit. Flash clears waiting state on false. Not a full furnace implementation; legacy source preserved unregistered|
|Special training|Live missing takeTXKCAdvanAward/takeTXKCFreeAward/openTXKCAdvanceCourse/openTXKCShop; getTXKCData is moving-window starter.|BLOCKED/NOT IMPLEMENTED paid/task/award system; no fabricated packages or paid success|
|Explorer medal|Normal init/score service exists; user log insufficient score for normal upgrade; levelUpEMedalByGold(cid,type) missing.|Paid upgrade NOT IMPLEMENTED; cannot claim whole medal absent or invent price/progress|
|Character delete gray|User confirms Flash selection screen. Checked-in CharSelectCanvas button7 initializes enabled=false; select toggles only visibility, no enabling assignment in class. Scheduling/restore RPCs and persistent deadline lifecycle absent.|BLOCKED; served-class parity still needs verification. No AS edit or invented enable callback/hard-delete|

## Dashboard research / plan

Current Progression exposes delta Up level/Add exp, not an absolute safe target. Direct backend lowering preserves previously earned higher stats/points. Creation hardcodes level90/Gold999999/GoldBind999999, once before save; selection doesn't grant. Box award editor is not creation-gift configuration. User selected **both increase/decrease**, **item/pet gifts**, and **Flash selection-screen deletion**; updated [Draft plan](../plans/2026-10-12_02_PLAYER_ADMIN_STARTER_DEFAULTS.md) accordingly, retaining required delivery/recalculation policy approval. No frontend functionality is claimed implemented by writing this plan.

## Verification boundaries

PASS: consolidated focused tests, affected vet, host/Linux builds, game-only deploy/startup. Key sort/voucher/PRS registration/heart/PetPVE/furnace response regressions repeated20 times PASS; sign-in suites also repeated20 times PASS. Full suite FAIL retains baseline four pet-stat, one auth PM EXP and two chat-stat failures, not a blanket PASS. Delegated final review and crafting implementation hit usage limits and did not complete; parent performed source spot-checks and reran tests, not a claimed successful independent review. No race-detector PASS or live economy/gameplay certification. Mid item-consumption then character-persistence failures remain nontransactional, report failure and never auto-replay. Draft dashboard plan still awaits detailed downgrade/delivery policies; nothing has been added to frontend merely by writing a plan.
