// Open-sourced by BaoLT

package combat

import (
	"testing"

	domainpet "mcgame-server/internal/domain/pet"
)

func TestSendTrainingPetStatRefreshesPushesEveryPet(t *testing.T) {
	recorder := &callbackRecorder{}
	first := &domainpet.Pet{ID: 11, Level: 80}
	second := &domainpet.Pet{ID: 12, Level: 80}
	first.RecalculateStats()
	second.RecalculateStats()

	sendTrainingPetStatRefreshes(recorder, []*domainpet.Pet{first, nil, second})

	if len(recorder.calls) != 2 {
		t.Fatalf("callback count=%d want 2", len(recorder.calls))
	}
	for index, call := range recorder.calls {
		if call.method != "onRefreshPetProp" {
			t.Fatalf("callback %d method=%q", index, call.method)
		}
		payload := call.args[0].(map[string]interface{})
		if payload["s"] == nil {
			t.Fatalf("callback %d payload=%+v", index, payload)
		}
	}
}
