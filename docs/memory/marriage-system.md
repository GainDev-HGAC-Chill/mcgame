# Marriage System

## Wedding completion (2026-10-29)

- Migration `20261028231846_wedding_system.sql` removes unconditional partner-column uniques, adds `wedding_line`, `wedding_flag`, and `wedding_booked_at`, and adds the active line/date index. `player.create_marriage` now locks both characters in stable order and rechecks non-divorced rows across both columns, preventing cross-column double marriage while allowing remarriage history.
- Item 122 is the divorce card. `useGoodCard` now calls `player.divorce_with_card`: card consumption, booking clear, `divorced_at`, and audit append commit atomically. `CachedMarriageRepository` flushes before the command and reconciles the committed stack count.
- Exact Flash RPCs are implemented: `bookWeddingHall({line,hour,minute,flag})`, `cancelWeddingHall()`, and `enterWeddingHall(cid)`. Booking/cancellation sends the complete `onInitBookList` array with `cid,pid,cname,pname,date,line,flag`; `date` is epoch milliseconds. NPC 1281 sends the initial list.
- Tier authority is centralized: normal `(ring1, invitation1691, map64, 2,990,000 silver)`, advanced `(2,1692,63,3,990,000)`, precious `(3,1693,65,5,000,000)`. Booking is same-day and at least ten minutes ahead; overlapping 60-minute bookings on one line serialize through an advisory lock. Cancellation does not refund.
- Hall entry is allowed from ten minutes before through sixty minutes after start. It requires the booked channel; spouses bypass invitation checks, open weddings admit everyone, and invitation-only guests must carry the tier invitation. Scene transport is single-player so party follow cannot bypass authorization.
- Ceremony vows/ring exchange, procession, decoration, invitation issuance, red-bag/reward flows, and ceremony-specific Achievement producers remain fail-closed because their complete wire/reward authority is unavailable.

## Current scope (2026-05-05)

The marriage flow is more shipped than `docs/details.md` initially indicated. Concrete state:

- ✅ `player.marriages` persists partner, tier, intimacy, ceremony booking, marriage, and divorce history. Active uniqueness is enforced transactionally across both partner columns; historical rows no longer block remarriage.
- ✅ Application service `internal/application/marriage/service.go` has full panel + request/respond flow:
  - `InitPanel` returns `seekingList` + `charMarriageList`.
  - `UpdateSeeking` writes a Redis seeking entry.
  - `CreateRequest` writes a Redis request entry; rejects if already married.
  - `RespondRequest(response=1)` calls `MarriageRepository.Create` which persists the marriage row.
  - `RespondRequest(response=2)` flags the request rejected.
  - `GetCoupleRank` paginates the couple ranking.
- ✅ 6 RPCs registered: `initMarriage`, `marriageSeeking`, `marriageRequest`, `marriageReqFeedback`, `cancelMarriageSekInfo`, `getCoupleRank`.

## Hardening landed in this session

Migration `supabase/migrations/20260505082932_marriage_audit_log_and_lock.sql`:

- New table `player.marriage_audit_log` with columns `id`, `marriage_id` (FK CASCADE), `event` (CHECK in `{married, divorced}`), `actor_id` (NULL for player-driven; future GM admin id), `partner1_id`, `partner2_id`, `payload jsonb`, `created_at`. Indexes on `marriage_id`, `partner1_id`, `partner2_id`.
- `player.create_marriage(...)` rewritten from a 3-line `LANGUAGE sql` function to `LANGUAGE plpgsql`:
  - Locks both partner rows with `PERFORM 1 FROM player.characters WHERE id IN (p_partner1_id, p_partner2_id) ORDER BY characters.id FOR UPDATE`. The `ORDER BY characters.id` is load-bearing — without it Postgres uses scan order, which permits a deadlock when `create_marriage(A,B)` and `create_marriage(B,A)` race and each transaction grabs one row before the other (fixed in follow-up migration `20260505104856_marriage_lock_order_by_id.sql`).
  - Inserts the marriage row.
  - Inserts an `event='married'` audit row in the same transaction with the partner pair and a payload of `{ring_type, intimacy}`.

The Go side is unchanged: the function signature `(bigint, bigint, integer, integer, timestamp) → (id, married_at)` is preserved.

## Historical gaps resolved or deferred

The original plan called for "transactional wedding flow" with divorce + wedding scheduling. After auditing the Flash client, those Flash-side RPCs do not exist as we expected:

- **Divorce is item-driven, not a marriage-panel RPC.** `Player.as` sends `useGoodCard()` for item 122; the item handler owns the atomic divorce command.
- **Wedding scheduling is implemented as its own subsystem.** The three exact booking/cancel/entry RPCs are registered and remain logged pending Flash smoke confirmation.
- **No `partner_id` column on `player.characters`.** `MarriageRepository.FindByPartner` works via the marriages table, so this denormalisation isn't load-bearing yet. Add it later if a marriage indicator is needed in `cData` payloads.
- **No `onUPP` / marriage-state push** to either partner on accept. The Flash side appears to refresh the panel via `initMarriage` rather than waiting for a server-pushed callback, so this is acceptable today; revisit if you observe missed UI updates.

## File pointers

- `supabase/migrations/20260505082932_marriage_audit_log_and_lock.sql`
- `internal/application/marriage/service.go`
- `internal/domain/marriage/repository.go`
- `internal/infrastructure/persistence/postgres/marriage_repository.go`
- `internal/presentation/rtmp/handlers/marriage/{handler.go, panel.go, requests.go}`

## Follow-up tickets

- Optional: add a `partner_id` column on `player.characters` if a future feature needs an O(1) marriage indicator on the character row.
- Capture original-server ceremony callbacks/rewards before implementing vows, ring exchange, invitation distribution, procession, decorations, or tier-specific wedding achievements.
