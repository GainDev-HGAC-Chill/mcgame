# Vietnamese dashboard, target levels and creation packages

## Approved scope

The user approved safe bidirectional target levels, allocated-point refund/recalculation on downgrade, and once-only item/pet gifts with rejection before creation for invalid or over-capacity packages. They also requested Vietnamese throughout the dashboard. Existing characters must not receive gifts retrospectively. No Flash source changes, resets, compensation, actual player mutation or live sample grants are part of assistant verification.

## Implementation tracks

- Vietnamese user-facing dashboard text, preserving API names and persisted enum values.
- Canonical admin target-level action and equipped-item downgrade preflight.
- Versioned `public.server_settings.new_character_defaults` snapshot and atomic character/items/pets persistence at creation only.
- Dedicated typed Vietnamese starter editor with authoritative catalog search, future-only confirmation, and strict key validation instead of a raw JSON-only experience.

## Parent verification results

- Preliminary dashboard typecheck: PASS (`corepack pnpm typecheck`). Bare `pnpm` is not on this host's PATH; use `C:/Program Files/nodejs/corepack.cmd`.
- Starter form parser tests: PASS, five cases cover preserved fallback, item/pet fields, malformed/unknown input, unsafe wallet integers, pet overflow and90-entry/1000000-quantity format limits (`node --test tests/new-character-defaults.test.mjs`). API validation additionally checks catalog IDs, mandatory binding, pet class, material color and merged/split stack capacity before writing.
- Integrated touched Go suites, full vet, Windows/Linux server builds and production dashboard builds: PASS. Deployment/browser evidence is recorded below.
- Parent full `go test ./... -count=1` after backend integration: FAIL only in the known four pet-stat tests, one auth daily PM EXP test (14900 versus15400), and two chat-stat tests. Final domain character/admin/cache/item/creation/config/Postgres/handler-item suites PASS; equip and conditional-transition regressions repeated20 times PASS.
- Live character creation, live target-level changes and reconnect: NOT TESTED, owner-driven only.

### Independent review findings

Creation backend review: no concrete blocker in the trusted service call path. Character, gifts and a `creation_package` marker share one pgx transaction using existing SQL functions; no schema change is needed. Unique names prevent duplicate grants after uncertain commit acknowledgment, but retries do not recover the committed result automatically. Snapshot overwrite protection is application-level, not an immutable database constraint; its hash covers the config, not generated entities. Real transaction execution remains NOT TESTED.

Read-only live compose metadata parity: PASS via Supabase CLI2.120.0. Exact70/15/33-argument character/item/pet functions and3-argument marker function match repository calls; security invoker/empty search_path and privileged-only function EXECUTE confirmed. `characters_name_key UNIQUE(name)` exists. All frontend catalog validation columns exist. This is metadata inspection, not actual transaction execution.

Security fence remains trusted loopback/internal deployment only: existing runtime config tables have RLS disabled and anon/authenticated DML grants, so reachable Data API clients can bypass dashboard config validation. No new public exposure or hidden auth/security changes were made. Untrusted exposure is BLOCKED until access-control remediation is explicitly scoped and verified.

Target-level review found bonus-point loss and stale equality; both were corrected and retested. Conditional cache transition preserves registry membership and checks current inventory while locked; no-op/rejection does not dirty state. Uncached/offline requests deliberately return409, rather than overwrite a stale full-character snapshot. A second review found the equipment commit path and handler full-character resource clamp could race with downgrade; both were repaired. Parent read the atomic equip/resource/cache implementation and reran focused suites plus paused-equip/conditional transitions20 times PASS. Final delegated re-review hit usage limits, so it is not claimed complete.

Vietnamese translation delegation also hit usage limits after editing84 files. Parent continued from those edits, translated remaining confirmations/error fallbacks/date locales, integrated both new forms and verified typecheck PASS. Known seeded runtime descriptions are translated in a display dictionary, without modifying database descriptions or operator content. Technical identifiers and backend protocol values remain unchanged. Production builds/deployment and the scoped browser checks below PASS; this is not Flash gameplay certification.

Parent latest full Go suite still fails only the seven known pet/auth/chat tests; touched packages PASS. Full vet and Windows server build PASS. No player/account SQL repairs, runtime config saves, sample grants, actual target-level changes or Flash actions have been performed by assistant.

## Deployment and browser verification

- Production Docker builds for game/dashboard PASS; two nonblocking preexisting lint warnings (creature img, wallet useMemo).
- Game recreated19:20:05; startup19:20:17 loads5530 quests/itemAwards71 and RTMP started, running/restart_count0. Final dashboard-only image recreated19:28:13, ready269ms, running/restart_count0. DB/Redis startup timestamps unchanged.
- Browser PASS: Vietnamese title/navigation and `lang=vi`; starter fields90/999999/999999 and empty supplemental lists; save disabled until confirmation. Item1132/equipment758/pet91 catalog lookups return200. UI pet search finds Hổ Bì Dương.
- Browser PASS: progression tab shows target form, effects/online-only notice and disabled confirm button before input/confirmation; no actual action submitted.
- Live negative API checks PASS: fractional level, unknown field, invalid consumable color and31-slot equipment package each return400 with Vietnamese errors. Compared reserved config before/after: unchanged. No valid config save or player mutation performed.
- Final fresh browser tab PASS: gift form and Vietnamese seeded-config labels rendered, `lang=vi`; player header renders Cấp instead of Level and Vietnamese tabs. An older open config tab retained a loading skeleton after deployment; a fresh tab rendered normally. Reopen the page or hard-refresh stale tabs after upgrading.

## Owner smoke checklist (after verified deployment)

1. Open `/dashboard/game-config`: verify Vietnamese navigation, dedicated creation package form and future-only notice.
2. Search item/equipment/pet names or IDs, choose entries, edit quantity/binding/quality/growth. Draft changes alone must not grant anything.
3. Save an explicitly chosen valid configuration. Create a new test character yourself and inspect wallet/inventory/pets; reconnect and verify nothing is regranted.
4. Check a preexisting character retains its original values and no package is delivered to it.
5. Open a deliberately chosen test character's progression tab: preview an increase then a decrease, confirm each explicitly. Check EXP reset, refunded points and retained assets/history.
6. If worn equipment requires a higher level, the lower-target request must fail without changes; unequip it yourself before retrying.

Do not interpret successful startup or parser tests as verified Flash gameplay. Do not fix unrelated known pet/auth/chat suite failures to hide them.
