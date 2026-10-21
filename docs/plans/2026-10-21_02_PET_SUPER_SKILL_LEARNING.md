# Pet Super-Skill Learning — Implementation Plan

**Date:** 2026-10-21 · **Status:** Implemented  
**Goal:** Implement all 11 official Pet Super-skill books through the existing `petBook` RPC. Learning requires the matching Sơ/Trung/Cao skills at level 3, atomically replaces those three skills with the exact Super level-1 skill without requiring an empty slot, and prevents lower-tier relearning while the Super family is owned.

Research base:

- `docs/research/2026-10-21_03_PET_SUPER_SKILL_LEARNING_RESEARCH.md` — official notes, Flash wire contract, exact 11-family ID matrix, slot behavior, and atomicity gap.
- `docs/memory/pet-system.md` — current `petBook`, quality, canonical slot, callback, and upgrade contracts.
- `docs/research/2026-10-21_02_PET_SKILL_BOOK_CATALOG_AUDIT.md` — current resolver coverage and fail-closed policy.

## 1. Confirmed mechanics

- Keep the existing RPC: `petBook(petId, bookItemInstanceId)`.
- Keep responder keys `f`, `rf`, and `n`; no new Flash callback or ActionScript change.
- Super books are exact item templates `3216..3226` and require Pet quality `qLevel >= 3`.
- Use the exact mapping in the research document. Do not infer a result from minimum ID, name sorting, or prerequisite level.
- Require all three exact Sơ/Trung/Cao level-3 IDs in canonical `skill1..skill15` slots.
- Result is always the mapped Super level-1 ID.
- No free skill slot is required.
- Replace the lowest-numbered prerequisite slot with Super level 1 and clear the other two prerequisite slots to `0`.
- Owning any level of the matching Super chain blocks learning its Sơ/Trung/Cao books.
- Missing/wrong-level prerequisites and family conflicts are non-consuming learn failures (`f=1,rf=false`).
- Preserve normal `petUpSkill` behavior after learning so Super level 1 can be upgraded to levels 2 and 3 through the catalog chain.

## 2. Implementation units

### Unit A — Exact family catalog and validation

Files:

- `internal/application/pet/skill_books.go`
- new focused catalog/helper file under `internal/application/pet/` if needed

Changes:

1. Add an explicit 11-row Super-learning catalog keyed by book template ID, containing the three level-3 prerequisite IDs, Super level-1 ID, and all Super chain IDs used for lockout.
2. Validate catalog integrity against loaded game data: expected names/family, levels, environments, and level chain. Fail closed if checked-in/runtime catalog drifts.
3. Route only exact books `3216..3226` into the Super-learning branch; all other books retain ordinary first-open-slot behavior.
4. For Super books, validate `qLevel >= 3` and the item `reqClass` against creature `classId` as the Flash Learn-button does. Do not reinterpret `creKind` family metadata as `classIds` eligibility.
5. Extend ordinary family-conflict validation so any owned Super-chain level blocks its matching Sơ/Trung/Cao book.

### Unit B — Atomic Pet + book mutation

Files:

- `internal/domain/pet/repository.go`
- `internal/infrastructure/persistence/postgres/pet_repository.go`
- `internal/infrastructure/cache/cached_pet_repo.go`
- `internal/infrastructure/cache/player_data.go` if a committed-state helper is needed
- `internal/application/pet/skill_books.go`

Changes:

1. Add a narrowly scoped repository contract for skill-book mutation.
2. In PostgreSQL, begin one transaction; lock the owned Pet and book item rows; revalidate ownership, current canonical skill-slot snapshot, and positive book stack.
3. Persist the fully recalculated Pet and decrement/delete one book in that transaction; return the committed remaining stack.
4. Roll back every database change on any error. Do not use best-effort compensating writes for this path.
5. Update active/cold/warm caches only after commit, including deletion or changed stack of the consumed book.
6. Keep a test-only/in-memory implementation path with the same all-or-nothing contract; production must fail closed if atomic persistence is unavailable.

