# Dialogue-Only Quest Completion — Implementation Plan

**Date**: 2026-10-14 · **Status**: Approved  
**Goal**: Allow the 160 newly audited type-2/type-4 dialogue-only quests to become ready and turn in at their catalog finish NPC, while preserving the existing 67 reviewed quests (including operator-confirmed quest 46) and keeping all ambiguous or contradicted empty definitions fail-closed.

**Implementation result**: CODE_FIXED. The 227-entry registry, policy integration, audit diagnostics, and regressions are implemented. Independent verification removed the audit test's dependency on an ignored `.local` artifact; registry tests now use only checked-in quest/requirement/NPC catalogs. Deployment and Flash smoke testing remain pending separate approval.

Research base:
- `docs/research/2026-10-14_03_DIALOGUE_ONLY_QUEST_AUDIT.md` — exhaustive catalog partition and exact approved-candidate sets.
- `docs/research/2026-10-12_06_ALL_QUEST_REQUIREMENTS_AUDIT_RESEARCH.md` — existing completion authority, canonical objective reconciliation, and fail-closed boundaries; its quest-46 statement is superseded.
- `docs/memory/npc-quest-system.md` — current quest lifecycle, objective mappings, NPC turn-in authority, and callback contract.

## 1. Confirmed mechanics

- Flash has no conversation requirement kind and never infers readiness from an empty requirement list. Server state `103/104` controls the Finish UI.
- Ordinary completion is `finishQuest({i,s,n})`; choose-one completion is `finishQuest({i,s})`. Existing response shapes and callback ordering remain unchanged.
- Positive `finishNpc` is the turn-in target. Current server readiness and final turn-in both require the character to be on the catalog NPC map within 300 units.
- Quest 46 is operator-confirmed dialogue-only and remains authorized despite its crafting-themed name/text.
- The implementation set is exactly 227 IDs: the current 67 type-1 entries plus the 67 type-2 and 93 type-4 IDs listed in the audit. All 160 new entries have no ordinary requirements, `rt=rn=0`, valid positive start/finish NPCs, and complete dialogue metadata in the checked catalog.
- The 711 ambiguous, 77 contradicted, and 669 other empty definitions are not authorized by this change.

## 2. Design

### 2.1 Single source of truth

Replace the ID-only boolean allowlist with a conversation registry in a dedicated `conversation_policy.go`. Each entry records the reviewed catalog fingerprint:

```go
type conversationQuestSpec struct {
	QuestType int
	StartNPC  int
	FinishNPC int
	Name      string
}
```

The registry contains exactly the 227 audited IDs. `ClassifyQuestCompletion` returns `PolicyConversation` only when:

1. no ordinary requirements exist;
2. `rt == 0` and `rn == 0`;
3. ID, type, start NPC, finish NPC, and name match the reviewed fingerprint;
4. start/complete dialogue is non-empty;
5. finish NPC is positive.

Any catalog drift fails closed as `PolicyUnconfigured`. Runtime must not scan Vietnamese text or infer new conversation quests automatically.

### 2.2 Existing lifecycle

Keep `reconcileCatalogObjectives`, `syncSpecialRequirements`, `ValidateQuestTurnIn`, `FinishQuest`, response payloads, reward handling, and NPC radius unchanged. Newly registered quests reuse the existing conversation path: canonical empty objectives, state 104 only at the finish NPC, and a second turn-in location check before rewards.

### 2.3 Audit tooling

Extend `cmd/questaudit` to report:

- reviewed registry count by quest type;
- registry/catalog fingerprint mismatches;
- missing or invalid finish NPC targets;
- overlap with ordinary requirements or nonzero `rt/rn`;
- exact policy partition using the current registry.

The audit tool validates explicit policy data; it does not authorize from text keywords.

### 2.4 Touch points

| Touch point | Change |
|---|---|
| `internal/application/quest/conversation_policy.go` | Add the 227-entry reviewed fingerprint registry and lookup helper. |
| `internal/application/quest/requirement_policy.go` | Replace type-1/boolean allowlist logic with exact fingerprint validation for types 1, 2, and 4. |
| `internal/application/quest/requirement_policy_test.go` | Test exact registry count/type partition, quest 46, representative new type-2/type-4 rows, drift rejection, requirement/currency precedence, and ambiguous rejection. |
| `internal/application/quest/requirement_authority_test.go` | Exercise state 103/104 and completion for quest 46 plus representative newly enabled quests at/away from finish NPC. |
| `cmd/questaudit/main.go` | Add registry/NPC/fingerprint diagnostics and updated partition counts. |
| `docs/memory/npc-quest-system.md` | Record the reviewed dialogue registry, fail-closed drift behavior, counts, and remaining blocked partitions. |
| `docs/ai-memory/WORK_STATE.md`, `FIX_HISTORY.md` | Checkpoint implementation and verification status. |

No migration or player-data change is required.

## 3. Non-goals

- Do not authorize the 711 ambiguous candidates, including class, loop, guild, and unresolved grow-up lifecycles. Their hidden conditions are not proven.
- Do not authorize the 77 contradicted candidates or implement `rt/rn` currency semantics.
- Do not treat every empty quest as complete and do not derive runtime policy from text keywords.
- Do not change rewards, prerequisites, active/completed player state, history, or replay any quest.
- Do not add persisted NPC-click state. The existing server-owned finish-NPC map/proximity gate remains the interaction proof for this pass.
- Do not change start-NPC acceptance authorization, generic NPC click range, loop/class/guild routing, Quest Admin activation, Flash assets, or ActionScript.

## 4. Verification

1. Unit tests:
   - registry has exactly 227 entries: 67 type 1, 67 type 2, 93 type 4;
   - quest 46 remains `PolicyConversation` with its exact catalog fingerprint;
   - representative type-2 and type-4 additions become state 104 only at their finish NPC;
   - wrong name/type/start NPC/finish NPC, added requirement, nonzero `rt/rn`, missing dialogue, unknown empty quest, and representative ambiguous IDs stay blocked;
   - reward and response behavior remains one-shot and unchanged.
2. Commands:
   - `go test ./internal/application/quest ./internal/presentation/rtmp/handlers/quest ./internal/presentation/rtmp/handlers/npc -count=1`
   - `go test ./internal/... -count=1` and compare against the known four Pet plus two chat-stat baseline failures.
   - `go vet ./internal/application/quest ./internal/presentation/rtmp/handlers/quest ./internal/presentation/rtmp/handlers/npc ./cmd/questaudit`
   - `go build -o bin/questaudit.exe ./cmd/questaudit`
   - `go build -o bin/gameserver ./cmd/gameserver`
   - run `bin/questaudit.exe` against the checked live snapshot and verify the exact 227 reviewed count with zero registry drift.
3. Manual Flash smoke test after separately approved deploy/restart:
   1. Use a test character with quest 46 active; verify it is 103 away from NPC 430 and 104 after approaching/opening NPC 430.
   2. Turn in quest 46 once; verify completion text/rewards and no duplicate claim.
   3. Repeat one newly enabled type-2 and one type-4 dialogue quest at the correct finish NPC.
   4. Verify one ambiguous empty quest remains 103 and cannot grant rewards.
4. Update `docs/memory/npc-quest-system.md`; do not modify Flash source or player data.
