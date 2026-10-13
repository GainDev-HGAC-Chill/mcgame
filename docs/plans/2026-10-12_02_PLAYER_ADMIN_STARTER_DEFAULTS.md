# Player target level and new-character defaults

**Date:** 2026-10-12 · **Status:** Approved — user approved downgrade refund/recalculation and once-only item/pet delivery; full Vietnamese dashboard added to scope

**Goal:** expose an absolute target level from player management and durable, editable defaults for future character creation, without changing existing characters or replaying gifts. Character deletion is a separate unresolved lifecycle, not a hard-delete button.

**Implementation result:** deployed locally2026-10-12 (game19:20:17, final dashboard19:28:13). Scoped tests/build/vet/typecheck/browser and negative config validation PASS. Creation/gifts/snapshot reuse existing SQL functions in one transaction, no migration. Target-level changes require current active cached state; offline changes remain fail-closed409 rather than unsafe overwrite. Owner creation/actual target-change/Flash smoke remains NOT TESTED. See [implementation verification](../research/2026-10-12_10_DASHBOARD_VIETNAMESE_STARTER_IMPLEMENTATION.md).

User decisions: support **both raise and lower target level**; creation gifts include **items and pets**, not just scalar wallets; disabled Delete button is on the **Flash character-selection screen**. User subsequently approved the downgrade and once-only-delivery policies below and requested Vietnamese text throughout the dashboard. Deletion remains separate.

Research: `docs/memory/dashboard.md`; `internal/infrastructure/adminhttp/player_action_character.go`; `dashboard/src/app/dashboard/players/[id]/_tabs/progression-tab.tsx`; `internal/application/character/starter_rewards.go`; `docs/research/2026-10-12_07_CHARACTER_DELETION_RESEARCH.md`.

## 1. Confirmed mechanics

- Player list Settings links to player detail. Current Progression has positive Up level delta/Add exp via advance_progression. Absolute level backend exists but lowering does not reverse earned stats/points; exposing it as a safe downgrade is invalid.
- RaiseLevelTo supplies normal stat/point growth and EXP reset. Admin mutations use cache-aware persistence and online onUPP. An HTTP successful response alone does not certify later write-behind durability.
- Creation currently applies level90, Gold999999 and GoldBind999999 before saving a new character. Selection/reconnect does not run starter grants. Existing starter-bag award configuration is different from automatic creation grants.
- Existing Game Config infrastructure uses PostgreSQL KV and notifications. A new typed key must be wired into creation; editing arbitrary KV alone has no effect.
- Flash deletion requires a retained pending character, delTime Unix-seconds final deadline, startDeleteChar with secondary password, restoreChar using authenticated ownership. Supported repository only hard-deletes; no proven grace period or schedule/cancel lifecycle. Do not enable hard-delete as a compatible substitute.
- Dashboard authentication is not enforced at application level; deployment remains loopback/internal only.

## 2. Proposed design

### Approval addendum

User explicitly approved refunding allocated base-stat points on downgrade, recalculating level-derived values while retaining assets/history, and rejecting invalid/over-capacity starter packages before creation. Implement atomic character/items/pets persistence rather than grant-on-login. Full dashboard user-facing text must be Vietnamese; preserve identifiers, API keys and game wire contracts. If equipped items would be invalid at the target, reject downgrade with a clear explanation rather than deleting equipment or silently retaining invalid bonuses. Skills remain owned; existing runtime eligibility rules apply. No existing-character grants or real player actions are authorized for verification.

### Single source of truth

Typed NewCharacterDefaults configuration in the existing config store: initial level, Gold, GoldBind, item/equipment entries and pet entries. Preserve currently approved90/999999/999999 and empty supplemental lists until operator edits the configuration. Immutable validated snapshot read once at creation; no retroactive application and no login grants. Durable config errors must fail creation before granting, not silently invent a package. New config keys/constraints require compose inspection and, if schema changes are actually needed, CLI-generated diff migration.

### Target level

Add Target level action beside delta controls, linked from players list. Raising uses RaiseLevelTo; same-level is a no-op. Approved downgrade policy: reset EXP to the target's starting progress, refund allocated base-stat points, recompute level-derived base values/remaining points and final stats; retain aptitude, currencies, quests/rewards and owned items/pets. Preserve skill ownership; reject downgrade if currently equipped items would become invalid, with an instruction to unequip first. Current direct setter is not an acceptable implementation because it retains higher-level stat growth. UI must preview the effects and require explicit confirmation.