### Unit C — Slot mutation and RTMP callbacks

Files:

- `internal/application/pet/skill_books.go`
- `internal/presentation/rtmp/handlers/pet/fusion.go`

Changes:

1. Locate the three prerequisite slots by exact ID.
2. Build the final state before persistence: lowest prerequisite slot → exact Super level 1; other two → `0`; unrelated and locked slots unchanged.
3. Recompute passive bonuses and Pet stats once from the final skill set.
4. Return all changed slots in the application result.
5. After commit, emit inventory delete/stack callback, then `onUpdatePet` for each changed slot in ascending order, then one `onRefreshPetProp`.
6. Return existing success payload `{f:1,rf:true,n:<remaining>}`.

### Unit D — Regression and catalog invariants

Files:

- `internal/application/pet/book_skill_test.go`
- `internal/application/pet/pet_skill_book_catalog_test.go`
- `internal/presentation/rtmp/handlers/pet/*_test.go`
- `internal/infrastructure/persistence/postgres/*_test.go` and cache tests where appropriate

Tests:

1. Table-driven coverage for all 11 mappings: exact three level-3 prerequisites → exact Super level-1 result.
2. Success with no empty slots; only the three prerequisite slots change.
3. Deterministic replacement uses the lowest prerequisite slot regardless of acquisition order.
4. Reject each missing prerequisite and levels 1/2 without consuming the book or changing Pet state.
5. Reject when any Super level 1/2/3 of the family is already owned.
6. Reject lower-tier book learning while matching Super is owned; unrelated families remain learnable.
7. Preserve quality and item `reqClass` restrictions without using `classIds` as the family gate.
8. Persistence failure, stale-snapshot race, item-delete failure, and item-stack-update failure leave both Pet and book unchanged and emit no success callbacks.
9. Handler callback order and payload: inventory → three sorted skill updates → one stat refresh; responder remains `f/rf/n`.
10. Existing ordinary-book and `petUpSkill` regressions continue to pass.

## 3. Verification

Run without real Pet/items and without restart/deploy:

```text
go test ./internal/application/pet ./internal/infrastructure/persistence/postgres ./internal/infrastructure/cache ./internal/presentation/rtmp/handlers/pet -count=1
go test ./internal/domain/pet ./internal/domain/combat ./internal/application/combat ./internal/presentation/rtmp/handlers/combat -count=1
go vet ./internal/domain/pet ./internal/application/pet ./internal/infrastructure/persistence/postgres ./internal/infrastructure/cache ./internal/presentation/rtmp/handlers/pet
go build -o bin/gameserver.exe ./cmd/gameserver
CGO_ENABLED=0 GOOS=linux go build -o bin/gameserver-linux ./cmd/gameserver
git diff --check
```

## 4. Non-goals and safeguards

- Do not edit decompiled ActionScript.
- Do not alter game-data seeds or add a schema migration.
- Do not use real books, Pets, items, or player rows for testing.
- Do not restart/deploy, commit, or push.
- Do not expand the existing 45 unresolved container/random/special book mappings.
- Do not change the operator override to the legacy “lowest prerequisite level” behavior.

## 5. Documentation after implementation

- Add the approved Super-learning contract and exact mapping to `docs/memory/pet-system.md`.
- Record tests and rollback/atomicity results in `docs/ai-memory/FIX_HISTORY.md`.
- Update this plan to `Implemented` only after all verification passes.

## 6. Implementation result

- Units A–D are implemented with the exact 11-family catalog, three-slot replacement, lower-tier lockout, atomic PostgreSQL/cache mutation, and unchanged `petBook` responder contract.
- Focused Pet/domain/PostgreSQL/cache/Pet-handler tests, broad application combat and combat-handler tests, and targeted vet pass.
- Windows and Linux gameserver builds pass. Repository-wide diff-check passes with only Windows line-ending warnings.
- No ActionScript, schema, seed, runtime deployment, or player data was changed.
