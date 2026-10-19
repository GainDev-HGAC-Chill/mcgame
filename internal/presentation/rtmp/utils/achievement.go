// Open-sourced by BaoLT

// Achievement callback helpers preserve the served Flash three-argument contract.
package utils

import (
	domainachievement "mcgame-server/internal/domain/achievement"
	"mcgame-server/internal/infrastructure/rtmp"
)

func SendAchievementProgressCallbacks(connection *rtmp.Connection, update domainachievement.ProgressUpdate) {
	if connection == nil {
		return
	}
	for _, requirement := range update.Requirements {
		if requirement == nil {
			continue
		}
		done := 0
		if requirement.Done {
			done = 1
		}
		_ = connection.SendCallback("onUpdateAchieve", 2, requirement.RequirementID, map[string]interface{}{
			"done":     done,
			"progress": requirement.Progress,
		})
	}
	for _, completion := range update.Completed {
		_ = connection.SendCallback("onUpdateAchieve", 1, completion.AchievementID, map[string]interface{}{
			"done": completion.DoneAt,
		})
	}
}
