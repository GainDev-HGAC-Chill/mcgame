# Live database snapshot

`mcgame-live.dump` is the operator-approved full live PostgreSQL snapshot used to reproduce the current database state. It is tracked as a regular Git file because GitHub does not allow a public fork to upload new Git LFS objects.

This snapshot can contain Supabase Auth accounts, sessions, player characters and all other live rows. Treat every committed version as public data. Removing a later file does not remove older copies from Git history.

`MCGAME.bat` updates the canonical snapshot after every successful database backup. Timestamped safety copies remain under `.local/backups/database` and are ignored by Git. The launcher validates custom-format archives before stopping or restoring the stack.

The dump preserves original owners and ACLs. Restore runs as the Supabase image's `supabase_admin` superuser; using `postgres` fails on managed objects such as `vault.secrets` and can leave a partial database.
