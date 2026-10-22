# Shop System

## Crystal credit-shop exchange (2026-10-23)

`exchangeItem(creditId, exNum)` now resolves real `TBL_CREDIT` rows instead of returning mock equipment 2001. The implemented authority is intentionally narrow: unlimited Crystal Shop rows with `shopId=1`, equipment `type=19`, primary score `cType1=32`, no secondary score, and valid equipment templates. Other credit shops remain unsupported rather than sharing an unverified currency mapping.

Credit score 32 maps to character currency `CurrencyWisdomCrystal` and Flash's intentional field typo `wisdonCrystal`. The service serializes purchases per character, validates cost/quantity, persists the debit before granting unbound/bound equipment according to the row, and compensates the currency if inventory grant fails. The responder remains truthy; callbacks send a complete `onAddItem` popup (`n/i/q/t/tt/c`), every changed `onAddCharactorSlot` DTO, and `onAddMoney(characterId,"wisdonCrystal",-cost,total)`.

## Dynamic standard-shop catalog publication (2026-10-19)

`getShopConfig` now serializes every cached `TBL_SHOP_SLOT` row in stable ID order using the exact `var obj={};obj.camelCase='value';...` format consumed by `CallBackGlobal.parseAndSetShopSlot`. Existing client rows are overridden except `position` (the Flash parser deliberately skips it); inserted rows enter the client's `sid`/`st` indexes. This does not publish or replace `TBL_SHOP` containers.

The NPC Admin editor therefore only targets existing standard-shop containers, keeps existing position immutable, and uses `st=5` instead of hard delete. Type-19 equipment and type-29 items are catalog-validated; point-only pricing is rejected because the current purchase service does not debit `point`. Database edits require an authorized server restart to refresh cache and a fresh Flash connection to invoke `getShopConfig`.

## NPC Direct Shop Callback Routing

Direct NPC template shops (`npc.ShopID > 0`) must always use the standard `onOpenShop` callback path from `internal/presentation/rtmp/handlers/npc/npc_click.go`.

`openShopById` is reserved for explicit special-shop scripts such as the handlers in `npc_script.go`. The Flash client binds those callbacks to different data sources:

- `onOpenShop` -> `TBL_SHOP` / `TBL_SHOP_SLOT`
- `openShopById` -> `TBL_CREDIT` / `ViewManager.NPCSHOP_CONFIG`

Shop IDs overlap between those systems, including low IDs such as `2`, so numeric heuristics like `shopID <= 7` are not valid for direct NPC shop routing. That overlap is what caused normal NPC shops such as `Tiệm Thuốc Ông Vương` to open the courage-shop panel (`NPC_SHOP_PANEL[100]`, "Shop Dũng Khí") instead of the standard shop UI.

## Shop Purchase Bag Slot Sync

Shop purchases can touch more than one bag slot in a single operation. A large buy may both fill an existing stack and create one or more remainder slots.

The Flash client only stays visually correct if the server sends `onAddCharactorSlot` for every changed bag slot, not just the last modified item. `internal/application/shop/purchase_sync.go` now snapshots matching bag items by `templateID`, `itemType`, and `isBound`, then returns `AddedItemDTOs` for every changed slot after `BuyItem`, `BuySystemItem`, and `BuySystemItemMulti`.

This keeps immediate shop updates aligned with later `bagSort` results and preserves the bind-state rule: stack only into slots with the same bound/unbound state, and send separate slot updates when the purchase creates a new remainder stack.
