# Movement & Position

Position update RPC: `udcp(posX, posY, centerX, centerY, dir, moveInfo)`. Speed validation: `distance * 1000 / elapsedTime` compared against `serverSpeedThreshold` (~300 units/sec).

## Scene portal fallback (2026-10-11)

`sceneChange` takes the clicked scene-item instance ID. Instance 757 is a type-3 portal on Vân Đài/map 3 labelled Xuất Vân Thôn but has no server `from` link, causing invalid arguments. Nonzero explicit links retain existing behavior. Unlinked portals resolve only an exact unique normal/non-copy destination map name with finite in-bounds safe coordinates; 757 resolves map 1 safe point (1000,1000). No arbitrary first match/source-coordinate/template-ID fallback. This is a server safe-point policy, not independently recovered historical arrival coordinates. [Research and tests](../research/2026-10-11_05_UNLINKED_PORTAL_RESEARCH.md).
