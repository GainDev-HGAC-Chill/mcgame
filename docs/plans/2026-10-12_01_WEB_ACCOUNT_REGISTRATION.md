# Web Account Registration — Implementation Plan

**Date**: 2026-10-12 · **Status**: Approved — user selected option2 and approved implementation with permissive credentials, including ID1/password1

**Delivery**: Implemented and deployed 2026-10-12. Independent focused Go tests/vet/Windows build, frozen frontend/edge Docker build, nginx config and host-side page/API isolation/throttling checks PASS. Actual account/Flash/Radmin smoke remains user-driven and NOT TESTED; see deployment results.

**Goal**: Make the existing Flash login registration button open a working Vietnamese web form on the game edge. The form creates a normal game account in the existing compose database; players then return to Flash, log in and create a character. User explicitly removed minimum-length/complexity restrictions: ID1/password1 must work. Only nonempty input and existing storage/client technical ceilings remain; do not introduce credential-strength policy.

Research base:
- `docs/memory/login-auth-flow.md`: Flash login uses an MD5 password argument; legacy rows migrate to bcrypt of the digest. New characters already receive the approved starter package.
- `docs/research/2026-10-12_04_NEW_CHARACTER_STARTER_PACKAGE_RESEARCH.md`: registration button exists but its web destination is missing.
- `docs/memory/dashboard.md`: administrative endpoints currently lack application authentication; keep them private.
- Read-only client and server surveys, plus live compose schema metadata obtained through Supabase CLI on 2026-10-12. No accounts were created or passwords queried during planning.

## 1. Confirmed mechanics

1. `docs/client/view/view/compFore/LoginCanvas.as` defines btnReg and opens GamePredef.SERVER_ADD_REG in a browser. `docs/client/predef/GamePredef.as` reads profile config.server.reg. The checked-in profile points to `/register.html`; the generated Radmin profile uses the same path at the selected Radmin IP. Neither Flash source nor original SWFs need edits.
2. Ordinary Flash login copies inputUser.text verbatim and sends MD5.hash(inputPass.text); it does not normalize username spelling/case or hash again on the server. The UI permits usernames up to50 characters and passwords up to20 characters. The bundled MD5 implementation is absent from the decompilation; upstream Adobe implementation uses UTF-8 and lowercase hex. Use MD5 of UTF-8 input and test Unicode vectors, but keep real Flash Unicode login marked NOT TESTED until verified. No ASCII-only restriction is approved.
3. Account.CheckPassword bcrypt-compares the exact supplied digest. Auth.Service.Register currently bcrypts its input unchanged. Passing plaintext directly to Register would store bcrypt(plaintext), which cannot authenticate Flash's MD5(plaintext).
4. AccountRepository.Create calls public.create_account; both the checked-in migration and live function insert supplied fields and return boolean true. Live public.accounts has varchar username/password fields and accounts_username_key UNIQUE(username), with RLS enabled. Username lookup uses exact equality; email is not unique. No schema migration is needed.
5. Service.Register creates only an account, not a session or character. New-character package remains level90/gold999999/goldBind999999 in Service.Create; registration must not credit account gold or create a character.
6. Edge publishes HTTP8888 and RTMPE1935 through the selected game bind address. Admin8080 is internal; dashboard3000, DB/cache/Studio/metrics remain loopback. Existing nginx serves static resources but has no registration API/page route.
7. Duplicate usernames found by the service precheck return ErrAlreadyExists, but insert races currently return wrapped PgError23505. The repository must map accounts_username_key conflicts into the same domain error without swallowing other failures.

## 2. Design

### 2.1 Public contract and validation

Serve a standalone Vietnamese registration form at `/register.html`, with three fields: username, password, confirm password. No email is collected in V1; existing Email remains empty.

The form sends a same-origin JSON POST to `/api/register`:

```json
{"username":"player01","password":"example-password","confirmPassword":"example-password"}
```

One application-layer registration boundary owns validation and conversion:
- Username: nonempty, at most50 UTF-16 code units to fit the existing Flash input (also fits Postgres varchar(50)); preserve spelling/case/whitespace. No minimum length beyond nonempty, alphabet/complexity or ASCII-only restrictions. Reject invalid UTF-8 and NUL, which Postgres text cannot store. ID1 is valid.
- Password: nonempty, at most20 UTF-16 code units matching the Flash input maxChars technical ceiling. Confirmation must match exactly. No complexity/ASCII/whitespace restrictions, never trim or normalize. Password1 is valid. Explain these as compatibility ceilings, not strength requirements.
- Reject missing/unknown fields, malformed JSON, additional JSON documents and oversized bodies (4KiB maximum).
- Compute lowercase hex MD5 once in Go, then call existing Register with that digest, yielding bcrypt(MD5_hex(password)). Do not persist plaintext/raw MD5 or log credentials.
- Responses:201 created,400 invalid input,409 username taken,429 overloaded/throttled,503 unavailable. JSON has a small stable error code and Vietnamese message; no hashes, database errors or secrets.
- UI validates for convenience but backend remains authoritative; disable repeat submit while pending, clear passwords on success and show instructions to return to Flash. Mention the new-character package without pretending account registration itself grants it.

