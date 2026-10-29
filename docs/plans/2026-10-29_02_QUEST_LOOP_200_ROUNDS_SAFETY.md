# Quest Loops and 200-Round Safety — Implementation Plan

**Date**: 2026-10-29 · **Status**: Approved
**Goal**: Prevent destructive or rewardless 200-round progression while hardening the shared loop lifecycle without inventing reward formulas, fees, item identities, pool weights, or boss protocols.

Approval basis: the operator requested comprehensive research followed by implementation and granted full automated-test authorization, then explicitly instructed the pipeline to continue.

Research base:

- `docs/research/2026-10-29_02_QUEST_LOOP_200_ROUNDS_RESEARCH.md` — Flash contract, catalog census, external gameplay corroboration, and current defects.
- `docs/memory/quest-loop-system.md` — current loop implementation and prior boss-loop work.

## 1. Confirmed mechanics

- `takeLoop(lid,npcID)`, `cancelLoop(lid)`, and `loopRepaire(lid)` are fire-and-forget.
- `onFinishLoopQuest` precedes the next round's `onAddChaQuest`.
- Loop 9 is NPC 48, level 70–160, solo, 200 rounds, with a 72-hour cancel cooldown.
- Its pool is 924 `7-9` children with zero static EXP, zero static money, and zero quest-award rows.
- Dynamic EXP, take fee, milestone item/plan IDs, weighting, and 12 special boss encounters are not authoritative.
- Existing states must remain cancelable and repairable; safety gating must not strand retained data.

## 2. Design

### 2.1 Availability authority

Add a single loop-reward-authority predicate in the quest service. Loop 9 returns unavailable until an atomic dynamic-reward command exists. `CanTakeLoop` uses it so NPC `loopInfo` and direct `takeLoop` calls agree. Completion checks the same predicate before objective consumption or any state mutation.

Other loops retain current availability. This scope does not silently change previously deployed gameplay families whose reward authority requires separate research.

### 2.2 Mutation serialization and take validation

Run take, cancel, and repair under the existing per-character quest mutation lock. Change `TakeLoop` to receive the supplied NPC ID and require:

- exact equality with `TBL_QUEST_LOOP.nid`;
- server-side proximity through the existing `atQuestNPC` check.

Grant the child before publishing the active loop state. If state persistence fails, delete only the newly granted progress and return failure. This ordering prevents a visible active loop without its child. No callback names or payload fields change.

### 2.3 Failure propagation

For non-guild completion, return reward, progress-delete, state-update, next-child-save, and deactivate errors instead of logging and continuing. This does not make the legacy multi-repository reward path atomic, but it removes known success-after-failure branches. A future atomic command remains required before loop 9 can be enabled.

### 2.4 Touch points

| Touch point | Change |
|---|---|
| `internal/application/quest/loop_service.go` | Reward-authority gate, locked take/cancel/repair, exact NPC/proximity validation, take compensation |
| `internal/application/quest/loop_completion.go` | Gate unavailable loop before consumption and propagate lifecycle errors |
| `internal/application/quest/loop_service_test.go` | Lock-safe behavior, loop-9 fail-closed tests, NPC validation, compensation and completion no-mutation regressions |
| `internal/presentation/rtmp/handlers/quest/loop.go` | Pass the parsed NPC ID into the service |
| `internal/presentation/rtmp/handlers/quest/*_test.go` | Preserve fire-and-forget failure callback behavior if harness coverage exists |
| `docs/memory/quest-loop-system.md` | Record census, safety gate, and remaining authority blockers |
| `docs/ai-memory/WORK_STATE.md` | Incremental implementation checkpoints |
| `docs/ai-memory/FIX_HISTORY.md` | Completed defect group and verification status |

No schema migration, database mutation, restart, deployment, or Flash client edit is required.

## 3. Non-goals

- Dynamic EXP and take fee: no authoritative formula or rounding evidence.
- Milestone items and feather rewards: categories are known, exact item/plan IDs are not.
- Diary and achievement producers: deferred with the atomic reward command so milestones cannot commit separately from the round.
- Special QIDs `14001..14012`: no authoritative encounter channel or NPC pairing.
- Weighted child selection: loop 9 has no surviving `scriptGet`; uniform eligible selection remains documented approximation.
- Making all legacy non-guild loop rewards atomic: requires separate reward-authority research for each loop family.

## 4. Verification

1. Unit tests prove loop 9 cannot be taken or completed and completion does not consume/delete/advance retained state.
2. Unit tests prove wrong NPC, `-1`, wrong map, and out-of-range position reject before mutation; NPC 48 in range succeeds for an authorized fixture loop.
3. Unit tests inject child-save/state-save/reward/delete/deactivate failures and assert no success path continues past the failing boundary.
4. Run focused quest application and RTMP handler tests.
5. Run `go test ./... -count=1`, `go vet ./...`, `go build -o bin/gameserver.exe ./cmd/gameserver`, and `git diff --check`.
6. Manual smoke after a later deploy: NPC 48 must not advertise loop 9; a direct `takeLoop(9,48)` must push `onTakeLoop{flag:false}`; retained active loop 9 can still cancel; other loop callbacks preserve ordering.
