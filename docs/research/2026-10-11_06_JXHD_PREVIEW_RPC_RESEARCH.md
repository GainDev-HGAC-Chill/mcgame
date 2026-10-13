# Missing featured-activity reward-preview RPC

## Runtime finding

At 19:38:27 the server reported repeated missing `getJXHDAward` calls. Within the inspected 20-minute window, this was the only unique missing method. This does not establish that all client features have complete server implementations.

## Client contract

Read-only client research: `docs/client/view/view/compDragable/JXHD.as:1113–1122` calls `getJXHDAward` with exactly one string activity key and a Responder. `onGetJXHDAward` at 697–707 expects `[key, groups]`, where groups contain `title` and a `list` of `{t,i,ii,n}` items. It explicitly returns if groups is missing or has length zero. Thus `[requestedKey, []]` is a handled no-preview response, not a reward grant or success claim.

Eight keys: ConsumeAward, Wawagame, PointAward, ComboShop, Juhuasuan, Treasurebowl, Happyline, PointNotice. `onGetJXHDList` at 1281–1338 eagerly requests previews for eligible entries, explaining repeated errors on a single panel open. Existing consumption/point/shop RPCs have different payloads and cannot be safely aliased.

## Change and limits

Register exact `getJXHDAward` and validate selected character, one string argument and known key. Return explicit empty preview because this handler has no configured preview reward metadata. No callback push, fake items, claim, currency write, database mutation, or blanket suppression of unknown RPC errors.

The existing GetJXHDList advertises synthetic activity windows; this patch does not validate/replace those windows or implement eight event systems. Empty previews are not cached by the client's handler and may be fetched again on reopening. Real reward previews require a separately implemented authoritative configuration/provider. Other missing methods still need their own trace when reported.

## Verification

Tests cover all eight key envelopes, empty-array rather than null, invalid args/unknown keys, unauthorized access, and actual dispatcher registration. Full activity/RTMP tests, affected vet and host build PASS. Flash panel behavior after redeployment remains NOT TESTED.
