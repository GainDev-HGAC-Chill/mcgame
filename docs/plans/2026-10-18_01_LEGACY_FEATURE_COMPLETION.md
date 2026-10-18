# Legacy Feature Completion Plan

**Date**: 2026-10-18  
**Status**: Safe implementation pass complete — runtime QA and explicitly blocked mechanics remain  
**Goal**: Complete or harden the ten requested legacy systems without editing the Flash client, trusting client-owned gameplay state, inventing missing rewards/formulas, or changing live player data.

Research base: `docs/research/2026-10-18_01_LEGACY_FEATURE_COMPLETENESS_AUDIT.md`.

## 1. Scope decisions

- Operator approved Units A–K. “Đấu Pet kiểu” is confirmed as Thiết lập/Đấu Trường Pet, and “Trang 2 Chân Hồn Pet” is confirmed as the outer Thăng Cấp tab.
- “Đấu Pet kiểu” is provisionally interpreted as Pet lineup/AI configuration and Pet Arena registration. This exact RPC surface already exists, so the unit is validation and wire-parity hardening rather than a replacement implementation.
- “Trang 2 Chân Hồn Pet” is the second outer PRS tab, Thăng Cấp. The alternative second divine-form subpage is also included in regression coverage.
- Trang Viên, Find Back, Pet setup, and PRS are existing features with targeted defects; they will not be rewritten.
- Unknown Card rewards, HMTXLS reward formulas, and PPVE floor enemies stay fail-closed unless implementation-time archaeology finds a checked authoritative source.

## 2. Implementation units

### Unit A — Registration and contract baseline

**Status**: Implemented; focused tests/vet PASS.

1. Add a combined-dispatcher test covering every requested RPC so legacy activity stubs cannot shadow real handlers.
2. Add table-driven request decoding and minimum callback-shape tests from the Flash contracts.
3. Keep every new or promoted RPC out of `quietMethods` until Flash confirmation.

Acceptance:

- Every method resolves to the intended handler after full gameserver registration.
- Existing real handlers cannot silently regress to starter payloads.

### Unit B — Ma Tâm completion

**Status**: Safe catalog-backed subset implemented. Placement now validates catalog type, hole compatibility, ownership count, and concurrent mutations; direct equipped-heart stats are active. Hole/heart/box economy and combination bonuses remain blocked behind an atomic ownership model and complete formula authority.

Likely files: `internal/application/monsterheart/`, `internal/presentation/rtmp/handlers/monsterheart/`, stat aggregation wiring, and focused tests.

1. Model bag entries as owned heart instances rather than accepting unverified IDs in a feature blob.
2. Implement hole-open, box-upgrade, heart-upgrade, and resolve operations from the four checked `CREATUREH_*` tables.
3. Validate every cost and source item server-side; make consume/currency/state writes atomic or compensating and idempotent.
4. Add the equipped-heart stat provider and trigger character-stat refresh after mutations.
5. Preserve exact panel callback names and `{data,bag}` shape.

Acceptance:

- Forged bag/heart/hole/box IDs and level skips fail without consumption.
- Upgrade/resolve/open costs apply once and callbacks reflect persisted state.
- Equipped heart bonuses affect effective character stats exactly once.

### Unit C — Thẻ Bài Ma Thuật session

**Status**: Blocked for gameplay; compatibility safety implemented. The panel now receives an explicit unavailable state and all mutation RPCs resolve as no-ops, preventing malformed active sessions or method-not-found errors while reward/probability authority is missing.

1. Replace only the legacy `initCardPlayPanel` stub with a dedicated application service and handler.
2. Persist daily play count, active five-card session, changed-card mask/count, state, and claimed marker.
3. Implement `startGame`, `changeCard`, and `addPlayTime` with server-generated draws and replay-safe transitions.
4. Implement hand classification from client/official rules.
5. Implement `getCardAward` only when an authoritative checked reward mapping is recovered. Otherwise return an explicit unavailable result and consume/award nothing.

Acceptance:

- Reconnect restores the same active hand and limits.
- A card can only be changed according to server state; retries cannot redraw or claim twice.
- No guessed reward enters inventory or currency.

### Unit D — Auto-complete instance state machine

**Status**: Blocked for gameplay; all Flash RPC names now resolve fail-closed. No sweep session, cost, task progress, attempt, or reward is fabricated without canonical task/instance integration.

