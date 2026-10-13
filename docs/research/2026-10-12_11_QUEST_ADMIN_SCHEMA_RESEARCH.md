# Quest Manager admin: live schema and safe aggregate editing

Date: 2026-10-12. Status: research only; implementation requires parent/user-approved plan.

## Scope and evidence

Read `CLAUDE.md`, feature-pipeline and Supabase skills, project knowledge index, the five `docs/database/details/TBL_QUEST*.md` references, quest/NPC/start-grant/loop/battle/dashboard memory, and the full-catalog requirement audit. Inspected generated models, cache/repository loading and live function definitions. Live queries used `bin/supabase.exe db query --db-url` against compose, with password and direct port loaded silently from `docker/.env` and password URI-encoded. CLI command help and compose `pg_dump --help` were checked. No secrets printed, mutations, DDL, backup execution, player actions, resets, commits, worktrees, or client edits. This document is the sole authored artifact.

Queries inspected information_schema columns, pg_constraint (including inbound quest FKs), pg_indexes, pg_policies, pg_trigger, role_table_grants and pg_proc; queried live counts, distributions, nulls, orphan relationships, loop pools and malformed quantities. Read-only queries ran separately, not in one exported snapshot; counts describe the observation, not a durable backup.

## Executive findings

- The five tables are real relational catalogs under `data`, not JSON override records.
- Live counts: quest **5530**, require **4765**, award **1437**, pre **2017**, loop **11**. Old prerequisite reference says 2007; live has ten more kind=2/type=45 rows. Do not regenerate current data from that historical JSON.
- All five have only `PRIMARY KEY (id)` plus ordinary NOT NULL constraints. No FK, enum domain, CHECK, child-qid index, custom trigger, identity, sequence default, revision or updated_at. Only PK indexes exist.
- Every non-ID numeric catalog field is unconstrained PostgreSQL `numeric` (no precision/scale); IDs are `integer`. All observed numeric values are integral, but the database itself allows fractional numbers.
- Preserve SQL NULL, empty string, -1, legacy types, duplicates, missing targets, scripts, disabled ranges and orphan rows unchanged unless explicitly edited with verified semantics.
- Existing generic write functions target `player.game_data_templates`, **not** `data.data_tbl_quest*`. There is no existing aggregate catalog save function to reuse.
- Quest EXP is stored in **award_exp_re**, consumed as `QuestTemplate.AwardExpRe`, and emitted as Flash **awardExp**. There is **no award_exp column**. Old table docs describing awardExpRe as blank/reserved are not applicable to this live schema.
- Loop metadata is shared across many quests; it is not a per-quest owned child. Editing one quest must not silently update/delete its loop definition.

## Exact column contract

Notation: `N` = NOT NULL; `?` = nullable. No column has a default; none is an identity. Lists preserve live column order. Numeric means unbounded `numeric`, not int4/float8.

### data.data_tbl_quest (table ID 45)

| Column | PostgreSQL type | Nullability |
|---|---|---|
| id | integer | N, PK |
| a_script_text | text | ? |
| at | numeric | N |
| award_exp_re | numeric | ? |
| color | numeric | N |
| complete_text | text | ? |
| finish_npc | numeric | N |
| gender | numeric | N |
| info | text | ? |
| is_rebirth | numeric | ? |
| lm | numeric | N |
| max_level | numeric | N |
| min_level | numeric | N |
| money_num | numeric | N |
| money_type | numeric | N |
| name | text | ? |
| pre_quest_type | numeric | N |
| qtest | text | ? |
| req_class | text | ? |
| rn | numeric | N |
| rt | numeric | N |
| script_give_up | text | ? |
| start_npc | numeric | N |
| start_text | text | ? |
| sub_type | text | ? |
| type | numeric | N |
| start_grant_item | integer | ? |
| start_grant_qty | integer | ? |

Current award_exp_re: no nulls; -1 occurs 99 times, 0 occurs 2749 times, many positive values. req_class has 24 SQL NULLs. is_rebirth has 5462 NULLs, 23 zeros, 45 ones. Do not coerce null rebirth to zero on save merely because gameplay treats both similarly. Start grants: 160→335×1, 162→515×1, 164→517×1; all other start_grant_item fields NULL.

### Child and loop tables

