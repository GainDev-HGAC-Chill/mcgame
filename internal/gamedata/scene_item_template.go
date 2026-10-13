// Typed scene-item template access supports portal validation without instance/template confusion.
package gamedata

import "mcgame-server/internal/gamedata/models"

func (c *Cache) GetSceneItemTemplate(id int) *models.SceneitemTemplateTemplate {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sceneItemTemplate[id]
}

func (m *Manager) GetSceneItemTemplate(id int) *models.SceneitemTemplateTemplate {
	return m.cache.GetSceneItemTemplate(id)
}
