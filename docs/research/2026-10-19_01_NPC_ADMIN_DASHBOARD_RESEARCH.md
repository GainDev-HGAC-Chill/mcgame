# NPC Admin Dashboard Research

**Date**: 2026-10-19  
**Status**: Complete for planning; no implementation, migration, runtime restart, or data mutation performed.

## 1. Requested capability

Add an NPC management area to the existing dashboard that brings together:

- the NPC's ambient dialogue;
- quests related to that NPC;
- the NPC's standard shop and its slots.

The safe implementation must preserve the existing Quest Admin revision/audit rules, must not treat the legacy NPC `qid` string as canonical quest ownership, and must not confuse standard `TBL_SHOP` shops with `TBL_CREDIT` special shops.

## 2. Existing dashboard and persistence baseline

- There is no `/dashboard/npcs` page, `/api/npcs` route, NPC hook/type module, or NPC/shop admin SQL API.
- Quest Admin is the closest established pattern: server-only service-role RPC calls, Zod validation, optimistic content revisions, idempotent request IDs, before/after audit images, table locking, and same-origin write checks.
- Quest Admin already edits `start_text`, `complete_text`, `start_npc`, and `finish_npc`. Duplicating those mutations in an NPC save path would create two catalog writers with different validation/audit behavior.
- Current dashboard access is intentionally internal-network based and unauthenticated for Quest Admin. This feature must not claim that application login protects the new page or expand its network exposure.

## 3. Catalog relationships

### 3.1 NPC dialogue

`data.data_tbl_npc.on_service_text` is the ambient NPC greeting. The Flash `Npc` object has an `onServiceText` property, and `NpcFuncPanel` both renders it in the greeting area and calls `npc.view.onSay(onServiceText)` when there is no higher-priority function.

Quest conversations are separate fields in `data.data_tbl_quest`:

- `start_text` is the acceptance dialogue;
- `complete_text` is the completion dialogue;
- `start_npc` and `finish_npc` determine which NPC offers and receives the quest.

The checked catalog contains 1,943 NPCs; 747 have non-empty `onServiceText`.

### 3.2 Quest links

The canonical runtime relationship is the reverse join from quests:

- `data_tbl_quest.start_npc = npc.id` for offered quests;
- `data_tbl_quest.finish_npc = npc.id` for turn-ins;
- `data_tbl_quest_loop.nid = npc.id` for shared loop definitions.

The server confirms this contract: `quest.Service.GetAvailableQuestsForNpc` scans `GetAllQuests()` and compares `StartNPC`; active turn-ins compare `FinishNPC` in `buildNPCInteractionData`.

`data_tbl_npc.qid` is not canonical general quest ownership. It is a sparse type-specific pipe-delimited field:

- only 38 of 1,943 NPC rows contain parseable IDs;
- 9 referenced IDs are absent from the checked quest catalog;
- 29 entries do not point to a quest whose start or finish NPC is that row;
- the server NPC interaction path does not read `NpcTemplate.Qid`.

Therefore the dashboard may show `qid` as a legacy diagnostic, but must not use it to build or rewrite the normal linked-quest list.

The checked catalog has 380 distinct NPC IDs referenced by at least one quest start/finish field. Nineteen quest rows contain at least one positive NPC reference missing from the checked NPC catalog; these are legacy warnings, not authorization for automatic repair.

### 3.3 Standard shops

For a direct NPC shop:

```text
data_tbl_npc.shop_id -> data_tbl_shop.id
data_tbl_shop_slot.sid -> data_tbl_shop.id
data_tbl_shop_slot.(type,item_id) -> item/equipment template
```

The server's direct route sends `onOpenShop(shopId)`. The Flash `ShopPanel` reads the shop from `TBL_SHOP` and the product list from the `TBL_SHOP_SLOT` index keyed by `sid`.

This is distinct from `openShopById`, `TBL_CREDIT`, and `ViewManager.NPCSHOP_CONFIG`. Numeric IDs overlap, so no ID range heuristic is valid.

Checked-catalog observations:

- 102 NPC rows reference shops, covering 95 distinct shop IDs; none of those references is missing.
- 57 shop-linked NPCs are also linked to quests, so a combined NPC detail view is necessary.
- The 95 referenced shops include 68 type-1 and 27 type-2 containers.
- The catalog has 2,542 shop slots. Every checked slot has a valid type-19 equipment or type-29 item target.
- Legacy anomalies exist: 122 slots reference an absent shop container and 360 `(sid, position)` collisions exist. A new editor must preserve/display these anomalies and reject creating new collisions; it must not globally clean them.

## 4. Runtime and Flash activation boundaries

### 4.1 Server cache

`gamedata.Manager.LoadAll` loads NPC, quest, shop, and shop-slot tables into process memory at startup. `Cache.LoadTable` overwrites rows that are present but does not clear removed rows and is not an atomic multi-table reload mechanism. A database save is therefore not an immediate runtime activation.

No restart, deploy, or live cache mutation is authorized by this research.

### 4.2 NPC ambient dialogue

