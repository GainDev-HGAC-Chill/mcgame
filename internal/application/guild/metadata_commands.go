// Guild metadata commands avoid replacing resource balances from stale snapshots.
package guild

import "context"

type GuildMetadataRepository interface {
	SetGuildName(ctx context.Context, guildID, actorID int64, name string) error
	SetGuildNotice(ctx context.Context, guildID, actorID int64, announcement, description string) error
}
