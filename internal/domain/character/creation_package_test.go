package character

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestDecodeCreationDefaultsStrict(t *testing.T) {
	valid := `{"version":1,"level":90,"gold":999999,"goldBind":999999,"items":[],"pets":[]}`
	for _, raw := range []string{
		`null`, `{}`, strings.Replace(valid, `"level":90`, `"level":90.5`, 1),
		strings.Replace(valid, `"gold":999999`, `"gold":null`, 1),
		strings.Replace(valid, `"items":[]`, `"items":null`, 1),
		strings.Replace(valid, `"version":1`, `"version":2`, 1),
		strings.Replace(valid, `"pets":[]`, `"pets":[{"templateId":91,"quantity":1,"bound":true}]`, 1),
		strings.Replace(valid, `"items":[]`, `"items":[{"templateTableId":29,"templateId":1,"quantity":1,"bound":null,"color":0}]`, 1),
		strings.Replace(valid, `"pets":[]`, `"pets":[],"unknown":1`, 1), valid + `{}`,
	} {
		if _, err := DecodeNewCharacterDefaults([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid defaults: %s", raw)
		}
	}
	d, err := DecodeNewCharacterDefaults([]byte(valid))
	if err != nil || d.Level != 90 || d.Gold != 999999 || d.GoldBind != 999999 {
		t.Fatalf("legacy defaults: %+v %v", d, err)
	}
}

func TestCreationDefaultsBoundsAndImmutableHash(t *testing.T) {
	d := DefaultNewCharacterDefaults()
	d.Items = []StarterItem{{TemplateTableID: 29, TemplateID: 1, Quantity: 1, Bound: true}}
	d.Pets = []StarterPet{{TemplateID: 91, Quantity: 1, Growth: 1.9}}
	hash, err := d.SnapshotHash()
	if err != nil || len(hash) != 64 {
		t.Fatal(hash, err)
	}
	clone := d.Clone()
	d.Items[0].Quantity = 2
	d.Pets[0].Growth = 2
	cloneHash, _ := clone.SnapshotHash()
	newHash, _ := d.SnapshotHash()
	if cloneHash != hash || newHash == hash {
		t.Fatal("snapshot aliases editable defaults")
	}
	for _, mutate := range []func(*NewCharacterDefaults){
		func(d *NewCharacterDefaults) { d.Level = 161 },
		func(d *NewCharacterDefaults) { d.Gold = -1 },
		func(d *NewCharacterDefaults) { d.GoldBind = MaxStarterCurrency + 1 },
		func(d *NewCharacterDefaults) { d.Items[0].Quantity = 1000001 },
		func(d *NewCharacterDefaults) { d.Items[0].Color = 6 },
		func(d *NewCharacterDefaults) { d.Pets[0].Growth = math.NaN() },
		func(d *NewCharacterDefaults) { d.Pets[0].Growth = math.Inf(1) },
		func(d *NewCharacterDefaults) { d.Pets[0].Growth = 0 },
	} {
		invalid := clone.Clone()
		mutate(&invalid)
		if invalid.Validate() == nil {
			raw, _ := json.Marshal(invalid)
			t.Fatalf("accepted %s", raw)
		}
	}
}