The Flash `Npc` class is dynamic and accepts `onServiceText`, but the current scene DTO contains only ID/position/name/resource/type. `GetMapNPCsForClient` and `buildNPCList` do not send `onServiceText`. Consequently a DB edit plus server restart is still insufficient for the greeting unless the server scene payload is extended to include the field. This can be done without editing ActionScript because `Core.createNpc` copies the server object into the dynamic `Npc` instance.

### 4.3 Quest catalog

The prior definitive activation audit remains binding: complete quest catalog edits cannot be activated consistently in the compiled Flash client. The embedded quest/index tables are built before the delayed ChFields overlay; the shipped overlay only changes existing quest text/EXP and cannot add/delete rows or rebuild indexes after `startNpc` changes.

The NPC page can safely list reverse relationships from the database and deep-link to Quest Admin, but it must retain Quest Admin's “database saved / Flash activation blocked” warning. Restart/relogin alone is not a complete quest activation solution.

### 4.4 Shop-slot transport

Unlike the quest catalog, standard shop slots have an existing dynamic transport:

- the client calls `getShopConfig` during line-list initialization;
- `onGetShopConfig` parses an array of serialized slot assignments;
- `parseAndSetShopSlot` updates existing slot objects and maintains the `sid` and `st` indexes;
- the server handler currently returns an empty list.

This is sufficient authority to publish current standard shop-slot values after a server restart/reconnect, with important limits:

- the parser deliberately does not update `position` on existing embedded rows;
- it does not dynamically replace `TBL_SHOP` container rows;
- hard-deleting a DB row gives the client no tombstone, while setting `st=5` can hide it;
- newly indexed slots are suitable for `ShopPanel`, but unrelated panels that scan the primary embedded array may not discover them.

Therefore the safe dashboard scope is existing shop containers, slot create/update, and soft hide. Existing slot position and shop container name/type remain read-only.

## 5. Mutation safety

The NPC aggregate has shared effects:

- one shop can be referenced by multiple NPCs;
- one quest links two NPCs and may have active player progress;
- shop-slot IDs are global across all shops.

Safe writes need the same class of controls as Quest Admin:

- optimistic content revision;
- idempotent request UUID;
- same-transaction before/after audit;
- table lock covering NPC/shop/shop-slot ID allocation;
- row ownership checks (`slot.sid` must equal the edited shop);
- explicit shared-shop impact count and confirmation;
- strict target and price validation;
- no direct player inventory/currency/progress mutation.

Quest relationship mutations should remain owned by `quest_admin_save`; the NPC page should navigate to that editor rather than introduce a second writer.

## 6. Recommended safe product scope

1. `/dashboard/npcs` list with search, type/map/shop/dialogue filters.
2. `/dashboard/npcs/[id]` with three tabs:
   - **NPC & dialogue**: identity/context plus editable `name`, `on_service_text`, and existing `shop_id` assignment;
   - **Related quests**: offered/turn-in/loop rows, dialogue previews, legacy `qid` diagnostics, and links to the existing Quest Admin editor;
   - **Shop**: shared-impact warning and existing-container slot editor.
3. Shop-slot writes support type 19/29 targets, prices used by the implemented purchase service, quantity/quality/status, alternative currency pairs, insert/update, and `st=5` soft hide.
4. Extend scene NPC payload with `onServiceText`.
5. Populate the existing `getShopConfig` callback from the server's current shop-slot cache so DB-backed slot edits can reach a freshly connected client after an authorized server restart.
6. Continue reporting activation honestly: NPC/shop changes require an authorized server restart and client scene/reconnect refresh; quest structural activation remains blocked by the compiled catalog contract.

## 7. Explicit exclusions

- No ActionScript or compiled Flash asset edits.
- No NPC creation/deletion, placement, sprite, schedule, combat roster, script, transport, or type editing.
- No new `TBL_SHOP` container creation or container type/name mutation.
- No `TBL_CREDIT` special-shop editor.
- No hard delete or reordering of existing shop slots.
- No automatic quest-link rewrite, legacy `qid` cleanup, orphan repair, active-progress rewrite, player-data mutation, restart, or deploy.

## 8. Sources

- `docs/client/object/Npc.as`
- `docs/client/view/view/compDragable/NpcFuncPanel.as`
- `docs/client/view/view/compDragable/ShopPanel.as`
- `docs/client/system/CallBackGlobal.as`
- `docs/client/data/DataManager.as`
- `internal/application/quest/service.go`
- `internal/presentation/rtmp/handlers/npc/npc_interaction.go`
- `internal/presentation/rtmp/handlers/npc/shop_routing.go`
- `internal/application/scene/service.go`
- `internal/presentation/rtmp/handlers/scene/world.go`
- `internal/application/shop/service.go`
- `internal/presentation/rtmp/handlers/shop/config.go`
- `supabase/data/quest_admin_*.sql`
- `docs/research/2026-10-12_14_QUEST_ADMIN_FLASH_ACTIVATION_RESEARCH.md`
- checked JSON catalogs for `TBL_NPC`, `TBL_QUEST`, `TBL_SHOP`, and `TBL_SHOP_SLOT`
