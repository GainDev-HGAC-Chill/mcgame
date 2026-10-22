# Selectable Gift-Bag Dashboard Research

**Date:** 2026-10-21
**Scope:** Data-driven dashboard configuration for packages where the player selects rewards instead of receiving a fixed or random award. No ActionScript changes.

## 1. Executive findings

- The Flash client already supports selectable packages end to end. The initial call is `useItem(1, -1, packageItemInstanceID)`, the server opens the panel with `itemMultitoSel`, and the player confirms through `selMultiItemByIdx(packageItemInstanceID, selectedIdx)`.
- The current Go server implements that protocol, but selectable contents come from description parsing or three hardcoded packages. Award rows in `data.data_tbl_item_award` take precedence and currently mean fixed/weighted awards, so adding ordinary rows through the dashboard can silently turn a selectable package into a random/fixed package.
- The existing `payload jsonb` column can carry reserved selection metadata without adding a second reward table. A package is selectable only when every row has a valid, consistent selection descriptor. Mixed or invalid definitions must fail closed.
- The existing dashboard detail workspace `/dashboard/box-items/[id]` already provides catalog lookup, icons, drag/drop, counts, quality and equipment prefix controls. It is the correct UI surface for adding a selectable mode.
- Runtime game data is loaded at gameserver startup. Dashboard changes do not affect an already running process until an authorized restart/reload.

## 2. Correct example identity

The blue package in the request is item **3376**, not item 3846.

| Package | Catalog name | Description intent | Type/useType |
|---|---|---|---|
| `3376` | `Túi Quà Cánh Xanh Dương` | Choose one of three blue wings | `550/1` |
| `3846` | `Túi Quà Cánh Cam` | Choose one of the same three wings at orange quality | `550/1` |

Evidence:

- `docs/database/game_data/TBL_ITEM_TEMPLATE.json:108972-109007`
- `docs/database/game_data/TBL_ITEM_TEMPLATE.json:122662-122697`
- `supabase/seeds/data_tbl_item_template.sql:5830-5831`

The three rewards are equipment templates:

| Wing | Equipment ID | Table type |
|---|---:|---:|
| Cánh Thiên Sứ | `1676` | `19` |
| Cánh Ác Ma | `1677` | `19` |
| Cánh Bích Không | `1678` | `19` |

The existing 3376 hardcode uses display order `1676, 1678, 1677`, matching its checked-in description. Evidence: `internal/application/item/effect_gift_box.go:96-105` and `docs/database/game_data/TBL_EQUIPT_TEMPLATE.json:70027-70203`.

## 3. Flash wire contract

### 3.1 Open package

The inventory double-click path calls:

```text
useItem(1, -1, packageItemInstanceID)
```

The third argument is the owned item instance ID, not the package template ID or bag SID.

Evidence:

- `docs/client/view/view/comp/Slot.as:720-729,1675-1688`
- `docs/client/object/Player.as:792-831,854-892,1225-1232`
- `docs/client/predef/GamePredef.as:1006-1007`

### 3.2 Open selection panel

The server callback shape is:

```text
itemMultitoSel({
  packItemList: {
    "0": {ti, ii, n, q},
    "1": {ti, ii, n, q}
  },
  packGetNum: 1
}, packageItemInstanceID)
```

Fields:

- `ti`: template table type
- `ii`: reward template ID
- `n`: displayed quantity
- `q`: display quality
- property key: selection index echoed to the server

The panel renders three entries per page. It does not sort entries itself, so the server must preserve deterministic index assignment and must validate the returned index against the pending snapshot.

Evidence: `docs/client/view/view/compDragable/MultiItemPanel.as:491-547` and `docs/client/view/view/compDragable/MultiItemPanel.as:508-525`.

### 3.3 Confirm selection

The client sends:

```text
selMultiItemByIdx(packageItemInstanceID, selectedIdx)
```

Responder semantics:

