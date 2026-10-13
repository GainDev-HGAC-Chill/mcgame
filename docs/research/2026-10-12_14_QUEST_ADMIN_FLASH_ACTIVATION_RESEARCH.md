# Quest admin: definitive Flash external-catalog activation research

## Verdict

**BLOCKED — complete edited quest catalogs cannot be activated consistently by the existing compiled ChFields overlay.** It mutates fields on existing rows only; it cannot add/delete rows, replace tables, or rebuild any of the three index families. It runs asynchronously 30 seconds after configuration completion, not as a prerequisite for NPC quest availability. A server restart and client restart/relogin do not make database edits appear in the embedded client catalog.

This is read-only protocol research. No backend/database/player writes, gameplay actions, reward actions, asset patches, invented callbacks, or edited ActionScript were performed. The only repository deliverable is this document. FFDec-generated decompilation artifacts are exclusively in the approved unique temporary folder below.

## Provenance and actual extraction

Read `CLAUDE.md`, `docs/PROJECT_KNOWLEDGE_BASE.md`, `docs/memory/database-infrastructure.md`, `docs/memory/npc-quest-system.md`, `docs/memory/local-lan-deployment.md`, the existing loader export, and the FPS research/tooling evidence. The old `.claude/agent-memory` location is absent in this checkout; repository memory and actual compiled exports were used instead.

Live `docker inspect mcgame-rtmp-edge` confirms these read-only mounts:

- `I:\mcgame\frontend\s` → `/usr/share/nginx/html/s`.
- `I:\mcgame\docker\client\radmin.xml` → `/usr/share/nginx/html/s/sv/profile/config.xml`.

Container-side `sha256sum` independently confirms the served files match the mounted host assets:

| Artifact | SHA256 |
|---|---|
| Served encoded main `frontend/s/sv/6a41a5ba5c0006ab4e0c1321a2b05027` | `07DA21616D5AC18E2354CD9F2F04AB126FAB85726033E6E861A7499BD064CD38` |
| Main after the loader's XOR decoding | `2FAE0552D5DB0B67414E20A337A2FCF7ABED330DECB126B2691A31BB80E395A5` |
| Served ChFields `frontend/s/res/73d7c51b6d3b9dd6252ba98408212f90` | `84F9D2CE566D5D73951EA8643F2AF25AC8851A50F93D5C20ED2494B56BEE92FC` |
| Exported `ChFields.as` | `5671BFD72F18D92BD8F49306A5B622D7E04AD7FE40332CE6B425A9F46EEA7AEC` |

The encoded main was decoded **in memory only** using XOR `(offset+1)%3` and hashed; its result matches the pre-existing `.local/research/mcgame-main-research.swf`. That verified copy was the input for the main-class exports. ChFields was exported directly from the actual mounted asset, not from `moreData.as`.

Export root (abbreviated **R** in citations):

`C:/Users/gaingain/AppData/Local/Temp/opencode/mcgame-questadmin-resource-20261012-activation/`

Actual successful FFDec 26.3.0 commands, using the installed BellSoft Java and bounded 3 GiB heap:

```text
java -Xmx3g -jar C:/Users/gaingain/AppData/Local/Temp/opencode/ffdec_26.3.0/ffdec-cli.jar -selectclass com.qeedoo.ui.resource.ResManager,com.qeedoo.game.config.Version,com.qeedoo.game.system.AbstractGame -export script <R>/main I:/mcgame/.local/research/mcgame-main-research.swf
java -Xmx3g -jar C:/Users/gaingain/AppData/Local/Temp/opencode/ffdec_26.3.0/ffdec-cli.jar -export script <R>/chfields I:/mcgame/frontend/s/res/73d7c51b6d3b9dd6252ba98408212f90
java -Xmx3g -jar C:/Users/gaingain/AppData/Local/Temp/opencode/ffdec_26.3.0/ffdec-cli.jar -selectclass com.qeedoo.MMOGame,com.qeedoo.game.system.Core,com.qeedoo.game.predef.GamePredef,com.qeedoo.game.utils.UrlUtil -export script <R>/startup I:/mcgame/.local/research/mcgame-main-research.swf
java -Xmx3g -jar C:/Users/gaingain/AppData/Local/Temp/opencode/ffdec_26.3.0/ffdec-cli.jar -selectclass com.qeedoo.ui.view.compDragable.NpcFuncPanel -export script <R>/npc I:/mcgame/.local/research/mcgame-main-research.swf
```

