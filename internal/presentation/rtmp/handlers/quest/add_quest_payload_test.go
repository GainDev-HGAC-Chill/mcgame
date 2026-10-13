package quest

import (
	"testing"

	domainquest "mcgame-server/internal/domain/quest"
)

func TestShopQuestProgressUsesSameAddEnvelopeAsQuestAcceptance(t *testing.T) {
	progress := &domainquest.QuestProgress{QuestID: 161}
	enriched := map[string]interface{}{"qid": "161", "state": 104, "data": map[string]interface{}{"id": 161, "name": "Existing quest"}, "require": []interface{}{}}
	payload := buildAddQuestPayload(progress, enriched, 1)
	if payload["qid"] != "161" || payload["st"] != 104 || payload["qn"] != 1 {
		t.Fatalf("missing add-quest metadata: %#v", payload)
	}
	nested, ok := payload["d"].(map[string]interface{})
	if !ok || nested["data"] == nil || nested["qid"] != "161" {
		t.Fatalf("missing enriched quest body: %#v", payload)
	}
	if _, exists := enriched["d"]; exists {
		t.Fatal("mutated service DTO or built recursive body")
	}
}
