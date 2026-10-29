# Wedding System — Implementation Plan

**Date**: 2026-10-29 · **Status**: Approved  
**Goal**: Complete the shipped Flash wedding booking flow end to end, fix destructive divorce-card behavior, enforce one active spouse while preserving remarriage history, and safely transport eligible players into tiered halls.

Approval basis: the operator explicitly authorized full testing and requested completion on 2026-10-29 after the scope and critical item-122 defect were presented.

Research base:
- `docs/research/2026-10-29_01_WEDDING_SYSTEM_RESEARCH.md` — wire contract, IDs, version conflicts, server defects, and external sources.
- `docs/memory/marriage-system.md` — existing marriage persistence and request/accept flow.

## 1. Confirmed mechanics

- Exact RPCs are `bookWeddingHall({line,hour,minute,flag})`, `cancelWeddingHall()`, and `enterWeddingHall(cid)`.
- Booking/cancellation refreshes the panel with `onInitBookList(array)`; rows use exact keys `cid,pid,cname,pname,date,line,flag`, with epoch-millisecond `date`.
- `flag=0` is open and `flag=1` requires an invitation for guests. Spouses are identified by `cid`/`pid`.
- Tier is persisted marriage `ring_type`, because booking sends no tier.
- Tier mapping is normal `ring_type=1`, invitation 1691, map 64, fee 2,990,000; advanced `2`, 1692, map 63, fee 3,990,000; precious `3`, 1693, map 65, fee 5,000,000.
- Booking is same-day, at least ten minutes ahead; entry opens ten minutes before and closes sixty minutes after the scheduled time. Cancellation has no refund.
- Item 122 is the divorce card. Its consumption must not commit unless divorce commits.
- No verified RPC authorizes ceremony vows, ring exchange, invitation issuance, decorations, procession, ceremony rewards, or ceremony-specific Achievement events.

## 2. Design

### 2.1 Single source of truth

Application marriage tier configuration owns `ring_type -> fee/invitation/map`. Booking payloads and transport derive from that table; handlers do not duplicate IDs or prices.

### 2.2 Persistence and invariants

- Drop the two historical-row unique constraints. Rewrite `player.create_marriage` to lock both character rows in stable ID order, recheck non-divorced rows across both partner columns, and insert the marriage/audit row atomically. The character locks serialize every overlapping pair without a duplicated membership ledger.
- Add DB functions for atomic divorce-card consumption, booking payment/reservation, cancellation, booking list, and entry lookup. Functions lock relevant marriage/member/item/wallet rows and use schema-qualified relations.
- Add booking line/flag/booked timestamp columns and a unique active line/date slot index.
- Capture schema/function changes through local compose DB and `supabase db diff -f wedding_system` rather than hand-writing a migration.

### 2.3 Application and cache behavior

- Extend marriage domain records/contracts with booking and command result types.
- Add a cached marriage repository wrapper. Booking flushes/reconciles the actor wallet; divorce flushes/reconciles the consumed item. PostgreSQL remains command authority.
- `Service.BookWeddingHall` validates tier, line, flag, server-local time, and builds the full list returned to the handler.
- `Service.EnterWeddingHall` checks booking window/line and invitation possession. The handler invokes a narrow transport interface after authorization.
- Item `UseGoodCard` delegates to atomic divorce and emits corrected user messages. It never pre-consumes item 122.

### 2.4 Handler behavior

- Booking and cancellation always send `onInitBookList`, including the refreshed list after success.
- Entry rejects invalid/closed/wrong-line/invitation-missing requests before transport and reports a user-visible system message.
- New RPCs remain absent from `quietMethods` until Flash smoke confirmation.

### 2.5 Touch points

| Touch point | Change |
|---|---|
| `internal/domain/marriage/repository.go` | Booking/divorce models and repository commands |
| `internal/application/marriage/{service,wedding}.go` | Tier registry, booking/cancel/entry/divorce rules and DTOs |
| `internal/infrastructure/persistence/postgres/marriage_repository.go` | Call schema-qualified marriage functions and map errors |
| `internal/infrastructure/cache/cached_marriage_repo.go` | Player command barrier and committed wallet/item reconciliation |
| `internal/presentation/rtmp/handlers/marriage/{handler,wedding}.go` | Register/handle the three exact RPCs and callbacks |
| `internal/presentation/rtmp/handlers/npc/{handler,npc_click}.go` | NPC 1281 sends the initial `onInitBookList` needed to open/populate the panel |
| `internal/presentation/rtmp/handlers/scene/admin_transport.go` | Add single-player transport so party followers cannot bypass hall authorization |
| `internal/presentation/rtmp/handlers/item/{handler,special}.go` | Inject divorce capability and fix item 122 flow |
| `cmd/gameserver/main.go` | Wire cached repository, item checker, divorce service, and scene transport |
| `supabase/data/wedding_system.sql` | Canonical functions/schema draft applied to compose DB |
| `supabase/migrations/<generated>_wedding_system.sql` | CLI-captured migration |
| Package tests | Contract, gates, cache reconciliation, SQL adapter, and handler tests |

## 3. Non-goals

- Ceremony vows/ring exchange, procession, decorations, bouquet/fireworks, red-bag rewards, and invitation distribution remain deferred because no complete callback/reward authority was found.
- Registration quest fees and exact party/intimacy checks are not moved into proposal acceptance; shipped quests own those mechanics and duplicating them would risk double charging.
- Wedding achievements 132/133 and invitation achievements 348–350 remain fail-closed because booking is not proof that a ceremony or distinct invitations completed.
- No Flash client or static game-data source is edited.

## 4. Verification

1. Unit tests: tier mapping; malformed args; same-day/minimum-lead checks; duplicate spouse/slot; no marriage; insufficient silver; no-refund cancel; divorce no-card/no-marriage/success; entry before/inside/after window; wrong line; spouse/open/invited/uninvited guest; exact DTO keys and epoch milliseconds.
2. Database: rollback smoke for create/divorce/book/cancel, cross-column concurrency, remarriage after divorce, duplicate-slot concurrency, ACL, shadow replay, and clean post-diff.
3. Go: focused package tests, `go test ./... -count=1`, `go vet ./...`, `go build -o bin/gameserver.exe ./cmd/gameserver`, and `git diff --check`.
4. Flash smoke: open booking panel; reserve each tier; verify list/fee; cancel and verify no refund; test open/invitation entry on correct/wrong line and time; use item 122 and verify card plus marriage terminate together; reopen proposal flow after divorce.
5. Update `docs/memory/marriage-system.md`, `docs/ai-memory/FIX_HISTORY.md`, and `docs/ai-memory/WORK_STATE.md`.
