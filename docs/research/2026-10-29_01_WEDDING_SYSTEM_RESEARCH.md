# Wedding System Research

**Date**: 2026-10-29  
**Scope**: Flash wire contract, current server/database state, shipped game data, and external gameplay references.

## Sources

- Flash client: `docs/client/view/view/compDragable/WeddingBookPanel.as`, `WeddingBookPanel_inlineComponent1.as`, `MarriageManagerPanel.as`, `MarriageSeekingPanel.as`, `MarriageFeedHBox.as`, `docs/client/object/Player.as`, `docs/client/config/{ItemConfig,Language}.as`, and `docs/client/predef/GamePredef.as`.
- Runtime-table references: `docs/database/details/TBL_{MARRIAGE,COUPLE,WEDDING_BOOK}.md`.
- Shipped data: `supabase/seeds/data_tbl_{item_template,map,npc,quest,quest_award,quest_pre,shop_slot}.sql`.
- Server: `internal/application/marriage`, `internal/domain/marriage`, PostgreSQL/Redis marriage repositories, marriage RTMP handlers, item special handlers, and marriage migrations.
- Chinese original-version article: http://web.4399.com/mlxt/zx/40868.html
- Vietnamese article: http://vuaphapthuat.go.vn/bai-viet/thanh-duong-hon-le/4050.html
- Current Vietnamese gameplay channel: https://www.youtube.com/@Magiccampusneo — the indexed video catalog/search available on 2026-10-29 contained no wedding-specific video suitable for contract or reward calibration.

External articles are gameplay context only. They are not RPC or schema authority.

## Confirmed Flash contract

### Existing marriage panel

The server already exposes `initMarriage`, `marriageSeeking`, `marriageRequest`, `marriageReqFeedback`, `cancelMarriageSekInfo`, and `getCoupleRank`. The client directly calls those exact names.

### Missing wedding RPCs

| RPC | Input | Client-visible completion |
|---|---|---|
| `bookWeddingHall` | one object `{line,hour,minute,flag}` | push the complete booking array through `onInitBookList(array)` |
| `cancelWeddingHall` | no arguments | push the complete booking array through `onInitBookList(array)` |
| `enterWeddingHall` | groom character ID `cid` | transport the caller into the booked hall |

Each booking row consumed by `onInitBookList` has `cid`, `pid`, `cname`, `pname`, `date`, `line`, and `flag`. `date` is epoch milliseconds. The client derives `couples`, formatted `time`, and `lineName` itself. It treats a row as the caller's booking when `cid == player.id || pid == player.id`.

`flag=0` is open entry and `flag=1` is invitation-only. When no booking belongs to the caller, the client defaults the controls to server-local now plus 15 minutes. The UI states that displayed time is server time.

The booking confirmation text is current-version authority for hall fees: 2,990,000 silver for normal, 3,990,000 for advanced, and 5,000,000 for precious. `bookWeddingHall` does not transmit a tier; tier must come from persisted marriage state.

### Item and map pairing

| Tier | Ring | Invitation | Hall map | Booking fee |
|---|---:|---:|---:|---:|
| Normal | 1707 | 1691 | 64 | 2,990,000 silver |
| Advanced | 1708 | 1692 | 63 | 3,990,000 silver |
| Precious | 1709 | 1693 | 65 | 5,000,000 silver |

Item 122 is `ITEM_GOOD_CARD` in the client branch that calls `useGoodCard`; shipped catalog/gameplay identifies it as the divorce card. The current server incorrectly consumes it and reports a lucky-card message without ending the marriage.

## Gameplay cross-check

The 4399 article describes level 30+, a two-person party, opposite-sex/unmarried eligibility, intimacy above 5,000, leader-paid registration, same-day booking, preparation, decoration, invitations, procession, vows, and ring exchange.

The Vietnamese article confirms three registration tiers (9/999/1999 gold), leader payment, same-day booking, at least ten minutes in advance, a 60-minute hall allocation, entry ten minutes before start, invitation or open entry, and no refund on cancellation. Its old single 5,000,000-silver booking statement conflicts with the current Flash tier-specific fee text, so the shipped client wins for this server version.

Quest data corroborates registration quests 3213/3214/3215 and ceremony quests 3216/3217/3218, with tiered gift bags and rings. The exact ceremony quest lifecycle is outside the three missing RPCs and is not reconstructed from website prose.

## Current server/database defects

- `player.marriages` has separate unconditional unique constraints on `partner1_id` and `partner2_id`. They do not stop the same character appearing once in each column, and they permanently block remarriage after divorce.
- `player.create_marriage` locks character rows but does not recheck active-marriage ownership after acquiring those locks.
- Proposal acceptance creates a tier-1 marriage without authoritative registration/party/economy checks. Existing quest registration remains the only shipped-data authority for those costs.
- `wedding_date` and `wedding_venue` exist, but line and entry policy do not; no booking RPC is registered.
- There is no atomic divorce command. `useGoodCard` consumes item 122 through the write-behind item path and never updates marriage state.
- Achievement RID 131 currently treats a tier-1 marriage audit event as a normal wedding. There is no independently verified ceremony-completion producer for tier-specific wedding achievements; those must not be invented in booking handlers.

## Implementation conclusions

1. Lock both character rows in stable ID order and recheck all non-divorced marriage rows across both partner columns before creation. This enforces one active spouse while allowing historical remarriage without a duplicated membership ledger.
2. Make divorce-card consumption, active marriage termination, booking clear, and audit append one PostgreSQL transaction, then reconcile the item cache through the existing player mutation barrier.
3. Persist booking on the active marriage with line, date, venue tier, and flag. Charge the current Flash tier fee atomically with booking and never refund cancellation.
4. Allow only same-server-day bookings at least ten minutes ahead. A line/time slot is unique. Entry is valid from ten minutes before start through sixty minutes after start, on the booked line; invitation-only guests must carry the tier invitation. Spouses may always enter.
5. NPC 1281 (`Quản Lý Lễ Đường`) must send the initial `onInitBookList`; the panel has no separate init RPC. Transport uses shipped map IDs and catalog safe points and must not carry party followers past per-character entry authorization.
6. No ceremony reward, vow state machine, procession, decoration, invitation distribution, or achievement event is fabricated without wire/reward authority.
