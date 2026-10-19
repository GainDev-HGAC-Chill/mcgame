# NPC Admin Dashboard — Implementation Plan

**Date**: 2026-10-19 · **Status**: Implemented
**Verification**: Code/tests complete; final SQL hardening re-apply is pending because the compose Docker engine is offline.
**Goal**: Add a safe Vietnamese dashboard for browsing NPCs, editing ambient dialogue and standard-shop assignment, reviewing linked quests, and managing slots in an existing standard shop without duplicating Quest Admin ownership or changing player state.

Research base (verified 2026-10-19):

- [`../research/2026-10-19_01_NPC_ADMIN_DASHBOARD_RESEARCH.md`](../research/2026-10-19_01_NPC_ADMIN_DASHBOARD_RESEARCH.md) — current dashboard gap, catalog joins, checked-data anomalies, Flash/runtime activation limits, and safe scope.
- [`../research/2026-10-12_14_QUEST_ADMIN_FLASH_ACTIVATION_RESEARCH.md`](../research/2026-10-12_14_QUEST_ADMIN_FLASH_ACTIVATION_RESEARCH.md) — proves complete quest catalog activation remains blocked in the compiled client.
- [`../memory/npc-quest-system.md`](../memory/npc-quest-system.md), [`../memory/quest-admin.md`](../memory/quest-admin.md), and [`../memory/shop-system.md`](../memory/shop-system.md) — current runtime and admin contracts.

## 1. Confirmed mechanics

- Ambient NPC dialogue is `data.data_tbl_npc.on_service_text`; Flash reads it as `Npc.onServiceText` in `NpcFuncPanel`.
- Quest acceptance/completion dialogue is `data_tbl_quest.start_text` / `complete_text`. Runtime NPC relationships are `start_npc`, `finish_npc`, and loop `nid`.
- `data_tbl_npc.qid` is sparse type-specific legacy data and is not read by the server's normal NPC quest relationship path. It is diagnostic only.
- Direct standard shops route through `onOpenShop(shopId)` and use `TBL_SHOP` plus `TBL_SHOP_SLOT`. `openShopById`/`TBL_CREDIT` is a different system even where numeric IDs overlap.
- A shop may be shared by several NPCs. Shop-slot IDs are global, and a slot belongs to a shop through `sid`.
- Flash calls `getShopConfig` during line-list initialization. `onGetShopConfig([0,"",rows])` accepts serialized slot assignments and updates the `sid`/`st` indexes. The current server returns an empty row list.
- The existing parser does not update `position` on an existing embedded slot. It can hide a slot through `st=5`; a missing DB row is not a client tombstone.
- The server loads NPC/quest/shop catalogs into memory at startup. This feature does not introduce live cache swap or automatic restart.
- Extending the scene NPC object with `onServiceText` is compatible with the dynamic Flash `Npc` object and `Core.createNpc`; no ActionScript change is needed.
- Quest Admin remains the sole writer for quest text, `start_npc`, `finish_npc`, and loop definitions. Complete Flash activation of those edits remains blocked; restart/relogin is not sufficient proof.

## 2. Design

### 2.1 Aggregate and ownership

The NPC detail read model is one database snapshot:

```text
{
  npc,
  offeredQuests,
  turnInQuests,
  loops,
  legacyQidDiagnostics,
  shop: { container, slots, referencingNpcs } | null,
  revision,
  warnings
}
```

`npc_admin_save` owns only:

- NPC `name`, `on_service_text`, and `shop_id`;
- slot insert/update/soft-hide rows belonging to the NPC's currently selected existing shop.

Quest rows are read-only in this aggregate and link to `/dashboard/quests/[id]`. Shop container ID/name/type and existing slot `sid`/`position` are read-only.

### 2.2 Transaction, revision, and audit

Create service-role-only SQL functions following the Quest Admin pattern:

- `npc_admin_list(...)`
- `npc_admin_detail(p_id)`
- `npc_admin_lookups(p_kind, ...)`
- `npc_admin_validate(...)`
- `npc_admin_save(...)`
- `npc_admin_verify()`

`npc_admin_save` must:

1. validate actor/request/reason/body bounds;
2. lock `data_tbl_npc`, `data_tbl_shop`, and `data_tbl_shop_slot` in deterministic order;
3. build the current snapshot and compare the expected MD5 revision;
4. enforce idempotency by request UUID and request hash;
5. verify the selected shop exists, every updated slot belongs to it, and the expected shared-NPC count is unchanged;
6. validate all changed fields and references;
7. write a before-image audit before catalog mutation in the same transaction;
8. allocate inserted slot IDs under the table lock;
9. convert “remove” to `st=5`, never delete the row;
10. write the after-image/new revision and return the canonical detail.