1. Introduce a persisted sweep session with task ID, authoritative instance identity, start/deadline, selected count, status, and idempotency key.
2. Verify prior clear, active task, daily selection cap, bag capacity, and available attempts server-side.
3. Derive duration, silver/gold cost, and rewards from canonical task/instance configuration; reject unsupported task types.
4. Implement start, status, cancel, timed finish, and gold instant-finish RPCs.
5. Commit rewards and quest progress once in a transaction; reconnect must resume the same session.

Acceptance:

- Client-supplied reward/count/time cannot increase output.
- Cancellation and completion are mutually exclusive and idempotent.
- Unsupported instances remain unavailable rather than receiving a generic reward.

### Unit E — Trang Viên atomic transfers

**Status**: Blocked — requires a compose-database transaction/function and generated migration; Docker engine is unavailable. No unsafe two-step replacement was made.

1. Move estate-bag-to-inventory transfer behind one repository/database transaction.
2. Apply the same guarantee to steal rewards and any harvest path that crosses estate and normal inventory state.
3. Preserve current Fazenda callbacks and official stock/steal rules.

Acceptance:

- A failure cannot duplicate or lose resources between estate and normal bags.
- Full-bag and retry cases leave both stores unchanged or exactly committed once.

### Unit F — Hồn Khí ownership, effects, and ten-level action

**Status**: Implemented for checked catalog mechanics with compensating economy writes; focused application/handler/rune tests PASS. Runtime Flash verification remains pending.

1. Add persisted decoration ownership/equip/show state for four positions.
2. Implement activate/equip/off/show/hide/show-level and expiry-refresh RPCs.
3. Replace synthetic suit activation with state derived from owned/equipped Hồn Khí and checked suit data.
4. Make hole-upgrade currency plus feature-state atomic; implement ten-times as repeated server-side attempts with one final callback.
5. Add Hồn Khí character-stat/effect projection only for properties defined by checked tables.

Acceptance:

- An unowned or expired Hồn Khí cannot be equipped or shown.
- One and ten-times upgrades spend exact costs and never retain a deduction after failed persistence.
- Effective stats and suit flags match persisted equipped state.

### Unit G — Pet lineup/AI hardening

**Status**: Implemented for the confirmed Thiết lập/Đấu Trường Pet scope; full multi-Pet command execution remains separate from validation/persistence.

1. Preserve existing `getPetConf`, `getPetArenaConf`, `petFightConf`, and `petArenaTicket` routes.
2. Validate 2–5 unique owned Pets, legal positions, maximum command length, action enum, learned skill ownership/level, and target codes.
3. Store separate contexts for farm, HMTXLS, PPVE, and arena instead of always saving under `farm`.
4. Ensure registration and fight simulation read the same persisted lineup snapshot.

Acceptance:

- Foreign Pets, duplicate positions, forged skills, malformed commands, and oversized sequences fail closed.
- Each panel reads back its own configuration without overwriting another mode.

### Unit H — Phần Thưởng Hồi Phục completion

**Status**: Blocked — banner IDs are now verified, but the flow needs persisted refresh count plus atomic inventory/wallet/claim handling. Compose DB is unavailable, so no migration was hand-written.

1. Move inline SQL into a focused repository and schema-qualified database functions following the project DB workflow.
2. Implement `findBackXuanShangByBanner` only with an exact banner item mapping and atomic item-consume/claim transaction.
3. Recheck entitlement capture for all six activities and retain zero entitlement when no verified historical row exists.
4. Keep 70% silver/100% gold, no activity-point recovery, and no VIP EXP recovery.

Acceptance:

- Every claim is backed by a prior-day entitlement and journaled once.
- Banner failure consumes nothing; concurrent claims cannot double-award.

### Unit I — Tu Luyện Ảo Ma lifecycle

**Status**: Blocked for gameplay; full RPC surface is registered fail-closed. Entry/exit/claim attempts refresh the truthful empty read model and never create synthetic room occupancy or awards.

1. Add persisted room sessions with mode, slot, lineup snapshot, enter/exit timestamps, and claim marker.
2. Implement read model, public/advanced entry, exit, reconnect, and occupancy cleanup.
3. Reuse the validated HMTXLS Pet config context from Unit G.
4. Add reward accumulation/claim only if a checked timing and reward source is recovered; otherwise expose the room lifecycle but keep claim explicitly unavailable.

