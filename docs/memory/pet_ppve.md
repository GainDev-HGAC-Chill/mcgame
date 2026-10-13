# Pet PVE — Không Gian Điêu Khắc (Carve Space)

Panel/login stage-container fixes deployed17:34:09 (2026-10-12), focused tests/vet/build PASS. Does not certify petPVE combat/replay/carve progression; Flash retest NOT TESTED. [Batch results](../research/2026-10-12_09_PANEL_PROTOCOL_ECONOMY_TRIAGE_RESEARCH.md).

## Empty-stage login consistency (2026-10-12)

In addition to the repaired lazy-panel stage containers, buildPetPVEDataJSON now supplies non-null p.s0 through p.s7 without changing persisted state or awarding progression. Populated stage data is retained; regression checks that shaping does not mutate the input state. No claim that combat/replay or carve advancement is fully implemented. Focused stat-feature/login tests PASS; real Flash retest pending.

Solo pet tower: assemble up to 5 bound pets with per-pet AI command lists, challenge
floor-by-floor (max 150). Clears earn **Văn Chỉ (KP)** to level **Khắc Văn (KhacVan)**
carve-runes that buff pet stats; a **master carve level (mlv)** tracks overall progress.
Vietnamese label array: `Language.PET_PVE_PANEL`.

## Login wire vs. lazy panel — the key finding

`petPVEData` (and `petTalentInfo`, `petStoneBag`) are **NOT read by the client at
login** — each panel lazy-loads via its own RPC. BUT the real server
(`docs/database/sample_data/hs.json`) emits all three **populated from per-character
state** at login. Since the parity north star is hs.json, all three are emitted from
the persisted `character_stat_features` rows via `statfeature.BuildLoginBlobs`
(reshapers `buildPetPVEDataJSON` / `buildPetTalentInfoJSON` / `buildPetStoneBagJSON`,
wired in `auth/login_cdata_blobs.go`). The reshapers operate on the raw JSONB (no
cross-package import), mirroring `buildPRSInfoJSON`/`buildStoneSealInfoJSON`.

- `petPVEData` wire = `{p, mlv, ppveConfig}` (the ppveData half only — kpData is never
  in the login blob). It deliberately does NOT leak kp/freeTime/etc.
- `petTalentInfo` wire = `{tal, inTal, b}` (pet_talent state is already wire-shaped).
- `petStoneBag` wire = sparse Array, element[slot] = `[giid, stackNum, skillId]`
  (mirrors petstone `toArray`); empty bag → `[]`.

## Persisted state (feature_key `pet_pve`)

`internal/application/pet/ppve_service.go` `PPVEState`:

```
ppvefloor, todayFloor, kp, freeTime, goldTime, goldClgTime, awardTime,
gold4awardTimeDaily (int), mlv, lastResetDay, lastDailyDay,
p (map), ppveConfig (map)
```

`p` and `ppveConfig` are stored raw (`map[string]interface{}`) — combat is deferred so
the server is a persist+echo layer for them. `gold4awardTimeDaily` was changed from
string→int (counter). Floor progression stays on the atomic DB function
`player.ppve_challenge_next_floor`, which `||`-merges (shallow) so it preserves
p/mlv/ppveConfig/kp; the Go panel paths full-Upsert the encoded state.

## Wire split (onUpdatePPVEPanel)

Push `onUpdatePPVEPanel(kpData, ppveData, ppveConfig|null, clearFlag)`:
- `kpData` (`PPVEService.KPData`) = `{ppvefloor,kp,freeTime,goldTime,goldClgTime,awardTime,gold4awardTimeDaily}`
- `ppveData` (`PPVEService.PPVEData`) = `{p, mlv, ppveConfig}`  — **mlv = master carve level, NOT char.Level** (the old activity stub used char.Level — fixed)
- arg3 = saved config on `savePPVEConf`, else `null`; arg4 = `clearFlag` (false)

## Panel RPCs

Handlers in `internal/presentation/rtmp/handlers/activity/ppve_panel.go` (push-style,
null responder → return `nil,nil`), registered in `activity/handler.go`:

| RPC | Service method | Behavior |
|-----|----------------|----------|
| `initPPVEPanel` | `InitPanel` | load + daily reset (+persist if changed) → push kpData/ppveData, arg3=null |
| `getPPVERank` | — | push `updatePPRankView([], -1)` (cross-player leaderboard deferred) |
| `savePPVEConf` | `SaveConfig` | store `args[0]` config → push with arg3=config (client shows "Cài đặt thành công") |
| `increaseChallengeTimeByGold` | `IncreaseChallengeTime` | charge tiered gold [10,20,50] by goldTime (clamped), grant +1 freeTime, bump goldTime/goldClgTime |
| `exchangeKP` | `ExchangeKPFree` | consume 1 awardTime → +KP (placeholder magnitude) |
| `exchangeKPByGold` | `ExchangeKPByGold` | 25 gold → 50 KP, bump gold4awardTimeDaily |

