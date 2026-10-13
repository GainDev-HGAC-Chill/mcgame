package character

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"

	"mcgame-server/internal/domain/item"
	"mcgame-server/internal/domain/pet"
)

const NewCharacterDefaultsKey = "new_character_defaults"
const CreationPackageFeatureKey = "creation_package"
const MaxStarterCurrency = int64(9007199254740991)
const DefaultCreationLevel = 90
const DefaultCreationGold = int64(999999)

type NewCharacterDefaults struct {
	Version  int           `json:"version"`
	Level    int           `json:"level"`
	Gold     int64         `json:"gold"`
	GoldBind int64         `json:"goldBind"`
	Items    []StarterItem `json:"items"`
	Pets     []StarterPet  `json:"pets"`
}

type StarterItem struct {
	TemplateTableID int  `json:"templateTableId"`
	TemplateID      int  `json:"templateId"`
	Quantity        int  `json:"quantity"`
	Bound           bool `json:"bound"`
	Color           int  `json:"color"`
}

type StarterPet struct {
	TemplateID int     `json:"templateId"`
	Quantity   int     `json:"quantity"`
	Bound      bool    `json:"bound"`
	Growth     float64 `json:"growth"`
}

type CreationPackage struct {
	Defaults NewCharacterDefaults `json:"defaults"`
	Hash     string               `json:"hash"`
	Items    []*item.Item         `json:"items"`
	Pets     []*pet.Pet           `json:"pets"`
}

func DefaultNewCharacterDefaults() NewCharacterDefaults {
	return NewCharacterDefaults{Version: 1, Level: DefaultCreationLevel, Gold: DefaultCreationGold, GoldBind: DefaultCreationGold, Items: []StarterItem{}, Pets: []StarterPet{}}
}

func DecodeNewCharacterDefaults(raw []byte) (NewCharacterDefaults, error) {
	var value NewCharacterDefaults
	if err := validateDefaultsFields(raw); err != nil {
		return value, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("invalid new character defaults: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return value, fmt.Errorf("new character defaults must contain one JSON object")
	}
	return value, value.Validate()
}

func requireCreationFields(raw []byte, fields ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, fmt.Errorf("creation defaults require an object")
	}
	for _, key := range fields {
		value, exists := object[key]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("creation defaults field %s is required and cannot be null", key)
		}
	}
	return object, nil
}

func validateDefaultsFields(raw []byte) error {
	object, err := requireCreationFields(raw, "version", "level", "gold", "goldBind", "items", "pets")
	if err != nil {
		return err
	}
	for _, field := range []string{"items", "pets"} {
		var entries []json.RawMessage
		if err := json.Unmarshal(object[field], &entries); err != nil {
			return fmt.Errorf("creation defaults %s must be an array", field)
		}
		for _, entry := range entries {
			keys := []string{"templateId", "quantity", "bound", "growth"}
			if field == "items" {
				keys = []string{"templateTableId", "templateId", "quantity", "bound", "color"}
			}
			if _, err := requireCreationFields(entry, keys...); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d NewCharacterDefaults) Validate() error {
	if d.Version != 1 || d.Level < 1 || d.Level > MaxLevel {
		return fmt.Errorf("starter version must be 1 and level must be 1..%d", MaxLevel)
	}
	if d.Gold < 0 || d.Gold > MaxStarterCurrency || d.GoldBind < 0 || d.GoldBind > MaxStarterCurrency {
		return fmt.Errorf("starter currencies must be 0..%d", MaxStarterCurrency)
	}
	if len(d.Items) > 90 || len(d.Pets) > 6 {
		return fmt.Errorf("too many starter reward entries")
	}
	for i, entry := range d.Items {
		if (entry.TemplateTableID != 19 && entry.TemplateTableID != 29) || entry.TemplateID <= 0 || entry.TemplateID > math.MaxInt32 || entry.Quantity < 1 || entry.Quantity > 1000000 || entry.Color < 0 || entry.Color > 5 {
			return fmt.Errorf("invalid starter item entry %d", i+1)
		}
	}
	for i, entry := range d.Pets {
		if entry.TemplateID <= 0 || entry.TemplateID > math.MaxInt32 || entry.Quantity < 1 || entry.Quantity > 6 || math.IsNaN(entry.Growth) || math.IsInf(entry.Growth, 0) || entry.Growth <= 0 || entry.Growth > 10 {
			return fmt.Errorf("invalid starter pet entry %d: growth must be >0 and <=10", i+1)
		}
	}
	return nil
}

func (d NewCharacterDefaults) Clone() NewCharacterDefaults {
	d.Items = append([]StarterItem{}, d.Items...)
	d.Pets = append([]StarterPet{}, d.Pets...)
	return d
}

func (d NewCharacterDefaults) SnapshotHash() (string, error) {
	raw, err := json.Marshal(d.Clone())
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:]), nil
}