| Table | Exact columns in order |
|---|---|
| data_tbl_quest_require (47) | id integer N PK; item_id numeric N; kind numeric ?; num numeric ?; q numeric N; qid numeric ?; type numeric N |
| data_tbl_quest_award (46) | id integer N PK; b numeric N; item_id numeric N; kind numeric ?; num numeric ?; q numeric N; qid numeric ?; type numeric N |
| data_tbl_quest_pre (64) | id integer N PK; item_id numeric N; kind numeric ?; num numeric ?; qid numeric ?; type numeric N |
| data_tbl_quest_loop (58) | id integer N PK; info text ?; max_level numeric N; min_level numeric N; name text ?; nid numeric N; num numeric N; refresh numeric N; script_get text ?; script_give_up text ?; team numeric N |

Max IDs observed: quest 50770, require 9032, award 3690, pre 3812, loop 16. These are observations, **not safe next-ID allocators**. If creating rows is approved, reserve IDs under a transaction/global allocator lock or add a sanctioned sequence with approved DDL; never browser-side max+1. Updates keep IDs immutable, including child IDs.

## Relationships: logical references, not enforced FKs

- require/award/pre.qid → quest.id. Use each child row ID as identity; qid is grouping, not unique.
- quest.start_npc/finish_npc and loop.nid → data_tbl_npc.id; sentinel -1 supports class NPC routing. Nonpositive finish NPCs have existing system-turn-in paths.
- require/award `(type,item_id)` is polymorphic: 12 creature, 19 equipment, 29 item. Never resolve by numeric ID alone.
- pre kind=2.item_id → quest.id (type=45 or legacy -1). pre kind=1,type=29.item_id → item.id; `num` is held quantity, not turn-in consumption.
- start_grant_item → item.id (no FK). Runtime defaults start_grant_qty to 1 when null/zero; catalog must retain stored value unchanged.
- loop children: type=7 and sub_type `7-<pool>`. Current service maps loop 1 to pool 10; other loops map to their own IDs. This is **not a universal direct FK** from subtype suffix to loop ID.
- Runtime `player.character_quests`, `player.quest_history`, `player.character_quest_loops` reference catalog IDs logically; catalog edit must never rewrite these records or replay grants.

Live loop child pools: 7-2=275; 7-3=6; 7-4=2; 7-5=123; 7-6=6; 7-7=17; 7-8=1; 7-9=924; 7-10=38; 7-16=3 (sum 1395). No 7-1 pool exists.

| Loop id | NPC | Rounds | Min–max level | Refresh seconds | Team |
|---|---:|---:|---|---:|---:|
|1|48|60|200–150|43200|0|
|2|1069|15|30–160|600|0|
|3|246|10|200–200|600|2|
|4|277|10|30–160|600|0|
|5|280|20|50–160|600|0|
|6|434|20|70–160|600|1|
|7|929|10|50–160|600|1|
|8|250|10|200–160|600|0|
|9|48|200|70–160|259200|0|
|10|1519|10|200–160|600|2|
|16|3|10|50–160|600|1|

Inverted/high ranges intentionally disable historical loops. A generic min<=max validator must not prevent an unrelated name edit or silently “fix” them.

## Enum meaning and live distribution

These are application/client enums, not PostgreSQL enums. UI shows known labels plus exact raw values for historical unknowns.

| Field | Meaning / live counts |
|---|---|
| quest.type | 1 tutorial=140; 2 main=790; 3 class=285; 4 world=775; 6 activity=810; 7 loop=1395; 9 callboard=712; 10 growup=211; 11 guild=320; 12 lucky=92. Client constants also define 5 business/8 challenge; no live rows. |
| quest.at | 1 all=5393; 2 choose-one=134; 3 class-filtered=3. Catalog metadata does not prove server implements selection correctly. |
| quest.pre_quest_type | 0 no prior-quest gate=11; 1 AND=5481; 2 OR=38. Item prerequisites remain required independently. |
| quest.gender | 0 male, 1 female, 2 unrestricted; all 5530 currently 2. |
| quest.req_class | `all` or pipe-wrapped IDs, plus 24 nulls; retain original serialization/order. |
| quest.money_type | -1=16; 0=204; 1=990; 2=4279; 3=39; 10=2. Use current currency contract, not ambiguous old prose. |
| quest.rt | 0=5246; 1=24; 2=180; 4=27; 10=33; 13=20. rt/rn are **requirements**, not extra rewards; 195 quests have rn!=0 and are blocked pending verified mechanics. |
| require.kind | 1 item=2181; 2 kill=1239; 3 pet surrender=1344; null=1. |
| require.type | 12=2583; 19=93; 29=2088; legacy 0=1. Supported normal pairs: (1,29), (1,19), (2,12), (3,12). |
| award.kind | 1 item=1430; 2 pet=6; null=1. Client skill kind=3 exists, but no live rows and current reward code must not be advertised as supporting it. |
| award.type | 29=1349; 19=81; 12=6; legacy 0=1. |
| award.b | 0=69; 1=1368. Historical JSON differs (68/1369). Current reward code forces bound=true instead of honoring b; edit storage faithfully, warn about runtime behavior. |
| pre.kind/type | kind1=60, kind2=1956, null=1; type45=1375, -1=580, 29=61, 0=1. |
| loop.team | 0=6; 1=3; 2=2. Party/seasonal semantics are not fully implemented. |

