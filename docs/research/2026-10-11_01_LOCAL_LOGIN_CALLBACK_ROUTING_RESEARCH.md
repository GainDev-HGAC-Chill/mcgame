# Local Flash login callback routing

## Reported failures

- `lvUp`: #2095/#1069 because the scene `CallBack` has no method of that name. This is client→server RPC, not inbound callback.
- `onIcl`: #2095 with inner #2088 `Proxy.getProperty`. This can occur when NetConnection attempts to look up `onIcl` on the Global client's `CallBackGlobal`, which extends Proxy but does not implement that callback or getProperty.

## Evidence from the served client

The checked-in `docs/client/system/CallBack.as` is only an incomplete decompiler stub. JPEXS FFDec 26.3.0 was installed from its official release through winget (installer hash verified). Read-only exports were made in the approved OpenCode temporary directory, not into client source.

`GameLoaders.swf` exports `DisplayLoader.completeHandler`, which reconstructs downloaded bytes by XOR with `(index+1)%3`. This existing loader transformation was reproduced on a temporary copy of `frontend/s/sv/6a41a5ba5c0006ab4e0c1321a2b05027` solely for inspection. No served SWF, `.as`, or protection mechanism was modified.

Recovered `CallBack.onIcl(param1:Object)`:

1. Hide waiting/login UI; show character selection UI.
2. If a freshly created character name is pending, find the matching flat character entry and call scene RPC `chooseCharactor(int(id))`.
3. Otherwise set state `choose` and populate `charactorDataList`.
4. If argument is null, show creation state.

Recovered `CharSelectCanvas.charactorDataList` enumerates rows into ArrayCollection. Each selection row reads flat `id/name/exp/classId/gender/iconCode/delTime`. The existing server payload shape matches these reads; no speculative payload changes are justified.

`MMOGame.createCore` creates separate NetConnections: `new RemoteObj(new CallBack())` for scene and `new RemoteObj(new CallBackGlobal())` for global. Existing `docs/client/system/CallBackGlobal.as` extends Proxy and has no `onIcl`.

## Server defect and minimal repair

`internal/presentation/rtmp/handlers/auth/characters.go:GetLineInfo` was sending `onIcl` before returning line information, regardless of connection type. `getLineInfo` belongs to the Global login flow, so this attempted callback is invalid there.

Remove this cross-flow side effect. Scene post-auth, `icl`, creation and voluntary return to character selection retain their existing character-list callback. Add a Global-connection guard at the final `sendOnIclCallback` boundary so other call paths cannot send scene data onto Global.

Regression tests cover Global apps `tcn[/]`, `master/test[/]`, scene apps, line-list return values, callback absence on Global, and unchanged one-argument scene character list. Live login/relogin still requires projector confirmation after deployment. No database reset, player modification, or SWF edit is part of this repair.
