# Full quest requirement audit and authoritative completion repair

## Scope and safety

User: all quests appear unconstrained/auto-completing; inspect the entire catalog and fix. This task audits all5530 unique live templates, hardens acceptance/completion/progress/UI, and refuses unsupported completion rather than manufacturing conditions. It does **not** certify that every quest's gameplay is implemented.

No assistant quest acceptance/completion, inventory/pet use, reward grant, reset, history rollback, player SQL update, schema/seed change, client asset edit, commit or push. Preserve Gain, GainTest and other player progression/rewards. The new server naturally re-evaluates active requirements on user reads/events; completed history is not repaired/replayed.

## Exhaustive catalog artifacts

- Read-only query: `supabase/data/audit_quest_requirements.sql`.
- Compose CLI full snapshot: `.local/research/quest-catalog-audit-20261012.json`.
- Reproducible Go classifier: `go build -o bin/questaudit.exe ./cmd/questaudit`, then `bin/questaudit.exe`.
- Per-ID policy report: `.local/research/quest-completion-policy-20261012.json` (one record for every5530 quest).
- PowerShell capture must set Console.OutputEncoding/OutputEncoding to UTF8; an initial OEM-decoded capture corrupted Vietnamese names and was discarded/requeried. Final counts below are from the corrected UTF8 snapshot.

| Live type | Total | Ordinary requirement rows present | No ordinary rows |
|---|---:|---:|---:|
|1 tutorial|140|54|86|
|2 main|790|400|390|
|3 class|285|267|18|
|4 world|775|395|380|
|6 activity|810|459|351|
|7 loop|1395|1134|261|
|9 callboard|712|712|0|
|10 grow-up|211|74|137|
|11 guild|320|259|61|
|12 lucky|92|92|0|
|**Total**|**5530**|**3846**|**1684**|

Final static policy partition:

| Policy | Count | Status |
|---|---:|---|
|Ordinary catalog item/kill/pet|3682|Authoritative evaluator; real ownership/quality/kill events required|
|Reviewed tutorial conversation/knowledge delivery|66|Actual finish NPC map and proximity required|
|Character level description-backed chain|14|Actual level threshold, not merely minLevel to accept|
|Pet level chain|14|Highest real owned pet level|
|Owned pet count chain|23|Distinct real owned pet IDs|
|Friendship chain|39|Distinct real friends plus all intimacy>=50|
|Unclassified empty definitions|1493|BLOCKED/NOT IMPLEMENTED; no implicit completion|
|Potential currency requirement rt/rn|195|BLOCKED/NOT IMPLEMENTED; no invented debit or reward|
|Missing target definitions|4|BLOCKED until source target is recovered|

3838 have a described/catalog evaluator;1692 are explicitly blocked. Counts are **not gameplay PASS** and do not verify every NPC route, activity subsystem, or reward mode.

Missing targets: quest3983 creature666; quest5005 creature838; quest50170 item3831; quest50171 item3832. No guessed replacements. All attached ordinary rows use supported(kind,type) combinations. There are116 requirement rows not attached to the live5530 catalog; no orphan deletion/reassignment. One live duplicate kill-target definition exists in quest1665: creature337 appears twice; require two distinct kill events, do not rewrite it to creature338 without evidence.

## Confirmed defects and repairs

1. **Client objective authority**: takeBuildQuest accepted an optional empty/fabricated objective list, including current/required. It now ignores that data; service acceptance independently rebuilds from catalog. Actual one-object guild RPC `{id,nid}` is accepted alongside legacy argument shape. Unknown dependencies/catalog do not silently authorize acceptance.
2. **Invented callboard conditions**: removed reachable quest-ID-range-based default objectives (monster/item1 fallback). Callboard uses actual GetQuestRequire through canonical service acceptance.
3. **Persisted omissions/forgeries**: active objectives are reconciled to ordinary catalog definitions for every read/event/completion branch. Preserve only exact matching event progress, clamp to required; missing/truncated/mismatched definitions cannot replace requirements. Matching historical kills are preserved because their provenance cannot be retroactively proven.
4. **Empty-objective authorization**: runtime-only CompletionBlocked is now based on authoritative policy for **all types**, not nonempty stored data or type10 alone. Known special definitions reconcile to canonical empty objective lists, so old synthetic collect/pet objectives neither strand them nor consume unrelated assets. All remaining empty unknowns stay state103.
5. **Real custom state**: level, pet level/count and friendship chains inspect current server-owned data. Description-backed local rules are explicit; original scripts unavailable. Unknown scripts/actions are not inferred solely from their quest type or display metadata.
6. **Missing dependencies/targets**: required item/pet/guild readers, valid class restriction/prerequisite kind/mode, and referenced target templates are verified. Missing target templates cannot fall back to generic renamed-pet eligibility. Type11 requires real guild membership. PreQuestType0 retains the client's intentional no-prior-quest mode;1/2 retain AND/OR.
7. **Turn-in authority**: FinishQuest checks actual catalog finish NPC plus server character map/coordinates, not client `n`. Local interaction radius is300 units (explicit server policy, not a recovered original constant). Omitted `n` in valid choose-one UI requests does not bypass the location check. Nonpositive finishNpc system turn-ins remain compatible. Reviewed conversation UI becomes ready only at its actual NPC.
8. **Invalid/expired objectives**: Required<=0 is never complete; expiry participates in CanComplete and special verification. PostgreSQL malformed/type-invalid objective JSON now returns identity-qualified errors instead of a valid empty slice. Legitimate[]/null are subsequently governed by policy, never automatically authorized.
9. **Stale read writeback**: GetActiveQuests/EnrichQuestDTO synchronization is non-persisting. An old read cannot overwrite completed status or history. Active item/pet ownership is calculated for UI/reward gating each time; event writes/completion save authoritative snapshots. Per-character striped mutation locks serialize acceptance, progress events and completion within this local monolith.
10. **Kill-credit authorization/idempotency**: battle cleanup requires an actual player participant; per-character ended-battle claims distinguish inflight/completed, do not ACK cleanup while credit is inflight, and release on errors. Each real dead enemy uses a server-derived battleID/enemyID key. Objective.killCreditEvents is persisted inside existing objective JSON, copied during canonical reconciliation, and not exposed in client DTOs. Successful prefixes survive failure; retries cannot double-count already-persisted events. Errors are propagated, not swallowed. Unknown/expired/missing-target quests are not mutated by events.
11. **Callback/presentation integrity**: combat uses the same qid/st/d/qn envelope as quest/shop progress for onAddChaQuest. Removed npcScript quest_func's synthetic test quest; it now reports unavailable. Unconfigured definitions append an explicit server-not-configured notice to the original quest info.