`q` must remain a numeric quality code, not replaced with a UI color code. Requirements: q<0 skips item color filtering; q>=0 uses exact derived item/equipment color; pets use q/10 growth-quality threshold. Live require q set: -1,0,1,2,5,6,10,11,12,15,16,19,20,23. Award q set: -1,0,1,5,6,7,10,11,12,13,14,15,16,17,19,20; current item/equipment/pet grant paths interpret it differently. Scripts are historical text, not executable admin input.

## Historical anomalies: preserve, do not normalize

- require row 8780: item_id=0, kind/num/qid=NULL, q=0, type=0.
- award row 2631: b=0, item_id=0, kind/num/qid=NULL, q=0, type=0.
- pre row 3675: item_id=0, kind/num/qid=NULL, type=0.
- 115 requirements have a non-null qid not present in quest; including the null-qid row gives the earlier audit's 116 unattached rows. Awards/prerequisites have zero non-null orphan qids.
- require 8548 has qid=1000000, num=0, kind=1, type=29, item_id=3550, q=0; it is historical orphan data, not a reason to delete/fix the whole table.
- Two prior-quest targets are missing: pre 1345 gates quest2629 on quest2625; pre1331 gates quest2590 on quest2586.
- Earlier full audit confirms missing objective targets in quests3983/5005/50170/50171 and a duplicated kill target in quest1665. Retain the duplicates and missing references until an explicit source-backed edit.
- 1692 templates were blocked by the earlier gameplay classifier (1493 unclassified,195 currency,4 missing targets). Admin editing metadata must not claim these are implemented or automatically add fabricated objectives.

Validate **changed** fields/rows and newly introduced references. Existing unchanged unsupported rows must remain savable and byte/semantic-identical in stored values, with visible warnings. Never convert unknown dropdown values to the first known option, null to 0/empty text, -1 to 0, scripts to trimmed text, or empty children to fabricated requirements. Removing an anomalous row requires explicit row selection and audit.

## Existing DB functions and safe reuse

Live definitions, all security invoker with empty search_path:

| Function | Reuse |
|---|---|
| player.get_game_data_row(p_table text,p_id integer) → jsonb | Reads schema-qualified regclass table with id=$1; can reuse for individual lookup, not consistent aggregate snapshot. |
| player.list_game_data_rows(p_table text) → setof jsonb | Reads entire table ordered by id; useful for loader, not paginated admin listing. |
| player.count_game_data_rows(p_table text) → bigint | Count helper only. |
| player.upsert_game_data_template(p_table text,p_record_id integer,p_data jsonb) → void | Writes player.game_data_templates only. **Do not use to save quest catalogs.** |
| player.delete_game_data_template(p_table text,p_record_id integer) → void | Deletes override records only; not catalog children. |

Other quest functions operate on player progress/history/loop state (`create_quest_progress`, `update_quest_progress`, `record_quest_history`, get/delete progress and get/upsert/deactivate loop state); none is appropriate for template editing. Live pg_proc search found no function body referencing data_tbl_quest and no quest-admin function.

Generic regclass validates a SQL identifier, not admin authorization/table scope. Keep an explicit five-table route allowlist; do not expose arbitrary p_table to browser callers. Add dedicated schema-qualified aggregate read/save/list functions after approval, not long SQL strings in Go or independent browser Supabase writes.

