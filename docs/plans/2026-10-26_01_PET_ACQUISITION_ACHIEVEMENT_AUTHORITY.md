# Pet Acquisition Achievement Authority — Implementation Plan

**Date**: 2026-10-26 · **Status**: Code complete — migration pending operator approval  
**Goal**: Make successful Pet capture and Wish Tree Pet rolls durable, idempotent achievement authorities for requirements 83 and 106–109. Prepare the future Mystery Merchant fact contract for requirements 102–105 without fabricating a merchant producer or activating those requirements.

Research base:

- `docs/research/2026-10-26_01_PET_ACQUISITION_ACHIEVEMENT_AUTHORITY_RESEARCH.md` — verified Flash wire behavior, catalog IDs, persistence boundaries, color encoding, and merchant limitations.
- `docs/memory/achievement-system.md` — existing snapshot, claim, callback, registry, and durable-fact architecture.
- `docs/plans/2026-10-19_02_ACHIEVEMENT_SYSTEM_REPAIR.md` — approved fail-closed authority policy and Wave 3 outbox design.
- [Vietnamese Wish Tree guide](http://vuaphapthuat.go.vn/danh-muc-bai-viet/cay-uoc-nguyen/206.html) — gameplay corroboration only, not a wire or schema source.

## 1. Binding mechanics

- Capture uses `battleUpdateCmd` action `-70`. Requirement 83 is eligible only after the captured Pet row and its fact commit together.
- Wish Tree uses `lottoByClient({type,num})`; a successful draw deposits results into the Lotto holding bag and returns truthy success.
- A Wish Pet is exactly `TBL_PLAN.ti=12`. Normalize its `q` to grow rate before applying `domain/pet.ColorByGrowRate`.
- Requirement 106 receives every committed Wish Pet roll; 107/108/109 receive blue/purple/orange rolls respectively.
- Achievement callbacks are server pushes after durable processing. Callback success is not authority.
- Requirements 102–105 stay unsupported until a real Mystery Merchant purchase transaction exists.

## 2. Implementation steps

### Step A — Shared durable fact contract

1. Add fact kinds and strictly validated payloads for captured Pets and Wish Pet draws under `internal/domain/achievement` and `internal/application/achievement`.
2. Add a reusable PostgreSQL append-fact function to `supabase/data/achievement_facts.sql` and a normalized migration. It must be idempotent by `fact_id` and usable inside caller-owned transactions.
3. Define the future Mystery Merchant payload/identity contract, but do not consume it, emit it, or activate requirements 102–105.
4. Add consumer mappings:
   - captured Pet → requirement 83 / achievement 2016 / target 1;
   - Wish Pet → requirement 106 plus exactly one color-specific requirement when blue, purple, or orange.
5. Reject malformed identities, invalid grow-rate encoding, mismatched color tiers, and unsupported fact kinds without marking them processed.

### Step B — Atomic capture producer

1. Add a capture-specific Pet persistence method; do not make generic Pet contracts count as captures.
2. In the PostgreSQL Pet repository, create the Pet and append `pet-captured:v1:<petID>` in one transaction.
3. Ensure the cache receives the new Pet only after the transaction commits.
4. Keep battle success false if either Pet creation or fact insertion fails.
5. After successful round processing, consume pending facts and send the existing `onUpdateAchieve` deltas. Login recovery remains the crash fallback.

Primary touch points:

- `internal/application/combat/catch.go`
- `internal/application/pet/contract.go`
- `internal/domain/pet/repository.go`
- `internal/infrastructure/persistence/postgres/pet_repository.go`
- `internal/infrastructure/cache/cached_pet_repo.go`
- `internal/presentation/rtmp/handlers/combat/commands.go`

### Step C — Atomic Wish Tree draw producer

1. Generate one operation UUID per server-side draw request and retain each rolled plan ID and ordinal.
2. Build facts only for `ti=12`, normalizing `q` and deriving color with the canonical Pet helper.
3. Replace the split Gold/state writes with one PostgreSQL transaction that locks the relevant wallet/state rows, validates sufficient Gold/current state, updates Gold and free quota/holding bag, and appends all Pet facts.
4. Update active in-memory Gold only after commit. On any failure, commit none of Gold, state, or facts.
5. Preserve the exact Flash response and `updateLottoBag` payload. Process pending facts and send achievement callbacks only after the draw transaction commits.

Primary touch points:

- `internal/application/lotto/draw.go`
- `internal/application/lotto/service.go`
- `internal/application/lotto/state.go`
- `internal/domain/statfeature/statfeature.go`
- `internal/infrastructure/persistence/postgres/stat_feature_repository.go`
- `internal/presentation/rtmp/handlers/lotto/handler.go`
- `cmd/gameserver/main.go`

Pet claim repair (`ti=12` → Pet creation and `onAddPet`) is explicitly excluded from this slice because the requested authority occurs at draw time and claim needs its own atomic grant/removal design.

### Step D — Registry and documentation

1. Move only requirements 83 and 106–109 from unsupported to event authority.
2. Keep 102–105 unsupported.
3. Update census assertions from `472 S + 74 E + 151 U` to `472 S + 79 E + 146 U`.
4. Update achievement memory, audit coverage, work state, and fix history after verification.

## 3. Verification

### Unit and integration tests

- Capture fact identity uses the persisted Pet ID and cannot be produced by generic Pet contract paths.
- Pet insert rolls back when fact insertion fails; no fact exists when Pet creation fails.
- Reprocessing the same capture fact completes requirement 83 once.
- Multi-draw emits one stable fact per Pet result and none for non-Pet results.
- Wish quality examples map correctly: `12→green`, `15→blue`, `19→purple`, `23→orange`.
- Every Wish Pet updates 106; only blue/purple/orange update 107/108/109.
- Wallet debit, free quota, Lotto bag, and all Wish facts roll back together on injected failure.
- Duplicate fact processing and login recovery do not duplicate progress.
- Callback tests assert real requirement and completion deltas after commit.
- Registry/census tests prove 83 and 106–109 are event-backed while 102–105 remain fail-closed.

### Commands

- Focused Go tests for achievement, Pet, combat, Lotto, PostgreSQL repository, RTMP handlers, and gameserver wiring.
- `go vet ./...`
- Windows server build only as `go build -o bin/gameserver.exe ./cmd/gameserver`.
- `go test ./...`, with any pre-existing unrelated baseline failures recorded separately.
- Migration lint, reset/shadow verification, rollback/idempotency checks, and `git diff --check`.

## 4. Safety and rollout

- Do not backfill old captures or draws; no historical source can prove them exactly.
- Do not auto-claim achievements, grant Pets, or mutate players for testing.
- Do not apply the migration, restart, deploy, or run runtime smoke without a separate operator checkpoint and permission.
- Before any database apply, record a checkpoint and take the required schema backup through the compose-backed Supabase workflow.

## 5. Acceptance criteria

1. A captured Pet and its capture fact are atomic; successful processing completes requirement 83 exactly once.
2. A committed Wish Tree Pet result and its fact are atomic; requirements 106–109 complete according to canonical color exactly once.
3. Crashes or callback failures leave pending durable facts recoverable at login, not missing or duplicated achievement progress.
4. Requirements 102–105 remain unsupported, while their future transaction/fact contract is documented and reusable.
5. Existing Flash RPC names, arguments, Lotto bag payload, and `onAddPet`/`onUpdateAchieve` contracts remain unchanged.
