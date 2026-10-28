# Launcher Operations

## Docker build-cache policy

`MCGAME.bat` rebuilds `bin/game.exe` and delegates Start/Reset to `cmd/game`. The stack start path runs `docker builder prune --all --force` before Compose builds and again after every build attempt, including failures. This intentionally trades incremental Docker build speed for bounded disk use. It does not prune running images, containers, named volumes, or database data.

The game-server `Dockerfile` copies only `api/`, `cmd/`, `internal/`, and the separately layered `pkg/` Go source instead of the repository-wide context. Dashboard and Flash assets therefore cannot inflate the game-server source layer. A verified cold build transferred an 8.77 MB context instead of the previous roughly 2.27 GB snapshot.

Docker Desktop's VHDX may retain its physical host size after cache deletion. Cache pruning prevents repeated build layers from accumulating, while reclaiming already allocated host space still requires `fstrim` followed by offline VHDX compaction.