`challengeNextFloor` (DB-backed floor advance, combat stubbed) stays in
`activity/ppve.go`; `replayPetPVEFight` delegates to `ReplayPetFight` in the petarena
handler. Insufficient-gold / no-free-exchange paths push `onMidNote` and mutate
nothing.

## Daily reset (two independent markers)

`applyPPVEDailyReset` resets off two keys so a reset lands regardless of which path
runs first on a new day (the DB challenge function only sets `lastResetDay`):
- `lastResetDay` → freeTime=3, todayFloor=-1
- `lastDailyDay` → goldTime=0, goldClgTime=0, awardTime=1, gold4awardTimeDaily=0

In practice `initPPVEPanel` always runs before any challenge (client calls it in
`showPanel`), so the gold counters reset on panel open.

## Security: consume-then-credit

Gold deduction (`char.Gold -= cost` + `chars.Update`) is persisted BEFORE the benefit
(KP / extra challenge) state save. A mid-failure leaves the player charged but
un-credited (safe-for-house), never the reverse. Gold uses `char.Gold` directly (NOT
`DeductCurrency(1,...)` — currency-type 1 is ArenaPoint, an ID collision with
`BasicCurrencyGold`).

## Deferrals

- Combat resolution (challenge/replay outcome) — stubbed; `challengeNextFloor` returns
  result code -1 and advances the floor via DB only.
- `goldClgTime` is display-only ("Lượt chiến mua"); the functional gate is `freeTime`
  (the DB function only knows freeTime), so a purchase bumps freeTime to be usable.

## Resolved (rounds 24–26)

- Combat result (round 25): `ChallengeNextFloor` success now sends `BATTLE_WIN(1)` (was -1);
  server has no combat engine — floor-advance == win, denial stays -1.
- Free-exchange KP magnitude (round 24): `TBL_CARVE_AWARD.freeExchagne[PPVEFloor+1]` (clamp 150),
  50 kept as the unwired/row-missing fallback.
- Cross-player rank (round 26): `getPPVERank` + post-challenge rank args now live via Postgres
  fn `player.get_ppve_rank(p_char_id, p_limit)` (migration `20260615060638_ppve_get_rank.sql`):
  `character_stat_features`(pet_pve) ⋈ `characters`(name,class_id) ⋈ `character_progression`(level),
  filtered `(state->>'ppvefloor')::int > 0`, ordered floor DESC (tiebreak character_id ASC),
  `row_number()-1` = 0-based rank; returns jsonb `{rank:[{cid,name,classId,level,floorNum}…top
  p_limit], myRank}` (cid/classId TEXT; myRank over the full set, -1 if unranked). Go: domain
  `PPVERankRow`/`PPVERankResult`; `StatFeatureRepository.PPVEGetRank`; `PPVERankProvider` iface +
  `SetRankProvider` + `GetRank` (limit `PPVERankLimit=50`); `ChallengeNextFloor` populates
  rankList+myRank best-effort; handler pushes `updatePPRankView(rankList, myRank)`. Only
  `goldClgTime` enforcement remains deferred for this feature.

## quietMethods

None of the 6 panel RPCs are quieted yet — await in-client confirmation before adding
to `quietMethods` (ask first).

Related: [[stat-feature-foundation]].

## 2026-10-12 protocol-only panel repair

- Confirmed `PetPVESystem.as:2411–2431`: `initPPVEPanel(null)` and `getPPVERank(null)` use null Responders; panel state is delivered by a named push, not an RPC result. `updatePPVEPanel` at 3427 takes `(kpData, ppveData, config, clearFlag)` in that order.
- `PetPVESystem.as:3479–3485` displays `PET_PVE_PANEL[13]` ("Lỗi dữ liệu") when the selected `p.sN` is missing. The eight part buttons select indices 0–7; `updateKPPart` at 2338 treats missing rune indices 0–5 as level zero. Thus an empty **part object** is valid, but an empty **outer p object** is not.
- `PPVEService.PPVEData` now supplies non-null empty `s0`–`s7` containers for missing/null/malformed parts. Existing rune maps, master level and config are preserved. This is wire-only normalization: it neither writes these containers to storage nor grants rune levels, KP, floors or pets. Tests cover empty/partial/malformed state and the exact four-argument handler push.
- Login `statfeature.buildPetPVEDataJSON` still emits `p:{}` for empty state; no shared login files were changed in this scoped repair. Its optional consistency patch belongs to the parent (copy `p`, fill `s0`–`s7` with empty maps before JSON marshal). The lazy panel path is fixed independently.
- **Still blocked, not validated gameplay:** carve upgrade/master-upgrade logic and real Pet PVE combat/replay. The old claim that database floor advancement represents a real `BATTLE_WIN` is not combat evidence: `ChallengeNextFloor` currently advances via bookkeeping and generates a replay ID without resolving a battle. This repair does not exercise or extend that behavior. No live player actions, database writes or deployment were performed.
- The decompiled `system/CallBack.as` aborts at the trait-count limit, so the outer named-callback implementation cannot be independently verified from this file; panel method signature and existing server push are verified.
