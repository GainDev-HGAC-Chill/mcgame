# Gameplay RPC, Transport, and Pet Fusion Fixes

Status: Approved by the user's 2026-10-13 execution instruction

## Scope

Fix four reported flows without editing Flash source or testing against live player assets: collection-glove shop routing, blood-bag purchase, timed transport scroll disconnect/consumption, and normal Pet fusion quality/tooltip/randomness.

## Confirmed mechanics

- `ProductPanel.buyGlove()` calls fire-and-forget `clickShop(275)`. ID 275 is NPC `Tiệm Tạp Hóa Ông Mộ`; its game-data `shop_id` is 32. Shop 32 contains gathering gloves 1123–1126. The server must push `onOpenShop(32)` and leave purchase to existing shop RPCs.
- `BloodAddPanel` calls fire-and-forget `buyBloodBag({bt,bn})`. `bt` is 1..4, `bn` is the requested reserve, each reserve caps at 10,000,000, and cost is `ceil(bn/5000)` Gold. The client refresh method is `setBloodBag("|hp|mp|petHp|petMp")`; login field is `cData.ep`.
- Timed transport items have `T > 0` in minutes and are reusable until a persisted expiry. `useTransport(mapId, consumeFlag)` is fire-and-forget. Existing runtime logs prove the old deployed path disconnected immediately after the call; no server panic was logged.
- Normal fusion calls `petJoin(main,sub1,sub2,sub3,sub4)` and expects `{flag,pid,grow}`. Success rates are 100/80/50/30 by current tier; maximum tier cannot fuse. Success must land in exactly the next tier; failure consumes only four materials.
- `onRefreshPetProp` takes two positional arguments `(petID, property)`. Tooltip/stat views read base `property.growRate` plus `property.growRateAdd`, while root `growRate` is the effective value used for color.

## Design and touch points

| File | Change |
|---|---|
| `internal/presentation/rtmp/handlers/shop/*` | Register `clickShop`, resolve NPC through game data, push `onOpenShop` |
| `internal/application/bloodbag/*` | Validate purchase, serialize four reserves, per-character replay/concurrency guard |
| `internal/presentation/rtmp/handlers/shop/*` | Register `buyBloodBag`, push Gold and blood-bag state |
| `internal/domain/statfeature/statfeature.go` | Add blood-bag feature key |
| `internal/application/statfeature/service.go` | Populate login `ep` from persisted state |
| `cmd/gameserver/main.go` | Wire game-data and blood-bag dependencies |
| `internal/application/item/timed_reusable*` | Preserve timed item, never extend active expiry, handle expiration/fallback safely |
| `internal/application/pet/service.go` | Enforce template/tier rules, probabilistic success, exact one-tier result |
| `internal/domain/pet/pet.go` | Correct base/add/effective growth DTO contract |
| `internal/presentation/rtmp/handlers/pet/*` | Refresh root growth and nested Pet property with exact callback signature |

## Non-goals

- No Flash edits, production restart, live purchases, Pet consumption, or player-data mutation.
- Do not redesign all character currency persistence in this change.
- Do not change configured fusion success rates or unrelated HP/MP recovery behavior.
- Do not change item 3722 duration without a separate data-design decision; its label/data mismatch is recorded only.

## Verification

- Unit tests for NPC 275 routing, malformed/valid blood-bag payloads, price boundaries, cap/insufficient Gold/replay, login `ep`, timed activation/reuse/expiry, fusion failure/success/tier boundaries/invalid materials, DTO and callback shape.
- Focused package tests for touched application/domain/handler/cache/postgres packages.
- `go vet` for touched packages and `go build -o bin/gameserver ./cmd/gameserver`.
- Manual Flash smoke test remains required after an authorized rebuild/restart.