Create a dedicated `data.npc_admin_audit` table rather than overloading the quest-named audit table. Enable RLS, expose no anon/authenticated policy, and grant only service-role read/insert/update required by the RPCs.

Before applying any SQL, record a checkpoint and take an operator-approved compose backup of the three catalog tables. Apply reviewed function drafts to compose, smoke-test with transaction rollback, then capture the schema change with `supabase db diff --db-url "$DB_URL" -f npc_admin_dashboard`; do not hand-write a migration.

### 2.3 Validation

- NPC name: non-empty, at most 300 characters.
- Ambient dialogue: nullable text, at most 16,000 characters.
- Shop assignment: null/zero means no direct shop; positive IDs must exist in `data_tbl_shop`.
- Slot target: only `(type=19, equipment)` or `(type=29, item)`, and the target must exist.
- New slots: immutable `sid` from the selected shop; unique global ID allocated server-side; position must be positive and must not create a new active `(sid, position)` collision.
- Existing slots: `id`, `sid`, and `position` immutable.
- Price/currency pairs must be internally complete and non-negative. `p_type1/p_num1` and `p_type2/p_num2` are accepted only for currency IDs implemented by both the Flash display mapping and server `DeductCurrency` registry.
- A newly edited slot must have an implemented positive purchase cost: supported `p_type1`, gold, money, or a verified item-template fallback. Do not enable point-only standard NPC rows through this editor because `BuyItem` does not deduct `point`.
- Preserve unsupported legacy rows unchanged and show warnings; do not normalize them merely by opening/saving the NPC.
- `amount` is labelled as the client purchase-selector limit, not guaranteed server stock enforcement.
- `st=0` is active and `st=5` is hidden. Other legacy statuses display read-only until separately authorized.

### 2.4 Dashboard/API

- `/dashboard/npcs`: paginated search by ID/name, filters for map/type/dialogue/shop, and badges for offered/turn-in quests and shop presence.
- `/dashboard/npcs/[id]`: tabs **NPC & đối thoại**, **Nhiệm vụ liên kết**, and **Cửa hàng**.
- Related quests show relationship role, quest type/level, relevant dialogue preview, activation warning, and an edit link to existing Quest Admin.
- Shop tab shows every NPC sharing the shop and requires explicit shared-impact confirmation before slot writes.
- Use the Quest Admin request/error patterns: server-only service-role client, same-origin write guard, strict Zod schemas, 128 KiB body limit, optimistic 409 conflict, 422 validation errors, and Vietnamese UI messages.
- Every save requires a reason and generated request UUID. Return `databaseSaved=true`, `serverRestartRequired=true`, and the precise Flash refresh warning.

### 2.5 Runtime presentation

- Add `OnServiceText` to `scene.NPCClientData`, populate it from `NpcTemplate`, and emit `onServiceText` in `buildNPCList`. Add focused tests for exact camel-case payload spelling.
- Add deterministic standard-shop slot serialization from the loaded cache and return it through existing `GetShopConfig` as `[0,"",rows]`.
- Serialize the fields consumed by `parseAndSetShopSlot` using the exact `obj.<field>='<value>';` format and camel-case slot names (`itemId`, `priceAll`, `pType1`, `pNum1`, `pType2`, `pNum2`). Escape or reject values so no quote/semicolon injection is possible; current slot values are numeric.
- Include all current standard shop slots on a fresh connection so existing DB changes override the embedded rows and inserted rows enter the `sid`/`st` indexes.
- Do not add the existing RPCs to `quietMethods`; they are already quiet and this plan introduces no new Flash RPC name.

Activation after an approved deployment is:

1. restart the game server to load the saved catalog;
2. reconnect the Flash client so `getShopConfig` runs;
3. leave/re-enter the map so scene NPCs are recreated with current ambient dialogue.

This does not make quest structural activation safe; the UI must keep that warning.

### 2.6 Touch points