## Proposed UI/API field shape (recommendation, not existing endpoints)

Use snake_case in raw catalog records to match SQL/model JSON; camelCase only for wrapper fields and presentation labels. Return nullable numerics/text without coercion; prefer numeric decimal strings in the admin API, with SQL numeric parsing on write, to avoid JavaScript precision loss. IDs are safe int4 integers. Existing float64 Go generated models are unsuitable as a lossless editor DTO (nulls become zero values); use raw JSON/nullable DB DTOs.

```json
{
  "questId": 162,
  "revision": "opaque-server-token",
  "quest": {"id":162,"award_exp_re":"100","is_rebirth":null,"start_grant_item":515,"start_grant_qty":1},
  "requirements": [{"id":101,"qid":"162","kind":"1","type":"29","item_id":"515","num":"1","q":"-1"}],
  "awards": [],
  "prerequisites": [],
  "loopRef": null,
  "warnings": [],
  "activation": {"databaseSaved":false,"serverReloadRequired":true,"clientRefreshRequired":true}
}
```

Example is intentionally partial field illustration, not the live complete quest162 record. GET must return all stored fields. Full UI tabs: identity/dialogue, eligibility/prerequisites, requirements, EXP/currency/awards, accept-time grant, advanced raw/script fields. Type-aware searchable NPC/item/equipment/creature/quest pickers show ID, table and name; missing referenced records stay visible as raw selected IDs. Unknown enum options remain selected with warnings. Render info/dialogues/scripts as escaped text; HTML preview requires sanitization.

Proposed save envelope: `{questId, expectedRevision, patch:{quest:<changed fields>, requirements:{update:[],insert:[],deleteIds:[]}, awards:..., prerequisites:...}, reason}`. Omitted fields/collections mean unchanged, **never delete-all**. Bind qid server-side to URL questId. Verify every update/delete child ID actually belongs to that quest; reject cross-quest ID or qid reassignment. Return canonical aggregate plus new revision and activation status; 409 for stale revision; 422 with field/row-qualified validation errors; 404 missing quest. List API needs search/id/type/sub_type/NPC/level/blocked-warning filters, counts and pagination; do not download all child tables to each browser view.

Loop metadata should have a separate editor/endpoint with its own revision and affected-quest count. A quest page may display linked loop info read-only. If parent explicitly requires one save affecting shared loop and quest, include an explicit loop patch and expectedLoopRevision, lock both aggregates in deterministic order, audit both, and show all affected children before confirmation.

## Atomic single-quest save + optimistic concurrency

After approval, implement one DB transaction/function for root + three owned child groups:

1. Require verified pre-change backup receipt for this schema/data deployment, a trusted operator attribution and nonempty reason. Do not trust actor identity supplied by the browser as authenticated identity.
2. Acquire per-quest transactional lock, lock root row, lock/read current children in deterministic order. Read before snapshot and compute canonical content fingerprint (root + child arrays sorted by id; include nulls and every field). Compare expectedRevision before any writes; 409 on mismatch. A sidecar monotonically increasing bigint revision can supplement fingerprint without changing catalog shape.
3. Validate allowlisted changed fields, owned child IDs, immutable root ID/qid, type-aware targets and integral/range rules. Preserve untouched legacy content. No automatic global cleanup, cycle repair, reward grants or history mutation.
4. Apply explicit root patch/owned child inserts/updates/deletes. Allocate child IDs safely across quests. No replace-all that changes child identities or drops unrecognized rows.
5. In the **same transaction**, append immutable audit with before/after full raw aggregate, old/new revision, actor/source/request ID/reason/time/backup reference. Roll back catalog write if audit fails. Make request ID idempotent and return stored result for retries.
6. Return post-save aggregate and revision only on commit. Audit restore should be another reviewed optimistic write, not blind replay; it cannot undo runtime grants.

No native revision, audit or allocator was found in these five tables, so these are new approved infrastructure, not present capabilities. All catalog writers must participate in locking/revision protocol; a parent row lock alone does not stop independent child inserts because there are no FKs. If arbitrary SQL writers must be supported concurrently, design table locks/serializable coordination or enforce exclusive write paths; do not claim a cooperative advisory lock prevents non-cooperating writers. Include current content hash so older external edits are detected at save.

