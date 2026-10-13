# New-character starter package and registration button

## Approved scope

User requested automatic gifts whenever a new character is created: level90,999999 gold and999999 kim phiếu (`Character.GoldBind`). Each new character qualifies, including another character on an existing account. Do not touch existing characters or grant on reconnect; do not create accounts/characters automatically.

## Implementation

- `internal/application/character/service.go`: after NewCharacter/class defaults, apply the package before repository.Create. Fill HP/MP after stat recalculation.
- `starter_rewards.go`: starterLevel90, starterGold999999; call domain RaiseLevelTo, set Gold and GoldBind.
- RaiseLevelTo preserves normal progression: +89 to each base stat,445 free/max attribute points, within-level Experience0. Class aptitude remains sourced from class data; existing CumulativeExpCapped supplies correct level90 to the Flash list/login DTO.
- NewChar handler remains unchanged: calls service, sends onIcl, returns ToDTO. Tests exercise actual handler + application service and snapshot repository.
- CachedCharacterRepository.Create delegates persistence before invalidating cache. Postgres Create passes level/experience/attribute points/currencies into existing player.create_character_full. No DB function/schema change needed.
- No rank promotion, quest completion, skill auto-learning beyond existing class starter skills, equipment gifts, bag expansion, GM privilege or account-wallet grant is included.

## Registration investigation

- `docs/client/view/view/compFore/LoginCanvas.as:267–279`: btnReg enabled. Handler524–526 opens GamePredef.SERVER_ADD_REG via navigateToURL, not RPC.
- `docs/client/predef/GamePredef.as:10954–10956`: URL from config.server.reg; optional global override11047–11055.
- `frontend/s/sv/profile/config.xml:6`: http://localhost:8888/register.html; Radmin generated profile replaces host.
- No register.html exists in repository. nginx root try_files returns404 for missing paths; no registration proxy/endpoint is wired. internal/application/auth/service.go:Register exists but is not called by a registration endpoint.
- This task does not implement registration or change passwords/credentials. Existing .as files are read-only.

## Validation

- PASS application character + domain character tests.
- PASS focused auth handler tests NewChar/onIcl/GetLineInfo; returned DTO contains package, onIcl carries level90 and positive cumulative exp.
- Tests cover six classes/two genders, gift values present at repository.Create, class stat growth/aptitude/full HP/MP, normal privileges/capacity, no regrant on load/select, existing characters unchanged, duplicate/name lookup/persist failure paths.
- PASS go vet ./... and Windows server build.
- Full auth suite has pre-existing failure TestChooseCharacter_GrantsDailyPMLoginExpOncePerDay (14900 vs15400 PM EXP), previously recorded in deployment results. Not modified by this task.
- Docker image built and game server recreated alone via --no-deps. No reset/migration/seed/prune or direct DB/player modification. Remote Flash create/login/persistence still NOT TESTED; unit/health checks do not certify live gameplay.