### Creation gifts

User requires editable initial level/gold/bound gold **and selectable item/pet entries**. Package entries must use authoritative catalog IDs, quantities, binding/quality/growth and a versioned immutable configuration snapshot. Approved overflow policy: validate the entire package before character creation; reject invalid/over-capacity packages instead of silently dropping rewards or placing them into an unimplemented mail bag. Persist character plus item/pet grants and a creation-package version/delivery marker atomically. A retry must never regrant an already committed prefix; login/selection must never create gifts. Admin send-item is not a tracked creation grant. Required persistence/schema changes will be designed against the compose database and captured via CLI diff, never hand-written migrations or destructive resets.

### Touch points

| Path | Proposed change |
|---|---|
| `dashboard/src/app/dashboard/players/page.tsx` | Clear route/action to target-level editor; no immediate player mutation on page load |
| `dashboard/src/app/dashboard/players/[id]/_tabs/progression-tab.tsx` | Target level input, bounds, EXP-reset notice, confirm/action state |
| `dashboard/src/app/dashboard/players/[id]/_components/target-level-control.tsx` | Dedicated Vietnamese bidirectional target form with effect preview and explicit confirmation |
| `dashboard/src/hooks/use-player-detail.ts` | Typed action and normal error/result handling |
| `internal/infrastructure/adminhttp/player_action_character.go` | Bidirectional target action, validated bounds, equipment preflight and canonical progression |
| `internal/infrastructure/config/store/types.go` | Typed defaults snapshot/validation |
| `internal/infrastructure/config/store/store.go` | Snapshot refresh and defaults accessor |
| `internal/application/character/starter_rewards.go` | Apply validated snapshot exactly once before initial persist |
| `internal/application/character/service.go` | Inject/read defaults at Create only, preserve Select behavior |
| `cmd/gameserver/main.go` | Inject the existing config provider |
| `dashboard/src/app/dashboard/game-config/page.tsx` | New Character Defaults form, future-only warning |
| `dashboard/src/components/game-config/new-character-defaults.tsx` | Dedicated item/equipment/pet lookup and editable reward lists, not a grant-on-open action |
| `dashboard/src/lib/new-character-defaults.ts` | Strict shared package structure and numeric/binding validation |
| `dashboard/src/lib/server/new-character-defaults.ts` | Server-only authoritative catalog validation for config writes |
| `dashboard/src/app/api/game-config/[group]/route.ts` | Validated allowlisted keys, no arbitrary player grants |

## 3. Non-goals / decisions required

- No existing-character automatic reset/compensation/sample grants, volume reset, or client source changes. Explicitly requested target-level changes occur only after the operator submits a specific player action.
- No unsafe hard-delete button. User must choose/recover deletion grace period and expiry/finalization rules before a separate persisted schedule/cancel plan can be approved.
- No public dashboard exposure or changes to auth/firewall as a hidden part of this feature.
- Full special-training, paid explorer-medal, recycling economics and secret-hunt gameplay need separately verified prices/rewards/state lifecycles. Bug repair does not authorize invented rewards or a success-shaped stub.

## 4. Verification

1. Target level: invalid/fractional/out-of-bounds/same/lower targets, point/stat growth, EXP reset, online/offline cache persistence, competing updates and callback errors.
2. Defaults: current fallback values, typed bounds/overflow, config refresh, failure before creation, multiple new characters, restart/reconnect without regrant, existing character untouched.
3. Build Go using `go build -o bin/gameserver.exe ./cmd/gameserver`; vet touched packages; run focused Go tests. Dashboard: pnpm typecheck/build and component tests where configured.
4. Owner-driven local smoke: save config without live sample grants; operator explicitly creates a new test character; verify chosen defaults survive reconnect. Operator confirms raise/lower targets on an explicitly chosen character. Assistant does not execute these player actions. Browser read-only checks of translated pages/forms are permitted.
5. Update `docs/memory/dashboard.md`, `login-auth-flow.md`, test-results with PASS/FAIL/BLOCKED/NOT TESTED. No full-suite PASS if baseline pet/auth/chat failures persist.