| Touch point | Change |
|---|---|
| `dashboard/src/components/sidebar.tsx` | Add Vietnamese NPC management navigation. |
| `dashboard/src/app/dashboard/npcs/page.tsx` | Searchable/filterable NPC list. |
| `dashboard/src/app/dashboard/npcs/[id]/page.tsx` | Detail shell, dirty state, optimistic save, reason/shared-impact confirmation. |
| `dashboard/src/app/dashboard/npcs/_components/*` | Dialogue, related-quest, shop-slot, lookup, warning, and save components kept below the repository file-size limit. |
| `dashboard/src/app/api/npcs/route.ts` | Validated list endpoint. |
| `dashboard/src/app/api/npcs/[id]/route.ts` | Detail GET and aggregate PATCH. |
| `dashboard/src/app/api/npcs/lookups/route.ts` | Existing shop/item/equipment/map lookup endpoint. |
| `dashboard/src/lib/server/npc-admin.ts` | Shared route/RPC/error/revision helpers mirroring Quest Admin. |
| `dashboard/src/lib/npcs/validation.ts` | Strict list/save/lookup Zod contracts. |
| `dashboard/src/types/npcs.ts` | Exact NPC aggregate and slot types. |
| `dashboard/src/hooks/use-npcs.ts` | Fetch/mutation hook with no background overwrite of dirty edits. |
| `dashboard/tests/npc-*.test.mjs` | Validation, permissions/same-origin, editor patching, and shared-impact tests. |
| `supabase/data/npc_admin_read.sql` | List/detail/lookups/snapshot functions and audit table draft. |
| `supabase/data/npc_admin_validate.sql` | Catalog, currency-pair, ownership, and collision validation. |
| `supabase/data/npc_admin_save.sql` | Locked, revisioned, idempotent aggregate mutation. |
| `supabase/data/npc_admin_verify.sql` | Read-only integrity and privilege verification. |
| `supabase/migrations/<generated>_npc_admin_dashboard.sql` | Compose-generated diff only after approved application/testing. |
| `internal/application/scene/service.go` | Carry ambient dialogue in `NPCClientData`. |
| `internal/presentation/rtmp/handlers/scene/world.go` | Emit exact `onServiceText` field in scene NPC payloads. |
| `internal/gamedata/cache.go`, `internal/gamedata/manager.go` | Add deterministic all-shop-slot accessor used by shop config serialization. |
| `internal/presentation/rtmp/handlers/shop/config.go` | Populate existing `getShopConfig` response from cached slot rows. |
| matching `*_test.go` files | Exact scene payload and shop-config serialization/order/escaping regressions. |

## 3. Non-goals

- No NPC create/delete, map position, sprite/resource, schedule, transport, combat roster, script, type, or class-gate editor; these affect scene/gameplay systems beyond the request.
- No quest mutation from the NPC aggregate. Quest Admin remains the only quest writer, preserving its revision/validation/audit and active-progress warning behavior.
- No claim that Quest Admin edits are Flash-activated; compiled catalog/index publication remains blocked.
- No new shop container and no container name/type edit because there is no verified dynamic `TBL_SHOP` replacement path.
- No `TBL_CREDIT` special shop, VIP rotation, limit/remain-stock, sale, `r`, `price_all`, or hard-delete semantics.
- No existing-slot reorder because the Flash override parser ignores `position` updates.
- No automatic restart/deploy, cache hot swap, online-player disconnect, player inventory/currency/progress write, seed rewrite, or legacy anomaly cleanup.
- No attempt to make the existing non-atomic shop purchase transaction atomic; that is an independent gameplay task and this editor must not weaken it.

## 4. Verification

1. **SQL rollback tests**: list/detail joins; stale revision; idempotent retry/conflict; wrong slot ownership; shared-shop impact drift; invalid target/currency pair; duplicate position; audit failure rollback; soft hide; inserted global ID; unrelated row/table/player fingerprints unchanged.
2. **Dashboard tests**: list query bounds, strict save envelope, same-origin write rejection, legacy warning preservation, quest deep links, shared confirmation, dirty-state retention, and API error mapping.
3. **Go tests**:
   - `go test ./internal/application/scene ./internal/presentation/rtmp/handlers/scene -count=1`
   - `go test ./internal/gamedata ./internal/presentation/rtmp/handlers/shop -count=1`
   - targeted `go vet` for the same packages.
4. **Dashboard checks** from `dashboard/`: `corepack pnpm test`, `corepack pnpm typecheck`, and the repository's production build; record the known Windows standalone symlink limitation separately if it recurs.
5. **Build**: `go build -o bin/gameserver.exe ./cmd/gameserver`.
6. **Static checks**: scoped `git diff --check`; inspect migration diff for only intended audit/functions/grants and no data inserts.
7. **Manual dashboard smoke** on fixtures only: browse NPC; verify ambient text; inspect start/finish/loop links; attach an existing fixture shop; create/edit/soft-hide a fixture slot; force a stale-revision conflict; confirm shared-shop warning. Roll back fixture transaction or restore only the isolated fixture rows.
8. **Flash smoke after separate deploy/restart approval**: reconnect a test client, verify `getShopConfig` applies a fixture slot, reopen the standard shop, leave/re-enter the fixture map, and verify ambient dialogue. Do not buy an item, spend currency, use Quest 166, Pet Nấm Lục, or real player/item data.
9. Update `docs/memory/dashboard.md`, `docs/memory/npc-quest-system.md`, and `docs/memory/shop-system.md`; update `FIX_HISTORY.md` and `WORK_STATE.md`. No new knowledge-base index entry is needed.