DDL workflow follows CLAUDE.md: first verified backup; apply functions/sidecars to compose, then capture with `supabase db diff --db-url ... -f ...`. Never add wholesale validating FKs/CHECKs as a convenience: historical orphan/missing targets would fail and cleanup is not authorized. qid indexes may be useful if list/save profiling warrants them; no index/DDL was executed here.

## Backup before any data/schema writes (not executed)

Use compose DB's installed pg_dump; its help was successfully read. A single custom-format dump of all five tables gives a consistent snapshot of their schema/data. Do not rely on old committed seeds as the current backup, and do not stream binary dump through Windows PowerShell text redirection.

Suggested explicit future commands, after backup authorization and host directory selection:

```powershell
docker exec supabase-db pg_dump -U postgres -d postgres --format=custom --strict-names --table=data.data_tbl_quest --table=data.data_tbl_quest_require --table=data.data_tbl_quest_award --table=data.data_tbl_quest_pre --table=data.data_tbl_quest_loop --file=/tmp/quest-admin-before.dump
docker cp supabase-db:/tmp/quest-admin-before.dump <approved-host-backup-directory>/quest-admin-before.dump
docker exec supabase-db pg_restore --list /tmp/quest-admin-before.dump
Get-FileHash <approved-host-backup-directory>/quest-admin-before.dump -Algorithm SHA256
```

Container /tmp is server-local, not a host scratch directory; host artifacts belong in a specifically approved backup directory (or project-preferred Temp/opencode if authorized). Check every exit code and resulting file size, record timestamp/compose target/CLI versions/hash/table counts and TOC, and perform isolated restore validation when authorized. Before function/sidecar DDL, also take a schema-only full compose database dump (or a full custom database dump if authorized), since table-filtered pg_dump does **not** include independent functions/dependencies. New audit/sidecar objects must join later backups. pg_dump archive inspection is not proof of a successful restore.

Stop admin saves during capture/deployment; keep backup immutable and separate from mutable seed exports. Never restore whole database/player state automatically for a quest template rollback. After approved catalog edits, seed export remains one table per file under supabase/seeds per project convention; it is distinct from pre-write recovery backup. No dump or backup file was created in this research.

## Runtime activation and security boundaries

- `GameDataRepository.GetAllByTable` reads real data tables via list_game_data_rows; Upsert goes to override storage. Manager.LoadAll loads startup cache. Loop child pools are memoized per Manager in deferred_questloop.go. There is no verified atomic catalog hot reload. Do not promise immediate gameplay changes merely because DB save succeeded, or call LoadAll on a live manager as an atomic refresh.
- Initial safe activation: report restart/reload required, coordinate with parent and user, and refresh/relogin Flash (client catalog delivery/versioning must be researched by the owning track). No restart/deployment is authorized by this research. Later hot reload needs complete cache snapshot swap and invalidation of loop/policy indexes.
- Changing an active quest may change canonical objectives on subsequent runtime reads/events; warn operators and show active-use impact rather than resetting progress. Completed histories and assets remain untouched.
- All five data tables have RLS enabled, no policies, and grants to postgres/service_role; inspected grants contained no anon/authenticated table access. Server credentials only; never expose service_role to frontend. Function privileges/route authentication must be explicitly reviewed for the new API.
- Current dashboard memory states application authentication is absent and security depends on loopback/internal ports; do not describe a new quest page as securely authenticated or expand network exposure.
- CLI reported pre-existing RLS disabled/exposure warning for public.game_tuning, public.gateway_config, public.gm_settings and public.server_settings. Surface to parent/user; no remediation performed. Enabling RLS without planned policies can break access and is out of this task's scope.

## Plan-ready acceptance tests

Untouched save preserves nulls/sentinels/scripts/unknown rows; unrelated edit succeeds with preserved missing refs; new invalid target/pair rejected; cross-quest child ID denied; missing versus explicit deletion distinguished; two concurrent saves yield one success/one409; inserted ID allocation safe across different quests; malformed later child/audit failure rolls back earlier root changes; idempotent retry does not duplicate children/audit; stale external content detected; shared loop changes separately confirmed; backup failure prevents writes; API reports database-only activation honestly; no player/history/grant SQL called. Validate old anomalies and table counts unchanged except the explicitly edited aggregate. No test writes or schema changes were performed in this research.
