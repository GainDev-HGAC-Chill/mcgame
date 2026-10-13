package scene

import (
	"encoding/json"
	"testing"

	"mcgame-server/internal/gamedata"
	"mcgame-server/internal/gamedata/models"

	"go.uber.org/zap"
)

func TestTeleportUnlinkedGate757UsesUniquePublicMapSafePoint(t *testing.T) {
	for _, tc := range []struct {
		name      string
		change    func(*models.MapTemplate, *models.SceneitemInstanceTemplate, *models.SceneitemTemplateTemplate)
		duplicate bool
		valid     bool
	}{
		{name: "gate 757", valid: true},
		{name: "unknown label", change: func(m *models.MapTemplate, g *models.SceneitemInstanceTemplate, t *models.SceneitemTemplateTemplate) {
			g.Name = "Unknown"
		}},
		{name: "ambiguous label", duplicate: true},
		{name: "instance map", change: func(m *models.MapTemplate, g *models.SceneitemInstanceTemplate, t *models.SceneitemTemplateTemplate) {
			m.CopyFlag = 1
		}},
		{name: "special map", change: func(m *models.MapTemplate, g *models.SceneitemInstanceTemplate, t *models.SceneitemTemplateTemplate) {
			m.Type = 3
		}},
		{name: "non portal", change: func(m *models.MapTemplate, g *models.SceneitemInstanceTemplate, t *models.SceneitemTemplateTemplate) {
			t.Type = 1
		}},
		{name: "bad safe point", change: func(m *models.MapTemplate, g *models.SceneitemInstanceTemplate, t *models.SceneitemTemplateTemplate) {
			m.SafeX = 3000
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := models.MapTemplate{ID: 1, Name: "Xuất Vân Thôn", Type: 2, SafeX: 1000, SafeY: 1000, Width: 2000, Height: 1500}
			g := models.SceneitemInstanceTemplate{ID: 757, Name: "Xuất Vân Thôn", Tid: 38, PosMapID: 3, PosX: 53, PosY: 1094}
			tpl := models.SceneitemTemplateTemplate{ID: 38, Type: 3}
			if tc.change != nil {
				tc.change(&m, &g, &tpl)
			}
			manager := gamedata.NewManager(nil, zap.NewNop())
			maps := []any{m}
			if tc.duplicate {
				copy := m
				copy.ID = 2
				maps = append(maps, copy)
			}
			loadTeleportRecords(t, manager, models.TableMap, maps...)
			loadTeleportRecords(t, manager, models.TableSceneitemTemplate, tpl)
			loadTeleportRecords(t, manager, models.TableSceneitemInstance, g)
			s := NewService(nil, nil, nil, nil, zap.NewNop())
			s.SetGameDataManager(manager)
			dest, err := s.GetTeleportDestination(757)
			if err != nil {
				t.Fatal(err)
			}
			if !tc.valid {
				if dest != nil {
					t.Fatalf("unsafe destination accepted: %#v", dest)
				}
				return
			}
			if dest == nil || *dest != (TeleportDestination{MapID: 1, X: 1000, Y: 1000}) {
				t.Fatalf("destination=%#v", dest)
			}
		})
	}
}

func TestTeleportExplicitLinkRemainsUnchanged(t *testing.T) {
	m := gamedata.NewManager(nil, zap.NewNop())
	loadTeleportRecords(t, m, models.TableSceneitemInstance,
		models.SceneitemInstanceTemplate{ID: 732, From: 733, PosMapID: 1},
		models.SceneitemInstanceTemplate{ID: 733, From: 732, PosMapID: 2, PosX: 120, PosY: 63})
	s := NewService(nil, nil, nil, nil, zap.NewNop())
	s.SetGameDataManager(m)
	dest, err := s.GetTeleportDestination(732)
	if err != nil || dest == nil || *dest != (TeleportDestination{MapID: 2, X: 220, Y: 163}) {
		t.Fatalf("explicit link changed: %#v %v", dest, err)
	}
}

func loadTeleportRecords(t *testing.T, m *gamedata.Manager, table string, values ...any) {
	t.Helper()
	rows := make([]json.RawMessage, 0, len(values))
	for _, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, data)
	}
	if err := m.GetCache().LoadTable(table, rows); err != nil {
		t.Fatal(err)
	}
}