### 2.2 Separate registration listener

Add `internal/presentation/http/registration` with a narrow Registerer interface. Run a separate Go HTTP listener on internal8081; it serves only POST `/api/register` and has body/header/read/write/idle timeouts and graceful shutdown. It is not the admin mux and exposes no account lookup/list/reset/grant functions.

Bootstrap config includes registration enabled/host/port; disabled by default outside compose. Compose enables it for the monolith, binds0.0.0.0 internally and adds expose8081 only, never a host port. Do not start it on gameplay-only line servers. An enabled listener startup failure participates in existing server-error/shutdown handling rather than silently claiming registration is ready.

Nginx adds an exact POST-only location `/api/register` proxied to8081. No `/api/` catch-all; no proxy to8080 or dashboard. The edge overwrites forwarded headers and retains the original Host so same-origin browser checks work. Reject mismatched Origin/Fetch Metadata and do not enable CORS. Use Docker DNS dynamic upstream resolution as already used by stream proxying.

### 2.3 Standalone frontend and edge build

Keep registration independent of the administrative Next.js application:
- Add a tiny TypeScript/pnpm package at `frontend/registration`, with a lockfile, HTML/CSS and TypeScript form code. No new frontend framework.
- Build with pinned TypeScript into a self-contained register.html (or a page plus assets under a dedicated `/registration/` prefix); no CDN dependencies or Supabase client/key.
- Add a multi-stage `docker/nginx/registration.Dockerfile`: pnpm build stage, then the existing nginx1.27-alpine runtime. Copy built registration assets into the edge image. Switch only rtmp-edge image definition to this build; keep existing game/static/profile mounts, port bindings and nginx stream behavior unchanged.
- Use exact `/register.html` and only the necessary registration asset prefix; set no-store for form/API, restrictive CSP appropriate to the generated build, nosniff and frame protection.
- Add generated output/node_modules exclusions without overwriting existing .dockerignore/.gitignore work.

### 2.4 Abuse protection and state ownership

- Nginx per-source-IP registration-only limiter:5 requests/minute with a burst of3, return429. Never apply this limiter to game assets or RTMPE. Shared VPN source IPs may share a quota; document retry behavior.
- Bound concurrent bcrypt work in Go to4 registrations; fail fast with429 when saturated. Release permits on every exit.
- The DB unique constraint is the authoritative duplicate gate. Map only accounts_username_key23505 to ErrAlreadyExists, even when concurrent calls pass the precheck.
- Accounts/password hashes live in Postgres; frontend carries no DB credentials. In-flight/limiter state is ephemeral and may reset on restart; it is abuse mitigation, not authorization or grant bookkeeping.
- HTTP is suitable only for the current trusted Local/Radmin context, not public Internet deployment. Warn against password reuse; require HTTPS before wider exposure. No claim that client-side MD5 replaces transport encryption.
- Existing shared secondary-password default is preserved and explicitly not advertised as hardened; a separate security change needs approval.

### 2.5 Touch points

| File/package | Change |
|---|---|
| `internal/application/auth/web_registration.go` + tests | Plaintext registration DTO, validation, MD5-hex compatibility conversion; delegate to existing Register |
| `internal/infrastructure/persistence/postgres/account_repository.go` + focused helper tests | Map username unique races to ErrAlreadyExists; honor unsuccessful create result |
| `internal/presentation/http/registration/{server,handler}.go` + tests | Narrow isolated listener, JSON/Origin/body/method controls, bounded bcrypt concurrency and safe responses |
| `internal/infrastructure/config/config.go` + tests | Opt-in registration listener defaults/environment |
| `cmd/gameserver/main.go` | Wire existing authService to registration listener, error lifecycle and Stop |
| `frontend/registration/*` | Vietnamese standalone TypeScript/pnpm form, styles and build/lockfile |
| `docker/nginx/registration.Dockerfile` | Build/embed registration page in existing nginx runtime |
| `docker/nginx/rtmp.conf` | Narrow page/assets and exact API proxy, registration-only rate limit/security headers |
| `docker/docker-compose.yml` | Build edge, internal registration env/expose; preserve all existing host bindings |
| `.dockerignore`, `.gitignore` | Targeted exclusions for generated frontend output/dependencies |
| `docs/memory/login-auth-flow.md`, `docs/local-lan-deployment/{radmin,setup,test-results}.md`, `README.md` | Registration URL/use/security/actual test status |

## 3. Non-goals

