# Web Registration — Contract and Security Research

## Approved user intent

Implement option2: existing Flash Đăng ký button opens a standalone web registration form backed by the existing game database. User explicitly permits ID1/password1 and rejects credential-strength/minimum-length rules. Preserve input case/whitespace; only nonempty input and existing database/client compatibility ceilings apply. Do not create accounts or change passwords on the user's behalf as a smoke-test shortcut.

Implementation plan: [web registration](../plans/2026-10-12_01_WEB_ACCOUNT_REGISTRATION.md).

## Delivery and independent verification

Implemented/deployed2026-10-12. Parent ran focused application auth/public registration/config/Postgres/cmd gameserver/character/domain auth tests, full vet and Windows build successfully, then built frozen pnpm frontend and both Docker images. nginx -t passed before deploying game/edge only with --no-deps. An independent source reviewer found no serious regressions. Startup08:34:35 confirms isolated registration8081 and RTMP.

Live page/JS/CSS all200; browser accessibility DOM loaded the form. Safe no-write tests verified API invalid JSON fields400, spoofed Origin/forwarding400, oversized body400, GET405, trailing/encoded routes404/400, administrative/Data API routes404 and burst invalid-requests429. Host bindings remain private where required, and original trial30 loader remains200. No successful signup was sent to the live database and no account/character/password was created/changed for testing.

Real new-account creation/Flash login/Unicode/Radmin second-machine progression and persistence remain NOT TESTED. ID1/password1 compatibility passed in application/handler test doubles. Race detector was not run on the CGO-disabled host. Browser tab was closed before interaction testing; DOM/HTTP success does not claim successful interactive signup.

## Confirmed compatibility

- LoginCanvas.as:995–997 copies username directly and MD5-hashes password; input descriptors282–325 carry50/20 maxChars. Force login1376–1383 follows the same digest convention.
- Account.CheckPassword compares the supplied string against bcrypt, while Auth.Service.Login performs no extra hashing or normalization. Legacy migration bcrypts the existing MD5 hex string.
- Auth.Service.Register currently bcrypts its supplied password unchanged. Calling it with browser plaintext produces a Flash-incompatible account. A separate web-registration application boundary must validate plaintext and convert to lowercase MD5 hex exactly once before existing Register.
- Upstream Adobe MD5 uses ByteArray.writeUTFBytes and lowercase hex; the exact bundled implementation is absent locally. ASCII1/password1 compatibility is directly testable; Unicode vectors can be unit-tested against UTF-8 MD5, but actual Flash Unicode login remains a manual verification item.
- Normal login username lookup is exact. Do not lowercase or trim registration usernames, or silently normalize passwords.

## Database sources

Compared checked-in schema/function migrations against read-only compose metadata using the project Supabase CLI, with connection values loaded from docker/.env and never printed:

- public.accounts username varchar(50), primary key(id), UNIQUE(username) named accounts_username_key. Email is optional and not uniquely constrained.
- public.create_account accepts supplied account fields, performs a plain INSERT and returns true. Existing account repository already calls it; no DDL/migration/backfill is necessary.
- public.accounts RLS is enabled. Database writes remain privileged server-side through the existing auth repository; registration does not require anonymous INSERT policies, browser Supabase keys or Supabase Auth users.
- Service precheck handles duplicate usernames, but race-time23505 from INSERT currently remains a wrapped PgError. Map only accounts_username_key conflicts to the domain duplicate error.
- CLI advisory also reports RLS disabled on public.game_tuning/gateway_config/gm_settings/server_settings. This is separate existing configuration security debt, not permission to auto-remediate or expose the Data API; maintain loopback access and exact registration proxy boundaries.

## Exposure boundaries

- Flash btnReg navigates profile config.server.reg, currently /register.html at Local/Radmin host; no client edits are needed.
- Existing admin HTTP8080 contains privileged unauthenticated player/action/grant endpoints. Registration must use a separate internal listener8081 and an exact nginx /api/register proxy, never an admin/public catch-all.
- The frontend is a standalone TypeScript/pnpm package, independent of the administrative dashboard. Build it into the edge image so host users need not install pnpm. No credentials, Supabase client or CDN assets belong in the page.
- Keep game HTTP8888/RTMPE1935 as the only selected-profile host publications. Dashboard, DB/cache/Studio/metrics remain loopback;8081 is container-internal only.
- JSON body limits, same-origin checks, registration-only edge throttling and bounded concurrent bcrypt mitigate abuse. Weak passwords are explicitly user-approved for trusted VPN play, not Internet deployment; HTTPS is required before public exposure.

## Baseline calibration before registration implementation

- PASS existing starter application/domain and focused newChar/onIcl/getLineInfo tests. Package still level90/gold999999/goldBind999999 only on new character creation.
- Existing auth handler failure: TestChooseCharacter_GrantsDailyPMLoginExpOncePerDay14900 vs15400, previously recorded.
- Existing pet failures verified: TestLearnSkillFromBookUsesItemInstanceAndAppliesBuff(24 vs146), TestEquipPetItemMovesItemToPetSlotAndAppliesBonuses(24 vs5), TestUnequipPetItemMovesBackToBagAndClearsEquipmentBonus(65 vs0), TestDeletePetSkillConsumesItemAndClearsSlot(20 vs120).
- Existing chat failures verified: TestSay_StatExp_AppliesExperienceAndPersists(finalHp26 vs148), TestSay_StatLevel_RaisesTargetLevelAndPersists(hpMax115 vs319).
- Do not fix or conflate these with registration verification.
- PASS baseline `go test ./pkg/rtmp/...`; the browser-registration feature must not alter transport handshakes or original SWFs.

## Independent verification checklist

- Browser: page loads from profile host; entering ID1/password1 satisfies client validation. Do not submit a successful registration or invent test accounts without approval of the specific test identity.
- HTTP negative smoke uses empty/malformed credentials only, so it cannot create DB rows: wrong method/content type/origin, mismatching confirmation, unknown fields and body-size boundary. Confirm any route fallback/encoded-path variant cannot reach administrative APIs.
- Origin checks must include null/missing with cross-site Fetch Metadata, scheme/port changes, deceptive host suffixes and overwritten forwarding headers. JSON input must not silently replace malformed UTF-8/unpaired surrogate escapes.
- Registration concurrency limits must hold permits through actual bcrypt completion, including after request cancellation. Test fifth-request rejection with blocking doubles rather than real successful account writes.
- Deployed compose inspect must show registration8081 internal only; DB/cache/Studio/dashboard/metrics remain loopback, and no /api/ catch-all forwards to admin8080 or the Data API.
- Current host has node.exe but no pnpm on PATH. Use the isolated Docker frontend build rather than installing unrelated global tools; available disk was checked before image build. Preserve .local exclusion because it contains live Docker disks/data.
