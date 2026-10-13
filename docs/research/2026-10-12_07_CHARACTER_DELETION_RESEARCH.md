# Character deletion — BLOCKED

## Follow-up: disabled Flash selection button

User confirms the gray Delete control is on the Flash selection screen, not dashboard. Checked-in CharSelectCanvas.as:265–276 initializes button7 (the onDelete control at964/1170) with enabled=false. Its select():1827–1854 changes visibility, not enabled state. No enabling assignment is present in this class. This explains the checked-in client-side disabled control separately from missing startDeleteChar/restoreChar backend support; exact served-class parity still requires verification. Do not edit ActionScript, fabricate a global enable callback, or pretend a backend hard-delete implements the original pending-deletion UI. This task has not enabled deletion or deleted any character.

## Outcome and scope

The reported deletion failure is explained by missing `startDeleteChar` and `restoreChar` registrations. A safe implementation is **BLOCKED**: the repository/domain/migration history has no persisted deletion deadline, scheduling/cancellation functions, or delayed-finalization lifecycle. Its existing `Delete` is an immediate hard delete and must not be wired to this client operation.

Read-only source inspection only. No Go/client changes, database queries or writes, migration, deployment, deletion, or commit were performed. Existing unrelated working-tree changes were preserved. The live database was not inspected; this finding concerns the supported repository persistence contract, not a claim about all live schema objects.

## Exact client requests

- `docs/client/view/view/compFore/CharSelectCanvas.as:1170–1173`: delete prompts for the protection/secondary password, not the login password. `Language.DELETE_BY_PASS[0]` advertises default `123456`.
- `:1403–1406`, `:1473–1480`: disables login, shows `POPU_WAIT` with a three-second display timer, hashes entered text using `MD5.hash`, then calls `startDeleteChar(int(selectedChar.cid), passwordMD5)`.
- `docs/client/rpc/RemoteObj.as:89–114`: dynamic methods prepend the method name and **null Responder**. Returning an RPC result or error alone is not a client-visible completion callback for deletion.
- `CharSelectCanvas.as:993–1007`: the restore/cancel action confirms the character name and calls **`restoreChar(_core.guid, selectedChar.cid)`**. It sends no secondary-password argument. The first argument is client-supplied legacy account/guid information, not authorization; server ownership must use authenticated `RPCContext.AccountID`.
- `:1590–1598`, `:1830–1853`: a pending-deletion character cannot log in, and offers restore rather than delete.

The three-second wait is UI text/timing, **not** evidence of a three-second deletion grace period. No authoritative grace-period duration or finalization policy was recovered; do not invent one.

## Callback and delayed-state contract

The checked-in `docs/client/system/CallBack.as` is incomplete. Existing read-only JPEXS exports of the matching served SWF were inspected under `C:/Users/gaingain/AppData/Local/Temp/opencode/`:

- `mcgame-callback-as-research/scripts/com/qeedoo/game/system/CallBack.as:536–574`: **`onIcl(param1:Object)`**, one flat character-list argument, hides `POPU_WAIT` and repopulates character selection. This is the supported list refresh, but client source alone does not prove the original server's exact successful delete/restore callback sequence.
- `:5397–5405`: **`onDeleteFaild()`**, exact spelling, zero arguments, hides `POPU_WAIT` and alerts `Language.CALLBACK_S[56]`. That text specifically says a guild leader must transfer leadership before deleting. It is **not a generic authentication/storage failure callback**.
- `:866–877`: **`a(message:String)`** is the supported generic alert; it does not hide `POPU_WAIT`. A future unavailable/error response must account for that separately and not send a fabricated success/list state or the guild-leader error for unrelated failures.
- No dedicated delete-success or restore callback was found in the complete exported callback class. Do not invent `onDeleteChar`, `onRestoreChar`, or a Responder return contract.
- `mcgame-select-research/scripts/com/qeedoo/ui/view/comp/CharactorSelectCanvas.as:268–302`: positive **`delTime` is Unix seconds**, multiplied by 1000 for the displayed final deletion date; it sets `deleted = true`, dims the character, and renders the deadline from `Language.CHARACTOR_DELETE[0]`. Pending characters remain in the list and retain their ID/name. Omitting them or immediately destroying their row breaks restore semantics.

## Existing server and secondary-password authentication

- `internal/presentation/rtmp/handlers/auth/handler.go:RegisterHandlers` registers `setDeletePass`, but neither deletion request. `internal/presentation/rtmp/handlers/stub/stub_handler.txt` has only a commented `startDeleteChar` reference.
- `auth/characters.go:characterListEntry` always sends `delTime: 0`.
- `internal/domain/character/repository.go` offers CRUD `Delete`, not schedule/cancel. `internal/domain/character/character.go` has no deletion deadline. `internal/application/character/service.go:Select` checks ownership but has no pending-deletion gate.
- `internal/infrastructure/persistence/postgres/character_repository.go:Delete` calls `player.delete_character(id)`. `supabase/migrations/20260514042214_batch4_complex_functions.sql:49–59` defines an unconditional `DELETE FROM player.characters WHERE id = p_id`, with no account guard or recovery window. Migration search found no character deletion deadline/schedule/cancel support. Query/scan paths do not load such state.
- `docs/client/view/view/compDragable/DPassPanel.as:543–552,579`: `setDeletePass(newMD5, oldMD5)` uses a real `Responder`; an empty returned string means success, other strings are displayed errors.
- `auth/misc.go:SetDeletePass` updates the account secondary password; `utils.CacheSecondaryPassword` pushes `setClientDP(newMD5)` after persistence. This RPC configures the password; it does not schedule deletion.
- `utils.VerifySecondaryPassword` loads the authenticated account and delegates to `Account.CheckSecondaryPassword`, which constant-time compares the supplied MD5 to `SecondaryPasswordHash()`. That accessor retains the documented default MD5 of `123456` for an empty stored value. A future deletion operation must verify this server-side on every request, not trust client cached `delPass`, guid, or the login password. Do not log the supplied digest.

## Required follow-up before implementation

1. Obtain an approved persisted deadline model and authoritative grace-period/finalization policy. No migration is authorized by this task.
2. Add account-scoped, atomic schedule/cancel persistence; preserve pending rows and deadline across restart, reconnect, repeated requests and cache flushes. Reject foreign IDs, active/online characters and guild leaders as required by verified rules.
3. Project persisted deadlines into `onIcl` and deny selecting pending characters server-side. Authenticate start using account secondary-password verification; restore uses authenticated ownership, never its supplied guid as identity.
4. Establish the successful callback sequence from server evidence/capture; use the verified list and alert signatures. `onDeleteFaild()` is only for its proven guild-leader condition.
5. Test wrong/missing secondary password, foreign account, malformed numeric ID, missing character, pending selection, retry/idempotency, concurrent start/cancel, expired deadline, restart/cache consistency, persistence errors and callbacks. Unit fakes must not be used to claim actual persistence support.

No new implementation tests were added/run because the prerequisite persistence contract is absent. Character deletion remains unavailable; no success is claimed.