- `< 0`: keep state unchanged
- `0`: close the panel
- `> 0`: keep the panel open with that many selections remaining

The client has explicit inventory-capacity behavior for `ti=12` Pet, `ti=19` equipment and `ti=29` item. Other award types are not established as safe selectable-panel entries by the client export.

Evidence: `docs/client/view/view/compDragable/MultiItemPanel.as:559-607`.

## 4. Current server behavior

- `internal/application/item/effect_gift_box.go` detects choose-one wording but generally resolves only creature names. Exact packages 6575, 6634 and 3376 are hardcoded.
- `internal/application/item/effect_quest_item.go` checks `awardEffectHandler` before `giftBoxEffectHandler`. Any award rows therefore bypass the selectable panel today.
- `internal/presentation/rtmp/handlers/item/use.go` sends `itemMultitoSel`, stores one pending package per character and implements `selMultiItemByIdx` for Pet/equipment/item.
- Current pending state is in memory. Reopening another package replaces it, and restart/disconnect can invalidate it safely because the package is not consumed until selection completes.
- Reward is currently granted before the package is consumed. This is a pre-existing non-atomic limitation and is not evidence that dashboard configuration is safe for every award type.

## 5. Current dashboard and persistence

- `/dashboard/box-items/[id]` uses `box-item-award-workspace.tsx` to edit `data.data_tbl_item_award`.
- Existing rows support reward type, quantity/range, weight, guaranteed status, quality, equipment prefix and arbitrary `payload`.
- There is no box-level selectable mode, selection count or explicit order.
- The POST endpoint allocates IDs from `max(id)+1` and upserts rows one by one, so a concurrent or partially failed save can produce incomplete configuration.
- The runtime cache returns matching item-award rows from a Go map without sorting.

Relevant files:

- `dashboard/src/app/dashboard/box-items/_components/box-item-award-workspace.tsx`
- `dashboard/src/app/api/box-items/awards/route.ts`
- `dashboard/src/services/box-item.service.ts`
- `dashboard/src/lib/box-item-awards.ts`
- `internal/gamedata/cache.go:GetItemAwardsByItemID`
- `internal/application/item/effect_award.go`

## 6. Recommended data contract

Reuse `data.data_tbl_item_award.payload` with a reserved object:

```json
{
  "selection": {
    "order": 0,
    "getNum": 1
  }
}
```

Rules:

1. An item is selectable only if every configured award row has `payload.selection`.
2. `order` values are unique contiguous integers starting at zero.
3. Every row has the same `getNum`, with `1 <= getNum <= number of choices`.
4. Selectable rows allow only types `12`, `19` and `29` in the initial scope.
5. `guaranteed`, `rate`, `countMin` and `countMax` do not control selection; random quantities are forbidden in selectable mode so the panel quantity equals the granted quantity.
6. Other reward-specific payload keys remain available, but the reserved `selection` object is written by the dashboard rather than raw JSON input.
7. Mixed selection/non-selection rows fail closed instead of falling back to the random picker.

This model avoids duplicating the reward catalog and keeps fixed/random behavior unchanged for all existing rows.

## 7. Risks and deferred mechanics

- Exact orange quality for item 3846 is not uniquely established within the orange 21-25 band. It should be configured explicitly by the operator rather than inferred from the word “cam”.
- Flash rendering/capacity authority only confirms selectable Pet/equipment/item rewards. Currency, EXP, buff, title, skill, medal, mount and Fairy rows remain fixed/random-only.
- Changes require gameserver game-data reload/restart; no hot reload is currently available.
- A fully atomic cross-domain “consume package + create equipment/item/Pet” transaction is larger than the dashboard mode itself. The implementation should preserve current behavior, add strict preflight/revalidation and regression coverage, and record the existing residual risk rather than claim transactional atomicity.
- `useMultiItem` cannot represent interactive selection and must reject selectable packages without consuming them.
