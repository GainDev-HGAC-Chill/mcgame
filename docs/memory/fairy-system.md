# Fairy (Tiểu Tinh Linh) System

## Authority and persistence

Fairy rosters are stored in `character_feature_states` under `feature_key='fairy'`; skin state uses `fairy_skin`. All whole-roster mutations use a per-character lock. A valid roster has at most one Fairy in guard state; bonus and combat readers fail closed if persisted legacy data contains multiple guards.

Fairy books are item type `521`. The book `propType` must equal the canonical level-1 skill ID, the Fairy growth level must meet `reqLevel`, and duplicate families are rejected. Upgrades follow the fixed 22-family, three-level chains and consume template `3583` in quantities 1 then 3. Skill presets require exactly three groups of five slots, only learned active skills, no duplicate family within a group, and `select` in `-1..2`.

## Character bonuses

Only the sole guarding Fairy contributes attributes. For each of `sta/ste/agi/inte/ener`, projected respectively to Strength/Stamina/Agility/Intelligence/Spirit:

```text
raw = base + (fairyLevel - 1) * (growthRate + 0.2 * growthLevel)
qualityPct = [20,25,30,35,40][floor((max(growthLevel,1)-1)/10)]
satiety = round(clamp(doh,0,10000) / 100)
satietyFactor = 1 - max(75 - satiety, 0) / 75
ownerBonus = round(raw * qualityPct * satietyFactor / 100)
```

The six passive families `5917..5934` apply their catalog buffs `2417..2434` through the normal `EquipmentStatBonuses` pipeline. Catalog ID, level, kind, buff ID, and percentage semantics must match exactly or the passive set is rejected. Guard/rest, growth, feeding, skill learning/upgrading, and deletion trigger an `onUPP` stat refresh.

## Battle rotation

PVE and PVP encounter creation snapshot the guarding Fairy's selected validated `c0..c4` preset. The rotation is battle-local and advances only on owner attack, skill, or auto-attack turns. Empty `-1` slots still consume their turn in the five-slot cycle.

Supported active families are catalog-validated before battle:

- `6230..6232`: magic attack +5/8/13%, 2 rounds.
- `6236..6238`: physical attack +5/8/13%, 2 rounds.
- `6239..6241`: magic defense +10/20/35%, 2 rounds.
- `6242..6244`: physical defense +10/20/35%, 2 rounds.
- `6260..6262`: speed +7/10/15%, 3 rounds.
- `6269..6271`: final incoming damage reduction 5/8/13%, 2 rounds.

All other active families remain explicit unsupported rotation entries. Activation emits a structured warning and applies no effect; do not infer timing, target, proc, stack, or damage formulas from prose.

## Operational status

Code and automated verification are complete. Runtime Flash verification remains pending explicit deploy/restart permission and must use only test characters/items.