- No .as/SWF edits, embedded Flash signup UI or new RTMP registration RPC: existing browser button is sufficient.
- No Supabase Auth users, cloud database, direct browser Postgres/Data API access or anonymous INSERT policy: use existing game accounts through the Go backend.
- No email verification/recovery, password reset, social login or account administration: V1 is for trusted VPN users; these require separate flows and threat review.
- No automatic login/character creation/backfill/grants to existing players: approved starter package is already owned by character creation.
- No public admin dashboard, additional host-published port, router forwarding or firewall changes.
- No reset, volume deletion, seed replay or opportunistic RLS/security remediation. CLI metadata also reported disabled RLS on four configuration tables; keep Data API/Studio private and review separately before any broader exposure. Accounts RLS is enabled.
- Do not fix the pre-existing PM EXP auth test failure in this feature.

## 4. Verification

1. Unit tests: ID1/password1 must register and authenticate with the Flash digest; nonempty/technical ceilings, Unicode vectors and whitespace/case preservation, confirmation mismatch, no credential logging; stored bcrypt(MD5_hex) accepts Flash digest and rejects plaintext/double hash; existing Register/Login behavior unchanged. Use in-memory repo/session test doubles to avoid player/account data writes.
2. Handler tests:201/400/409/429/503, wrong method/media type/origin, body limit/unknown fields/trailing JSON, concurrency permits released, unrelated/admin routes404. Duplicate-race mapping test via wrapped PgError with exact constraint; unrelated constraints/errors remain failures.
3. Commands: `go test ./internal/application/auth ./internal/presentation/http/registration ./internal/infrastructure/config/... ./internal/infrastructure/persistence/postgres`; `go test ./internal/application/character ./internal/domain/auth`; `go build -o bin/gameserver.exe ./cmd/gameserver`; `go vet ./...`; `pnpm --dir frontend/registration install --frozen-lockfile`; `pnpm --dir frontend/registration build` and typecheck. Calibrate existing failures, including auth handler PM EXP14900 vs15400.
4. Deploy only game and edge using compose --no-deps after successful build. No migration/reset/seed/prune, no DB/Redis restart. Test nginx config before reload, then registration page HTTP200 and API validation without creating unwanted accounts. Verify edge cannot reach admin or Data API routes; DB/cache/admin/Studio/metrics bindings remain loopback.
5. User-driven smoke: click Flash Đăng ký → page opens at active profile host → create an intentionally chosen new test account → reject repeat username → return to Flash/login → create character → verify level90/gold999999/goldBind999999 → spend a small chosen amount, logout/relogin and verify no regrant. Account creation smoke requires approved test identity; do not invent accounts or log credentials.
6. Verify from a second Radmin machine. Host HTTP/infra/unit tests alone do not prove remote registration/Flash login/gameplay/persistence.
7. Update memory/deployment results with separate PASS/BLOCKED/NOT TESTED labels and remaining transport/security limitations.

### Independent design-review addendum

- Use UTF-16 ceilings for both Flash inputs; supplementary-character usernames must not exceed what maxChars50 can enter.
- Reject raw invalid UTF-8 before JSON decoding, and reject unpaired JSON surrogate escapes rather than silently accepting replacement characters. Test username/password/confirmation, preserving legitimate U+FFFD and valid emoji pairs.
- Origin cases include null/missing/cross-site metadata, scheme/port mismatch, deceptive host suffixes and spoofed forwarding. Nginx overwrites forwarding headers; do not trust client-supplied forwarding as an authorization boundary.
- Bcrypt concurrency permits remain held until hashing actually finishes even if a request context is cancelled. Verify fifth-request rejection with deterministic blocking test doubles, without writes to real accounts.
- Deployed smoke tests cover wrong methods, trailing/encoded paths, oversize body and only registration-specific throttling; never create real accounts simply to saturate rate/concurrency tests.

### Implementation outcomes and deviations

- Frontend pins pnpm10.11.0/TypeScript5.9.3 and uses node24-alpine build stage. HTML+external module/CSS are packaged into the nginx1.27-alpine edge image, independent of the admin dashboard; host pnpm is not needed.
- public registration uses separate internal8081, opt-in outside compose, with a synchronously acquired listener/error handoff and graceful shutdown. No admin mux/routes reused.
- Invalid Origin/malformed input/oversize produce400; methods405; missing Origin is accepted for nonbrowser clients only when cross-site Fetch Metadata is absent. Forwarded headers do not authorize requests. TLS-offload is not implemented; current deployment remains trusted-network HTTP.
- JSON decoder also rejects duplicate/case-variant field names and unpaired surrogates, preserving valid U+FFFD/emoji instead of silently normalizing.
- Host-side browser DOM/page/assets, safe negative requests, edge429 and administrative/Data API404 checks verified; no account was created for smoke testing. Browser tab closed before interactive test, so no interactive-submit PASS is claimed.
- Race testing remains NOT TESTED on the CGO-disabled Windows host. Known unrelated pet/chat/auth PM EXP baseline failures remain unchanged.
