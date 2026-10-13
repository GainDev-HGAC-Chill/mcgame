# 60 FPS client-copy feasibility and timing audit

## Authorized objective

Operator approved a separate SWF copy targeting 60 FPS, preserving originals and stable server load. This does not authorize modifying checked-in ActionScript or resetting game/player state. No FPS patch or launcher/profile change has been applied.

## Direct evidence

Matching main SWF read-only copy at approved OpenCode temporary path: `mcgame-main-research.swf`, SHA256 `2FAE0552D5DB0B67414E20A337A2FCF7ABED330DECB126B2691A31BB80E395A5`, compressed CWS v9, header FPS 24. Actual GamePredef/RPCConfig/RemoteObj/CreatureView/PlayerView/BattleCreatureView/BattleStage P-code exported to `mcgame-fps60-pcode`. Actual EnterFrameMove/CharactorGraphic/AbstractGameRes read-only source exports in `mcgame-fps60-resources`. No exported ActionScript was edited.

FFDec's default JVM allocation initially failed with insufficient paging space; a 1 GiB heap could not parse the ABC and produced an incomplete export despite its OK message. A bounded 3 GiB JVM successfully exported all required classes. Therefore only successful later exports were used. No paging-file, reboot, security or server setting was changed.

## Why a constant-only patch is rejected

- MMOGame startup assigns stage.frameRate from GamePredef.GLOBAL_FRAME_RATE; header-only changes are overridden.
- GamePredef cinit assigns GLOBAL=24, GLOBAL_FRAME_RATE_24=24, GLOBAL_FRAME_RATE_20=20 and DEFAULT=28. DEFAULT is a speed calibration numerator, not a rendering cap.
- Actual `CharactorGraphic.as:108–195` advances the bitmap frame on every ENTER_FRAME, with compensation only below 12 real FPS. No high-FPS elapsed-time accumulator exists. At 60, unmodified animation/completion timing becomes roughly 2.5x faster.
- Actual `EnterFrameMove.as:94–107,118–173` calculates a fixed step count, but intervals <=35 ms additionally scale movement by elapsedMilliseconds*0.028. With globally scaled step=4 at 60, this double scaling covers about 47% of a route before the count expires. Need progress/time-based completion, not a blind constant swap.
- Actual CreatureView party following applies elapsedMilliseconds*0.025 at intervals <=40 ms, also double scaling at 60. Fixed position/escape/flying per-frame steps need their own timing treatment.
- Some battle movement effects do not set EnterFrameMove's second inBattle argument, so this is not restricted to ordinary scene travel.

## Server traffic

Actual RPCConfig assigns deny-count thresholds 24 for udcr, udcp and cbom. Actual RemoteObj allows a send when either wall-clock delay expires OR denied-attempt count reaches the maximum. Increased per-frame attempts can bypass cbom's nominal 1000ms interval more often at 60. Raising thresholds alone is only a nominal-rate mitigation; a validated wall-clock cap is stronger. Keep existing server authority/rate limits and do not increase encounters or server tick frequency to match rendering FPS.

## Required future implementation and acceptance

Implement a separate client copy with 60-Hz rendering and elapsed-time calibrated movement/animation, preserved battle sequencing, and verified RPC limits. This requires targeted AVM2 logic changes (not edited .as or global literal replacement). Audit all affected movement paths and validate bytecode output against the matching SWF. Keep original launcher/default and assets until the experimental copy passes live tests.

Acceptance requires measured rendered FPS, movement distance/time versus baseline, full route arrival, battle/escape/party timing, RPC calls per second, and sustained server autosave/no restarts. Unit tests, header metadata or server health alone cannot certify 60-FPS compatibility or load capacity.

## Status

- PASS: read-only matching-client bytecode/timing investigation, server-load hazard identified.
- BLOCKED: no timing-safe 60-FPS client patch currently implemented/validated; constant-only patch deliberately not activated.
- NOT TESTED: live 60 FPS gameplay, sustained load, and frame-rate measurement.
- Existing served client/default launcher remains unchanged at its native supported settings. No gameplay server restart, data reset or player modification performed for this audit.
