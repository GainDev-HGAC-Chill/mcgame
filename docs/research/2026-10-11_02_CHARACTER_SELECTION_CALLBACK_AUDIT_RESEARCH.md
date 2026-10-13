# Character-selection callback audit

## Scope and evidence

Investigated the reported Flash `updateWSPPanel` #2095/#1009 after choosing Gain. Complete callback and panel implementations were exported read-only from the matching served SWF using JPEXS to the approved temporary directory. No `.as` or served SWF was edited.

Recovered source references (under `C:/Users/gaingain/AppData/Local/Temp/opencode/`):

- `mcgame-callback-as-research/scripts/com/qeedoo/game/system/CallBack.as:2156–2163`: `updateWSPPanel` obtains `PANEL_WAR_BATTLE` via `getUI`, immediately calls `updateWarSpritePanel`, and does not check initialization.
- `mcgame-panel-research/scripts/com/qeedoo/game/view/ViewManager.as:1110–1116`: `getUI` creates an absent panel; a non-null result does not mean its Flex children are initialized.
- `mcgame-panel-research/scripts/com/qeedoo/ui/view/compDragable/WarSpritePanel.as:2935–2948`: normal `showPanel` waits for `initialized`, sets selection indices to 1, then requests `getSpWarData` with a Responder.
- Same panel `3036–3054`: update immediately indexes sprite data/templates and writes child controls. Any missing template/control can produce #1009; exact failing dereference is not established by the supplied stack.

## Repair

Remove unsolicited `updateWSPPanel` from auth login callbacks. Do not substitute an arbitrary timer: the existing client's panel-open RPC is the readiness signal. Preserve `getSpWarData` response and upgrade-triggered updates. Skip the provider's panel-state `warSprite` key during login payload merging; retain scalar `cData.warSprite` and `cData.battleSprite` currencies, without mutating provider state. Login no longer carries unused `warSpritePanel` data.

## Wider callback review

Reviewed explicit selection callbacks against the complete served `CallBack`: `onReadCharData`, `onChooseCharactor`, `updateUserBar`, `initLongBuff`, `updateJewelOffPrice`, `allServerMailsWarning`, `initMountTimer`, `initDecoShowTimer`, `initPRSShowTimer`, `addOrMinusPmExp`, `updateScenePmLevel`. Method names/signatures exist. Empty timer/mail objects bypass optional panel updates; mount required fields are supplied. No additional unconditional exception was proven, so these callbacks were not speculatively altered.

An independent client trace identified `updateDiaryData` from the daily-login event as initialization-sensitive. However, its broadcaster filters for positive map ID, so normal fresh selection does not receive it before entering a scene. No change was made without evidence of an actual regression. This remains a review risk, not a proven failure or a guarantee of complete compatibility.

## Regression coverage

- Test entire minimal character-selection callback order twice (relogin), one non-null argument per callback, preserved currency scalars/discount, no lazy-panel push or payload, and no provider-state mutation.
- Test legacy `warSpritePanel` cannot re-enable the login push.
- Test panel-open handler returns existing upgraded state, defaults on fresh state/error/no character, and never writes player state.
- Focused auth regressions, WarSprite handler/service/domain, RTMP, utils, activity and character tests PASS; affected vet and host build PASS.
- Full auth package retains the pre-existing daily-PM assertion failure (`14900` versus `15400`); unrelated reward logic was not changed.
- Flash character selection and opening WarSprite after redeployment remain NOT TESTED until operator confirmation.
