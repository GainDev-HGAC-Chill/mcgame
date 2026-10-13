# Autosave cache crash and Flash logout error

## Incident evidence

At 2026-10-11 19:07:15 +07:00, `mcgame-server` logged `fatal error: concurrent map iteration and map write`. Stack: `encoding/json.Marshal` → `PlayerCache.saveToRedis` → `SaveAllDirty` → center periodic persistence. Docker restarted the server at 19:07:16; inspection showed `restart_count=1`. This is the confirmed cause of the connection loss. The reported Flash #1009 in `Battle.battleOnEnd` → `sceneLeave` → `Core.logout` is a separate client cleanup exception following connection loss; it was not established as the cause of the server crash.

The preceding session completed quest 160 (reward response `flag=true`), moved to map 3, completed one battle and began another with its pet. Logs show pet RPCs running concurrently with the autosave failure. This is server evidence of activity, not proof that every client animation/callback completed correctly.

## Ownership defect

PlayerData's RWMutex protected only outer containers and setter calls. GetCharacter/GetItem/GetPet/GetQuest/GetSkill and list reads returned cached entity pointers; setters also retained caller pointers. Battle setup's pet `SetBinded`/`SetClientState` mutates `Pet.Property` outside the cache lock. Consequently Redis JSON iteration and DB dirty snapshots observed the same mutable nested map as gameplay. Taking another shallow map copy or recovering the panic cannot fix this fatal runtime condition.

## Repair

- Clone cache entities on writes, non-dirty additions and reads. Initial DB load also detaches inputs. Copy dynamic maps/slices recursively without JSON numeric coercion; preserve nullable fields, objectives, pet SoulInfo and private bonus maps.
- Full Redis and dirty DB snapshots detach entities under PlayerData lock; JSON/Redis/DB work uses those snapshots after releasing the entity lock.
- Serialize each player's DB persistence and Redis snapshot writes independently to avoid out-of-order saves.
- Generation-aware acknowledgement leaves later changes dirty. Successfully saved deletion prefixes are acknowledged even if later work arrived, so saved tombstones are not retried unnecessarily.
- Character position/stat patches are atomic under cache ownership lock; UpdateCachedCharacter detaches both its callback input and returned value. Debug dumps/stats do not read unprotected live containers.
- Combat test expectations were aligned with the earlier removal of invalid `lvUp` callback; no combat reward rule was changed.

The dynamic-container copy helper handles acyclic JSON-style maps/slices, not arbitrary cyclic object graphs or external concurrent mutation of the caller's input during a setter call. Gameplay reads obtain detached copies and must explicitly Save/Update them. Exported PlayerData fields remain for compatibility; production access must continue through its methods.

## Validation and limits

Tests cover original-input and single/list-read isolation, stable dirty/full snapshots, concurrent pet map writes versus repeated JSON serialization, private bonus-map independence, exact int64 retention, nullable fields, character position/stats, and dirty/deletion acknowledgement during an intervening update. Focused cache/pet/valuecopy/combat/quest/NPC/RTMP suites and `go vet ./...` PASS; host build PASS. Final Linux race detector PASS for cache/domain pet/valuecopy. Independent static review found no remaining confirmed serious repair-induced defect after corrections. Compose redeployed only game server at 19:21:44; `game check` PASS and new container running/restart_count=0 immediately after deploy, not yet a long-running stability result.

Windows has no gcc/clang or golangci-lint available. Race tests run in an ephemeral official Go Alpine container with a read-only source mount, not against gameplay DB/Redis. No player reset, volume deletion, reward replay, or SWF change.

Existing partial-persistence failure retry behavior and concurrent full-row read/modify/write conflicts are not comprehensively redesigned here. Abrupt crashes before an autosave can lose unflushed write-behind changes; do not promise the last session was fully persisted or replay rewards without checking. Live reconnect/battle/autosave after deployment and Flash cleanup remain NOT TESTED until operator confirmation.
