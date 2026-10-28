# Pet Acquisition Achievement Authority — Research Addendum

**Date**: 2026-10-26  
**Scope**: Static research only for capture requirement 83, Wish Tree requirements 106–109, and the future Mystery Merchant requirements 102–105. No migration was applied and no player data was changed.

## Sources

- Vietnamese gameplay reference: [Cây Ước Nguyện](http://vuaphapthuat.go.vn/danh-muc-bai-viet/cay-uoc-nguyen/206.html) confirms that wishing can yield items, equipment, or Pets.
- Chinese-version reference: [4399 魔力学堂](https://web.4399.com/mlxt/) and a [4399 activity notice mentioning 许愿树](http://web.4399.com/mlxt/zx/651312.html). These confirm the feature name/context but do not define RPC, quality, or persistence contracts.
- Flash client evidence under `docs/client/` is authoritative for wire behavior.
- Catalog seeds, current Go services, and PostgreSQL functions are authoritative for current server persistence.

## 1. Catalog requirements

| Requirement | Achievement | Meaning | Current authority |
|---|---:|---|---|
| 83 | 2016 | Successfully capture one Pet | Unsupported |
| 102 | 2035 | Buy one green Pet from Mystery Merchant | Unsupported |
| 103 | 2036 | Buy one blue Pet from Mystery Merchant | Unsupported |
| 104 | 2037 | Buy one purple Pet from Mystery Merchant | Unsupported |
| 105 | 2038 | Buy one orange Pet from Mystery Merchant | Unsupported |
| 106 | 2039 | Receive any Pet from Wish Tree | Unsupported |
| 107 | 2040 | Receive a blue Pet from Wish Tree | Unsupported |
| 108 | 2041 | Receive a purple Pet from Wish Tree | Unsupported |
| 109 | 2042 | Receive an orange Pet from Wish Tree | Unsupported |

All targets are authority-defined because catalog `num=0`; each condition is a one-time event target of one.

## 2. Capture authority

- Capture is battle action `-70`, sent in `battleUpdateCmd`; there is no standalone capture RPC (`BattleCreatureView.as:838-855`, `Battle.as:845-850,1105-1131`).
- The server resolves capture in `internal/application/combat/catch.go`. It only reports success after `pet.Service.ContractPet` returns a persisted Pet.
- PostgreSQL assigns `character_pets.id` during `player.create_character_pet`; this instance ID is the stable business identity.
- The safe fact identity is `pet-captured:v1:<petID>`. The fact must be inserted in the same transaction as the new Pet. Emitting from `onAddPet` or after the transaction leaves a crash gap.
- Immediate UI delivery remains the existing server callback `onUpdateAchieve`; the Flash client never submits achievement progress.

## 3. Wish Tree authority

### Wire and reward semantics

- `getLottoData(null)` returns four arrays of 18 `TBL_PLAN` IDs and `highestAwardArr` (`LottoPanel.as:2032-2077,2700-2717`).
- `lottoByClient({type,num})` accepts types 1–3 and counts 1, 5, or 10. Its direct responder only needs a truthy success result (`LottoPanel.as:2692-2698,4005-4024`).
- Drawn rewards first enter the 500-slot Lotto holding bag. They are not granted directly to inventory or the Pet list (`LottoBagPanel.as:864-867,1036-1049,1076-1079`).
- `TBL_PLAN.ti=12` is a Pet reward and `ii` is the creature template ID. For Pet entries, `q` is grow rate multiplied by ten: the result bag calls `colorByGrowRate(q/10)` (`LottoBagPanel.as:2350-2353`).
- Canonical color bands are `1.1/1.3/1.6/2.0/2.5` for white/green/blue/purple/orange (`internal/domain/pet/quality.go`). Thus examples `12/15/19/23` normalize to `1.2/1.5/1.9/2.3`.

### Achievement boundary

The requested behavior is “when the wish rolls a Pet,” so the authoritative mutation is the successful holding-bag deposit, not the later claim. One durable fact is created per drawn Pet:

- requirement 106 for every `ti=12` result;
- requirement 107 when canonical color tier is blue;
- requirement 108 when canonical color tier is purple;
- requirement 109 when canonical color tier is orange.

A draw UUID plus result ordinal provides a stable identity: `wish-pet-drawn:v1:<drawUUID>:<ordinal>`. Payload must retain the plan ID, creature template ID, normalized grow rate, and canonical color tier. Draw facts, wallet debit/free-quota update, and holding-bag state must commit atomically.

### Existing defects affecting the boundary

- `internal/application/lotto/draw.go` currently updates Gold and Lotto state separately. With the active character cache, a crash can preserve the reward while losing the Gold debit, or debit Gold while losing the reward.
- Lotto entries do not retain plan ID or an operation ID.
- Pet claim is independently broken: `grantEntry` treats `ti=12` as a consumable. This does not change the draw-time achievement boundary and is not required to record requirements 106–109; it should be repaired as a separate transaction-safe feature.

## 4. Mystery Merchant status and marker contract

- No dedicated Mystery Merchant purchase RPC was recovered.
- Mystery Merchant NPC rows are type 4 plan NPCs with no `shop_id`; `TBL_NPC_PLAN` is empty and the current server does not implement their economy.
- Generic shop catalogs contain no direct `ti=12` slot and generic shop purchase paths do not create Pets.
- The existing `contractPet` RPC is a mismatched soul-contract endpoint and is not valid purchase authority.

Therefore requirements 102–105 must remain unsupported. The future transaction contract is:

- fact kind `mystery-merchant-pet-purchased`;
- stable identity `mystery-pet-purchased:v1:<purchaseID>:<petID>`;
- payload: purchase ID, Pet instance ID, creature template ID, normalized grow rate, canonical color tier, merchant/offer identity, and paid price;
- purchase ledger, currency debit, Pet creation, and fact insertion in one PostgreSQL transaction.

The shared append-fact primitive and domain fact shape may be prepared now, but no producer, synthetic purchase, backfill, or requirement activation is justified until the merchant protocol and catalog are authoritative.

## 5. Reusable fact pipeline

`player.character_achievement_facts`, pending-fact recovery on login, idempotent event recording, and `onUpdateAchieve` callback helpers already exist. The implementation should extend this pipeline rather than create counters:

1. business transaction inserts a stable fact with the authoritative state mutation;
2. the handler immediately processes pending facts and pushes real deltas;
3. login recovery processes facts left pending after a crash or callback failure;
4. event key `fact:v1:<factID>` prevents duplicate progress if processing retries.

## 6. Confirmed limitations

- No runtime smoke, migration apply, backfill, claim, or player mutation was performed.
- Flash `CallBack.as` did not decompile, but the callback payload is independently established by `AchievementPanel.updateView` and the current server helper.
- The Wish Tree Pet claim defect and generic shop crash-safety defects are recorded but remain outside this achievement-authority slice.