Each completed with all requested classes exported and `OK`. No script import/replacement command was used.

Additional verified pre-existing authoritative exports:

- **D** = `C:/Users/gaingain/AppData/Local/Temp/opencode/mcgame-quest-sort-research/scripts/com/qeedoo/game/data/DataManager.as`.
- **C** = `.local/research/mcgame-callback-as-research/scripts/com/qeedoo/game/system/CallBack.as` (complete served-main callback export).
- **L** = `.local/research/mcgame-loader-research/scripts/GameLoader.as`.

## Six questions answered

### 1. Only text/EXP, or arbitrary fields on existing rows?

The **merger is generic at the field level**, not limited by a whitelist to names/text/EXP. `R/chfields/scripts/ChFields.as:71093–71123` iterates `fields[table][row]`, then each property, and assigns:

```text
if (param1[table][row])
    param1[table][row][property] = overlayRow[property];
else
    errors.push([table,row,property]);
```

It preserves the existing row object's identity. Existing index entries referencing that object can therefore observe changed non-key scalar fields. This is **not** proof that already-rendered panels or copied objective structures refresh: the loader dispatches no catalog/UI refresh event.

The **actual shipped quest payload**, unlike the generic merger, contains only `startText`, `completeText`, and `awardExp`: read-only enumeration found 5,530 occurrences of each under `fields[45]`. No entries were found under quest award 46, requirement 47, loop 58, or prerequisite 64. First quest entry is at `ChFields.as:31714`. Thus the shipped external SWF is not a complete quest catalog or a database-fed transport.

### 2. Can it add rows, replace rows, or replace entire tables?

**No.** The row must already be truthy. Missing rows are recorded in the returned error array; there is no row/table allocation and no assignment to `param1[table][row]` itself. It cannot add a new quest/objective/award/prerequisite, or replace a row/table wholesale. A missing table can throw rather than be created because the table access is not guarded.

`R/main/scripts/com/qeedoo/ui/resource/ResManager.as:1198–1201` instantiates the exact `ChFields` class and calls `setData(GameData.d)`. The returned error array is assigned but not inspected, logged, or used to enforce success.

### 3. Can omission, null, or a deletion marker delete rows?

**No supported deletion mechanism exists.** Omitted rows/properties are untouched. A null property value is only a field assignment; a supposed deletion marker is just another field. The compiled method contains no row removal, table clearing, or tombstone interpretation. Existing rows remain embedded and indexed.

`D:698–703` does contain an unrelated `delData` helper, but ChFields/ResManager never calls it; it also does not remove index entries. Its existence does not establish an external-catalog deletion protocol.

### 4. Are all three index families rebuilt after the overlay?

**No.** `D:41–49` assigns `gameData = GameData.d` then invokes `createDataIndex()` in the constructor. `D:335–367` creates all three per-table index families and inserts row object references (`createIndex`, `createIndex2`, `createIndex3`). Neither `ChFields.setData` nor the full `ResManager.loadTxtFields` invokes that routine or any incremental index maintenance.

Verified compiled `GamePredef` key mapping:

| Table | `gameDataIndex` | `gameDataIndex2` | `gameDataIndex3` |
|---|---|---|---|
| Quest 45 | `subType` | `startNpc` | `type` |
| Award 46 | `qid` | none | none |
| Require 47 | `qid` | none | none |
| Prerequisite 64 | `qid` | `itemId` | none |
| Loop 58 | `nid` | none | none |
| NPC quest mapping | `nid` | none | none |

Source: `R/startup/scripts/com/qeedoo/game/predef/GamePredef.as:10246–10272,10334–10360,10409–10436`.

Changing an indexed field on an existing row leaves it in its **old bucket**, while its object now reports the new value. A hypothetical replacement without a rebuild would likewise leave old row references. Generic field assignment is therefore insufficient even for existing-row edits involving quest type/subtype/start NPC, prerequisite item ID, or loop NPC.

`D:141–168` has `addNewData`/`updateData`, but these only assign rows (and the former dispatches a per-row event); they do not rebuild the three indexes. The complete callback's observed `addNewData` calls (`C:7808,8482`) concern runtime item/slot data, not a quest catalog synchronization channel. No generic catalog-refresh callback was established; no new callback is proposed here.

### 5. Does overlay completion gate startup or NPC availability?

**No; it is explicitly delayed and non-blocking.** Actual compiled order:

