# Safe launcher backup and restore

Status: Implemented and focused-verified on 2026-10-23; live restore remains operator-driven.

## Confirmed behavior

- Menu has six actions: start/build, stop with database backup, reset as stop then start/build, Flash 30 FPS, database restore, and connection profile selection.
- Existing start already runs Compose with `--build`; the BAT also rebuilds `bin/game.exe` so launcher source changes are active.
- Database backups are machine-local artifacts and must not be committed.

## Design

- Store custom-format full database archives under `.local/backups/database`, with SHA-256 sidecars. Per explicit operator choice, every normal backup also atomically updates `supabase/backups/mcgame-live.dump` with a checksum sidecar. The 3.5 MB snapshot is tracked as regular Git content because GitHub rejects new LFS objects on public forks.
- Stop fails closed if backup cannot be created and verified.
- Reset calls the same safe stop path, then start; it no longer wipes volumes or reseeds.
- Restore lists local archives, requires confirmation, creates a safety backup, stops services, restores the selected archive, then runs normal start/build/migration checks.
- Add Local, LAN, Radmin and Online VPS profiles. Online accepts an explicit public IPv4/DNS host, binds only game HTTP/RTMPE publicly, and keeps dashboard local. It prints a warning that TLS and public-internet hardening remain operator responsibilities.
- Replace the machine-specific dashboard VPN bind with profile-configured values.

## Non-goals

- No automatic backup retention deletion.
- No encryption or private off-host disaster recovery. The chosen Git snapshot is full live data and is explicitly treated as public after push.
- No claim that the Online VPS profile is security-certified.
- No automatic execution of stop/reset/restore during implementation verification.

## Verification

- Unit-test backup discovery/naming, restore selection safety and online host validation/config rendering.
- Run focused `cmd/game` tests and vet.
- Build `bin/game.exe`; validate Compose config for Local and Online-compatible defaults.
- Inspect BAT routing. Do not execute lifecycle or database mutation commands.
