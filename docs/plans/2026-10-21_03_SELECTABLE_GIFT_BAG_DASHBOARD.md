# Selectable Gift-Bag Dashboard — Implementation Plan

**Date:** 2026-10-21 · **Status:** Code implemented; database verification blocked
**Goal:** Extend the existing Box Items dashboard and item runtime so an administrator can configure ordered choose-one reward packages. Opening such a package must use the existing Flash `itemMultitoSel` / `selMultiItemByIdx` flow instead of the fixed/weighted award picker. Item 3376 (`Túi Quà Cánh Xanh Dương`) is the reference case: choose one of equipment templates 1676, 1678 and 1677.

Research base:

- `docs/research/2026-10-21_04_SELECTABLE_GIFT_BAG_DASHBOARD_RESEARCH.md` — Flash wire contract, current server/dashboard gaps, exact blue-wing identities and proposed persisted metadata.
- `docs/memory/item-inventory.md` — authoritative item-use, award and existing selectable-package contracts.
- `docs/memory/dashboard.md` — current Box Items dashboard architecture and award editor behavior.
- `docs/research/2026-10-12_02_SHOP_BAG_AUDIT_RESEARCH.md` — classification of known selectable packages and the prohibition on inventing unresolved contents.

## 1. Confirmed mechanics

- Flash opens an owned package with `useItem(1, -1, packageItemInstanceID)`.
- The selection callback is `itemMultitoSel({packItemList:{index:{ti,ii,n,q}},packGetNum}, packageItemInstanceID)`.
- The confirmation RPC is `selMultiItemByIdx(packageItemInstanceID, selectedIdx)`.
- A responder result of `0` closes the panel, a positive value keeps it open with that many selections remaining, and a negative value leaves it unchanged.
- The confirmed selectable reward table types are `12` Pet, `19` equipment and `29` item.
- The package is not consumed when the panel opens. It is consumed after the configured number of distinct selections succeeds.
- Item 3376 is `Túi Quà Cánh Xanh Dương`; item 3846 is the orange package. The current 3376 order is `1676 Cánh Thiên Sứ`, `1678 Cánh Bích Không`, `1677 Cánh Ác Ma`.
- `data_tbl_item_award` rows currently take precedence over description/hardcoded selection and are interpreted as fixed/weighted awards. Selectable metadata must be detected before that picker.
- Runtime game-data changes require an authorized gameserver reload/restart.

## 2. Design

### 2.1 Single source of truth

Keep reward identity and grant metadata in `data.data_tbl_item_award`. Reserve this payload shape for selectable rows:

```json
{
  "selection": {
    "order": 0,
    "getNum": 1
  }
}
```

An item is selectable only when all of its award rows have valid selection metadata, unique contiguous order values and the same `getNum`. Mixed or malformed definitions fail closed. Existing rows without `selection` retain current fixed/weighted semantics.

### 2.2 Dashboard behavior

- Add a package behavior selector to `/dashboard/box-items/[id]`: existing fixed/random award behavior or player selection.
- In selectable mode, allow only Pet, equipment and item reward types; hide weight/guaranteed/random-range controls; expose selection count and ordered choice controls.
- Preserve reward count, quality, equipment prefix and non-reserved payload fields.
- Display an explicit warning that gameserver restart/reload is required.
- Save the complete award snapshot, including removals and order, through one validated PostgreSQL function call. Reject stale/mixed ownership and invalid item templates rather than partially saving rows.
- Require selectable packages to be existing `type in (508,550)` and `use_type=1` item templates.
- Add same-origin mutation validation and strict request validation to the award endpoint.

### 2.3 Runtime resolution

- Add helpers on `ItemAwardTemplate` for parsing selection metadata without mutating cached payloads.
- Return item awards in deterministic selection order, with ID order as the fallback for ordinary rows.
- In `questItemEffectHandler`, detect a complete selectable award definition before ordinary award handling and build `GiftBoxData` from those rows.
- Extend `GiftBoxChoice` with the persisted award metadata needed to grant the exact selected row. Do not infer contents from package description once a valid selectable definition exists.
- Refactor award granting so selection reuses the same Pet/equipment/item validation, capacity checks, quality, prefix, binding/material payload and callback result structures as ordinary award rows.
- Keep the pending snapshot keyed by character and package item instance. Validate ownership, template identity, selected index and current stack on every selection.
- Reject `useMultiItem` for selectable packages without consuming anything.
- Preserve description/hardcoded fallback for selectable packages with no configured award rows. Once 3376 is represented and verified by data, remove its hardcoded choice list and special color override only if the configured payload reproduces the current visible blue result exactly.

### 2.4 Persistence and migration

- No new reward table or column is required; selection metadata uses existing `payload jsonb`.
- Add a schema-qualified, `security invoker` PostgreSQL function for atomic full-snapshot save of one package's award rows. Revoke execution from `anon`/`authenticated`; dashboard service role remains the caller.
- Apply the function to the compose database, verify rollback/error cases, then capture the migration through the repository Supabase CLI workflow. Do not hand-write the final migration.
- Export changed `data_tbl_item_award` rows to the canonical seed only when actual package configurations are saved. The feature implementation itself must not invent contents for unresolved packages.

