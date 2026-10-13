# Opt-in 30 FPS client trial

## Scope

After the 60-FPS timing audit, the operator requested a 30-FPS trial. This is an experimental separate SWF copy, not a timing-certified client upgrade. Original assets, `.as` sources, default launcher, player data and game server configuration are untouched.

## Reproducible Go tool

Build `go build -o bin/clientfps.exe ./cmd/clientfps`, then run `bin/clientfps.exe` from repository root. It pins both original SHA256s, parses SWF tags/ABC class traits, finds target method bodies and uniquely matched instruction sequences, changes fixed-length byte operands and re-encodes using the existing loader XOR transformation. No global numeric replacement or AS compiler is used. It refuses existing outputs; handled write failures clean up only files created by that invocation.

Original SHA256:

- main `6a41a5ba5c0006ab4e0c1321a2b05027`: `07da21616d5ac18e2354cd9f2f04ab126fab85726033e6e861a7499bd064cd38`
- loader `GameLoaders.swf`: `e0092d207d306299bff3611fb37750c6c550be9fed6f1252d34cd34d1a707689`

Generated siblings:

- `frontend/s/sv/6a41a5ba5c0006ab4e0c1321a2b05030`
- `frontend/s/sv/GameLoaders30.swf`

These are gitignored reproducible outputs. The loader copy redirects its three verified main-asset URL strings to the new same-directory filename, preserving profile/resource relative paths.

## Exact changes

The decoded main SWF changes exactly nine bytes:

1. Header target FPS 24→30.
2. GamePredef initializer GLOBAL_FRAME_RATE, GLOBAL_FRAME_RATE_24 and GLOBAL_FRAME_RATE_20→30. Retain GLOBAL_FRAME_RATE_DEFAULT=28 as the movement calibration numerator. The low-FPS settings toggle consequently cannot restore 20 on this trial copy.
3. RPCConfig deny-count thresholds for udcr, udcp and cbom 24→30. Keep wall-clock delays and server limits unchanged; count-based bypass remains, so this is nominal-FPS mitigation, not a strict request-rate guarantee.
4. EnterFrameMove.nextPoint's short-frame compensation threshold 35→30ms.
5. CreatureView.followHandler's threshold 40→30ms.

At nominal 30FPS (~33ms/frame), the latter two changes bypass double-scaling. Ordinary movement stays approximately `int(9*28/30)*30 = 240` units/sec, matching nominal 24FPS's `int(9*28/24)*24 = 240`. Short frame intervals <=30ms can still trigger old compensation; jitter/party/full-route behavior needs live checks.

## Remaining experimental risks

CharactorGraphic and other unscaled frame-driven animations can run about 25% faster than at 24FPS, including battle callbacks and fixed-step visual effects. This patch intentionally does not claim identical animation timing. Rendering target is not a measured sustained frame rate. No server capacity guarantee follows from these constants.

Only one projector should log into an account. Roll back by closing the trial and opening the original through MCGAME.bat option2. No DB rollback/reset is needed.

## Validation

- PASS: Go tests cover source hashes, exactly nine changed main bytes, unique instruction matching, SWF round-trip, loader redirection, refusal to reapply/overwrite, and unchanged default launcher URL.
- PASS: `go test ./cmd/clientfps ./cmd/game`, affected vet and both tool builds.
- PASS: independent JPEXS export of the generated main verifies all three FPS assignments=30, DEFAULT=28, three RPC thresholds=30 and both scoped movement comparison thresholds=30. Branch/tag lengths remain unchanged.
- PASS: original asset hashes unchanged after generation.
- BLOCKED: both HTTP asset fetches and Docker inspection failed because Docker Desktop Linux engine/stack is currently unavailable. No service restart or database operation attempted.
- NOT TESTED: real projector target/actual FPS, login, full movement paths/portal/party, battle pacing and server RPC/autosave load under trial.
