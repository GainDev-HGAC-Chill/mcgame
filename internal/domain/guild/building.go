// Guild building state is durable and isolated by guild extension point.
package guild

import "time"

type BuildingState struct {
	ID                int64
	GuildID           int64
	ExtendID          int
	TemplateID        int
	PendingTemplateID int
	Mode              int
	ActorID           int64
	Revision          int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (b *BuildingState) Pending() bool {
	return b != nil && b.PendingTemplateID > 0
}