Acceptance:

- A character cannot occupy multiple rooms or claim twice.
- Server time controls elapsed duration; disconnect/reconnect does not reset it.
- No fabricated reward is granted.

### Unit J — Không Gian Điêu Khắc truthful challenge flow

**Status**: Safety portion implemented; challenge now fails closed without consuming an attempt or advancing a floor while the authoritative opponent roster remains unavailable.

1. Remove immediate floor advancement and fabricated `BATTLE_WIN`.
2. Split challenge into attempt reservation, server-owned battle, and idempotent result commit.
3. Reuse validated PPVE lineup/AI configuration and existing battle playback.
4. Advance floor/rank and award Văn Chỉ only after a server-verified win.
5. If no authoritative floor roster is recovered, keep challenge unavailable with a structured diagnostic; panel/config/KP functions remain usable.
6. Apply carve/master bonuses to Pet effective stats only from checked `TBL_CARVE*` data.

Acceptance:

- Opening/clicking challenge can never grant a free floor.
- Loss/disconnect/retry cannot become a win or duplicate an attempt/reward.
- A verified win updates floor, rank, KP, and Pet stats consistently.

### Unit K — Chân Hồn Pet page 2 integrity

**Status**: Implemented for the operator-confirmed outer Thăng Cấp tab. The alternative special-form subpage was not changed.

1. Add exact outer-tab regression proving `addSoulTreeLvl` advances the checked next tree row and returns full PRS state.
2. Make tree currency deduction and PRS save atomic.
3. Apply compensation to paid collect/chip exchange paths where deductions can survive a state-save failure.
4. Do not change `activePRSShowSpe`; the operator confirmed that “page 2” means the outer Thăng Cấp tab.

Acceptance:

- Second-tab progression works after reconnect and charges once.
- Missing consumer, insufficient item/currency, persistence failure, and duplicate request consume nothing.
- The outer Thăng Cấp tab has focused service and wire tests.

## 3. Ordered rollout

1. Unit A first.
2. Harden existing systems: E, G, H, K.
3. Complete catalog-backed systems: B and F.
4. Add new persisted systems: C, D, I.
5. PPVE Unit J last because it depends on validated Pet configuration, combat integration, and an authoritative floor roster.

Each meaningful unit must update `docs/ai-memory/WORK_STATE.md`; each completed defect group must update `FIX_HISTORY.md` and the relevant feature memory.

## 4. Verification

For every unit:

1. Focused application, repository, handler, wire-contract, concurrency, rollback, and idempotency tests.
2. Existing farm, Pet Arena, item, stat-feature, combat, PPVE, and PRS regressions as applicable.
3. Targeted `go vet` for changed packages.
4. `go build -o bin/gameserver.exe ./cmd/gameserver`.
5. `git diff --check` and a broad `go test ./internal/... -count=1`, compared with recorded unrelated baselines.

Database work, if approved:

- Write a checkpoint before starting the compose stack or applying schema changes.
- Use only the project compose database and sanctioned Supabase workflow.
- Capture generated migrations through `supabase db diff`; never hand-write them and never alter live player rows.

Runtime tests remain separate and require explicit deploy/restart permission. They must use test-only characters/items and must not use Quest 166, Pet Nấm Lục, or real player inventory.

## 5. Explicitly blocked mechanics

- Card reward/probability mapping.
- HMTXLS timing, costs, accumulation rate, and reward formula.
- PPVE floor-to-enemy roster.
- Any Hồn Khí effect not represented by checked static data/client constants.

Implementation-time research may unblock these only with a checked repository source, historical wire sample, or explicit operator rule. Website prose alone is insufficient for exact economy values.

## 6. Approval boundary

Approval authorizes code and test work for Units A–K, plus local compose schema migrations required for new persisted state, under the checkpoint rules above. It does **not** authorize guessed blocked mechanics, deployment/restart, live player-data mutation, use of real items/players, ActionScript edits, commit, or push.

## 7. Completion summary

- Implemented/hardened: A, F, G, K and the safety portion of J.
- Safe partial: B direct ownership/type/stat path; economy/combination mechanics remain blocked.
- Compatibility-only fail-closed: C, D, I.
- Database-blocked: E and H because Docker/compose was unavailable; no migration was hand-written.
- Runtime QA is not authorized yet. No deployment, restart, ActionScript edit, real-player test, commit, or push was performed.
