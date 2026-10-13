// Open-sourced by BaoLT

package monsterheart

import (
	"testing"

	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

func TestInitMonsterHeartData_ReturnsEmptyCollections(t *testing.T) {
	handler := NewHandler(nil, zap.NewNop())

	result, err := handler.InitMonsterHeartData(&rtmp.RPCContext{ConnID: 13, CharacterID: "5"}, nil)
	if err != nil {
		t.Fatalf("InitMonsterHeartData() error = %v", err)
	}

	payload, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("result type = %T, want map[string]interface{}", result)
	}
	data, ok := payload["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data type = %T, want map[string]interface{}", payload["data"])
	}
	levels, ok := data["heartBoxLev"].(map[string]interface{})
	if !ok {
		t.Fatalf("heartBoxLev type = %T, want map", data["heartBoxLev"])
	}
	for _, box := range []string{"1", "2", "3", "4", "5"} {
		if levels[box] != 1 {
			t.Fatalf("box %s level = %v, want baseline 1", box, levels[box])
		}
	}
	if _, ok := payload["bag"].(map[string]interface{}); !ok {
		t.Fatalf("bag type = %T, want map[string]interface{}", payload["bag"])
	}
}