1. `R/main/scripts/com/qeedoo/game/system/AbstractGame.as:21–27`: `createCore → createStage → createUI → loadConfig`.
2. `R/startup/scripts/com/qeedoo/MMOGame.as:313–346`: `createCore` obtains `DataManager.getInstance()` at line 321, so embedded catalog indexing occurs before config loading.
3. `MMOGame.as:176–184`: config completion sets `_configComplete = true`, installs config, then schedules `setTimeout(ResManager.loadTxtFields,30000)`.
4. `ResManager.as:1177–1180`: loading is skipped entirely if embedded item table 29 row 1 already has a nonempty description.
5. `ResManager.as:1182–1231`: SWF load is asynchronous. On initialization, call `ChFields.setData(GameData.d)`, then refresh only instantiated NPC `onServiceText` fields. Exceptions are swallowed. IO/security failure also proceeds to the next cosmetic-resource load. There is no readiness flag, RPC gate, index rebuild, or quest-panel invalidation in this path.

`R/npc/scripts/com/qeedoo/ui/view/compDragable/NpcFuncPanel.as:824–835` takes the server's `npcQuest.list` and resolves each qid through `_core.data.getChaQuestFullData`. That uses embedded `gameData[TBL_QUEST]`, indexed requirements/awards/prerequisites (`D:613–631,692–707`). Missing local quest data returns null (`D:614–616`). Server availability IDs are not full template replacement payloads.

Therefore the first NPC availability evaluation is **not guaranteed** to happen after overlay load, and the path cannot guarantee a consistent edited catalog even if its network request succeeds. Already-created/copied quest views may remain stale after scalar mutation.

### 6. Which resources and version parameters are requested?

- `L:213–225`: default main request is `./6a41a5ba5c0006ab4e0c1321a2b05027?v=<loader version>`; `client` can override its base. Loader `version` defaults to an empty string. This parameter selects/cache-keys the main request; it does not import database catalogs.
- `MMOGame.as:186–202`: profile `assets.txt` is requested with `?v=` plus a random string. Its lines are parsed as `hash=value` into `Version.ASSETS_VERSION` (`159–173`).
- Actual `Version.as:10` is `0.9.9a33.490`. `ResManager.hash` (`710–718`) computes MD5 of the logical resource path and uses `SERVER_ADD_RES + "res/" + hash`. When its assets dictionary entry is truthy, it appends **`?v=490`**, the final three characters of the compiled version — **not the value from assets.txt**. `_NEW.swf` names bypass that suffix.
- Compiled logical path is `resource/ui/ChFields.swf` (`GamePredef.as:191`); its MD5 is `73d7c51b6d3b9dd6252ba98408212f90`. Request is `<configured resource base>/res/73d7c51b6d3b9dd6252ba98408212f90`, conditionally suffixed `?v=490`.
- The actual served `frontend/s/sv/profile/assets.txt` is just two bytes `0D 0A` (CRLF), so it contains **no ChFields version entry**. With this profile file, the normal startup ChFields request has **no version query**. Changing a manifest entry's value alone would not introduce an admin revision query.

These are compiled request-construction findings plus live mount/asset verification, not a claimed packet capture. No JSON sidecar consumer is present in the examined loader/config/ChFields path; changing JSON exports cannot change the literal compiled `fields` array.

## Activation decision and proof boundaries

The existing overlay can only demonstrate a narrow subset: mutation of existing, non-index-key fields after its delayed load, with shared row identity. That is **not complete consistent catalog activation**, and no asset modification was made to test that subset.

The quest admin API contract in `2026-10-12_13_QUEST_ADMIN_API_IMPLEMENTATION.md` permits requirement/award/prerequisite insertions/deletions and shared-loop changes. Those operations exceed this compiled overlay's capabilities. Its `mode:"restart-only"` and `clientRefreshRequired:true` must not be interpreted as proof that restarting/relogging synchronizes Flash definitions. Fresh client processes still reconstruct the same embedded tables and old index buckets, then apply the same shipped text/EXP overlay. A reconnect in the same process is even less of a reload.

**Precise blocker:** no verified compiled external hook exists here for full table/row replacement plus deletions, followed by all three index rebuilds, before NPC availability or dependent quest views consume the catalog. There is also no established live catalog-refresh callback. Consequently a complete no-AS-edit/no-asset-patch activation proof plan cannot be offered using this overlay alone. Backend persistence/server reload may be independently valid, but full Flash consistency remains BLOCKED under the current constraints.

No further code work or activation was performed.