### 2.5 Touch points

| Touch point | Change |
|---|---|
| `internal/gamedata/models/item_award.go` | Parse and validate reserved selection metadata. |
| `internal/gamedata/cache.go` | Return item awards deterministically and expose valid selectable definitions. |
| `internal/application/item/effect.go` | Carry exact persisted choice metadata in `GiftBoxChoice`. |
| `internal/application/item/effect_quest_item.go` | Route valid configured choices before the weighted award picker. |
| `internal/application/item/effect_award.go` | Extract reusable single-row resolution/grant and capacity logic for selectable types. |
| `internal/application/item/effect_gift_box.go` | Keep description fallback; retire 3376 hardcode only after data parity is proven. |
| `internal/application/item/service.go` | Reject multi-use for interactive selectable packages. |
| `internal/presentation/rtmp/handlers/item/use.go` | Emit configured choices and grant the selected row through shared application logic. |
| `internal/presentation/rtmp/handlers/item/handler.go` | Store validated pending choice snapshots without broad global serialization. |
| `dashboard/src/lib/box-item-awards.ts` | Define mode/selection payload constants and selectable type rules. |
| `dashboard/src/services/box-item.service.ts` | Add package mode, selection count and ordered snapshot types. |
| `dashboard/src/app/api/box-items/awards/route.ts` | Strict validation, same-origin guard and atomic snapshot RPC. |
| `dashboard/src/app/dashboard/box-items/_components/box-item-award-workspace.tsx` | Mode selector, ordered choice editor, selection-count input and restart warning. Split components if needed to stay under repository size limits. |
| `supabase/data/selectable_gift_bag_admin.sql` | Draft the validated atomic save function before migration capture. |
| generated migration + `supabase/seeds/data_tbl_item_award.sql` | Capture function schema and export only approved example/config rows. |

## 3. Non-goals

- Do not edit ActionScript or introduce new RPC names; the existing Flash panel is sufficient.
- Do not support currency, EXP, buff, title, skill, medal, mount or Fairy as selectable panel entries until their Flash display/capacity contract is independently proven.
- Do not infer package contents from names alone or bulk-convert the unresolved selectable-package audit list.
- Do not guess item 3846's exact orange quality within 21-25.
- Do not add live game-data hot reload in this feature; dashboard must disclose the restart requirement.
- Do not redesign ordinary fixed/weighted award probabilities.
- Do not claim full cross-domain transaction atomicity for legacy selection. Preserve current consumption timing, add preflight/revalidation, and document the residual grant-before-consume risk.
- Do not restart/deploy, mutate player data, commit or push without explicit authorization.

## 4. Verification

1. Go model/application tests:
   - valid selectable metadata, deterministic order and `getNum`
   - mixed/malformed/duplicate/non-contiguous metadata fails closed
   - configured rows take precedence over ordinary award picker and description parsing
   - Pet/equipment/item choice carries exact count, quality, prefix and supported payload
   - unsupported award types and `useMultiItem` do not consume the package
   - existing fixed/weighted and description-based package tests remain unchanged
2. RTMP handler tests:
   - exact `itemMultitoSel` payload for 3376-equivalent configuration
   - stale item, wrong item instance, bad index, duplicate selection, capacity/grant failure and consume failure return negative without false success callbacks
   - successful reward callback precedes package stack/delete callback; responder returns `0` when complete
3. Dashboard tests:
   - mode conversion preserves rows but writes/removes reserved selection metadata correctly
   - selectable mode only accepts types 12/19/29
   - order/getNum validation and complete-snapshot deletion
   - invalid item type/useType, cross-item row ID, stale revision, non-same-origin mutation and partial payload rejection
4. Database verification:
   - atomic save/rollback, concurrent ID allocation, full replacement, mode conversion and service-role-only execution
   - inspect 3376 readback if the operator approves seeding it; do not use a player-owned package
5. Commands:

```text
go test ./internal/gamedata ./internal/application/item ./internal/presentation/rtmp/handlers/item -count=1
go vet ./internal/gamedata ./internal/application/item ./internal/presentation/rtmp/handlers/item
go build -o bin/gameserver.exe ./cmd/gameserver
corepack pnpm --dir dashboard typecheck
corepack pnpm --dir dashboard test
git diff --check
```

6. User-driven Flash smoke test after authorized apply/restart:
   1. Configure a disposable test package definition or approved 3376 definition in the dashboard.
   2. Restart/reload gameserver under operator control.
   3. Double-click one owned package and confirm the three ordered choices appear.
   4. Select one reward; confirm only that reward is granted and exactly one package is consumed.
   5. Repeat with a full bag and an invalid/stale panel; confirm no package or reward changes.
7. Update `docs/memory/item-inventory.md`, `docs/memory/dashboard.md`, `docs/ai-memory/FIX_HISTORY.md` and `docs/ai-memory/WORK_STATE.md` after implementation.