## Client research and interpretation

Delegated read-only researcher confirmed GamePredef.as:336–359 enums, Player.as:1824–1910 acceptance prerequisites, QuestPanel.as:910–930 server-driven state103/104, and finish requests at441–503 (ordinary{i,s,n}, choose-one{i,s}). A visible quest/Finish button is not proof it is enabled; ready-to-turn-in is not already rewarded.

QuestCanvas.as:1009–1038 places rt/rn currency text in **requirements**, not rewards. Consumption/precise server semantics are not recovered, hence195 conservative blocks. aScriptText is an award tooltip at1334–1346, not executable completion evidence; qtest has no proven predicate.

Empty definitions include legitimate conversation quests and hidden/scripted checks. No universal “all type1/2 are conversations” heuristic was added. The66-ID conversation list is a narrow reviewed local catalog policy, not proof all other dialogue quests are invalid.

## Verification

- PASS: focused quest application/handler, NPC handler, combat application/handler, cache and PostgreSQL suites; affected vet; host build.
- PASS: critical authority/concurrency/claim/retry/read-safety suites repeated20 times.
- Coverage includes forged/empty build args; canonical acceptance; missing/truncated/incorrect type/target/count definitions; real kills; level/pet/friend thresholds; dependency/target/currency/unknown gates; NPC context; expiry; malformed JSON; one concurrent normal completion; UI read cannot write stale status; special quest cannot consume fabricated objectives; failure/retry prefix dedup; duplicate target rows; foreign participant and party inflight cleanup.
- Full-suite run: FAIL in existing pet stat fixtures, auth daily PM EXP14900 vs15400, and chat stat fixtures. Quest/combat/NPC/cache/persistence suites PASS. Do not present the full suite as PASS or alter unrelated gameplay to satisfy these tests.
- Race detector: BLOCKED by disabled CGO/no configured compiler. Repeated goroutine unit tests are not a race-detector substitute.
- Independent initial review found three blockers (special synthetic objectives, read writeback, permanently-spent credit claim); all were corrected and targeted regression-tested by parent after delegation quota ended.
- PASS deployment/startup: rebuilt/recreated only mcgame-server;12:34:08 startup loads5530 quests/itemAwards71, RTMP started, running/restart_count0. PostgreSQL/Redis untouched. Real Flash gameplay remains NOT TESTED by assistant.

## Remaining boundaries

The1692 blocked templates need authoritative conditions/targets/currency semantics, not a blanket success fallback. Quest46(craft emblem),166(capture Nấm Lục), many empty main/world/activity quests and custom systems remain unsupported where source conditions are missing. Do not claim “all quests implemented.”

Reward selection/distribution modes and multi-repository transactional grants are separate unresolved boundaries. Local serialization prevents concurrent same-process normal-quest claims but is not a PostgreSQL transaction or cross-process idempotency claim. Mid-consumption/grant/storage failure still requires diagnosis; never replay rewards automatically. Ended-battle retry availability is bounded by existing battle-cache retention; committed event keys prevent replay, but lost uncommitted events after expiry/restart are not reconstructed automatically.

Historical completed quests/awards are intentionally unchanged, including previously false completions. Existing matching kill counts are preserved rather than retroactively reset. No old failed or successful reward is replayed by this task.

Existing local CLI RLS advisory for four public configuration tables remains outside scope; no policy/schema edits were auto-applied.
