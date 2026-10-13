# Unlinked portal: sceneChange(757)

## Evidence

Runtime 2026-10-11 19:27:19: client sent `sceneChange(757)`; Go rejected it because the source instance had `from=0`. This is missing server-side routing metadata, not an invalid Flash argument type.

Client trace (read-only): `PlayerView.as:192–203` sends the clicked SceneItem instance ID. `GameScene.as:264–274` and `Core.as:370–378` keep scene-item instance/template distinct; template `type=3` identifies portals. The client receives server-selected map and arrival coordinates in `GameScene.sceneEnter`; it does not compute a destination from `from`.

Original `docs/database/game_data/TBL_SCENEITEM_INSTANCE.json:311–326` identifies instance 757: source map 3, source position (53,1094), template 38, destination label `Xuất Vân Thôn`. There is no original `from`/`to`. Current `supabase/seeds/data_tbl_sceneitem_instance.sql:30` likewise has `from=0`. `data_tbl_map.sql:29` identifies the uniquely named map 1 with safe point (1000,1000) and dimensions (2000,1500). Exact original portal arrival coordinates are not recovered.

## Repair policy

Existing nonzero `from` links retain their prior destination-instance coordinates/offset. For an unlinked portal only, resolve an exact, unique map-name match and use that map's validated safe point. Require portal template type 3, normal destination map type 2 and copy_flag=0, different source/destination, and finite in-bounds nonzero safe coordinates. Unknown/duplicate names, special/copy maps, non-portals and invalid coordinates fail closed. Never treat source coordinates or template ID as the destination; never pick the first ambiguous map or fabricate a reciprocal gate.

This is a documented server fallback policy, not a claim that (1000,1000) is the historical portal-757 spawn. No seed/DB/schema/player-position mutation or SWF edit is required. Only a successful normal player-triggered SceneChange subsequently updates position using the existing handler.

## Validation

Tests cover gate 757 → map 1 (1000,1000), unknown/ambiguous labels, non-portals, copy/special destinations, invalid safe point, and unchanged explicit 732→733 routing. Full scene service/handler tests, affected vet and host build PASS. Live portal traversal after redeploy remains NOT TESTED until operator confirmation.
