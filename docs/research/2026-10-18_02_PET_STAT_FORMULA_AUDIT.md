# Pet Stat Formula Audit

Date: 2026-10-18

## Scope and safety

This audit traced persisted Pet fields, `TBL_CREATURE`, Flash display formulas, server stat calculation, equipment, passive skills, DTOs, and combat projection. Database and runtime inspection was read-only. No Pet/player row, ActionScript, container, or deployed process was changed.

## Finding: the reported 70k aptitude is malformed quality data

Pet 168 is template 2290 `Bạch Hổ Cuồng Nộ`, level 80. Its persisted base quality is `grow_rate=15.000`; item 6974 and the current contract path define this award as quality `1.5`. The Flash panel computes displayed final Strength aptitude as:

`round((aptStrength + aptStrengthEvolution + aptStrengthEx) * (growRate + growRateAdd))`

With `aptStrength=4450`, `aptStrengthEx=0`, and quality near `15.8`, the panel therefore shows roughly 70k. This does not mean the Pet has 70k allocated Strength.

The row was created while the old award path passed encoded quality `15` directly as `15.0`. Fourteen local Pet rows have the same legacy signature (`15.000` or `19.000`). Current code must prevent new rows but this audit does not rewrite existing player data.

The catalog itself contains both encodings: `15/19/20/25` and a raw decimal `1.5`. Quality normalization therefore now divides only values `>=10`; raw decimal values remain unchanged.

## Current server formula

For creature-backed Pets, each raw attribute is:

`raw = template att + (level - 1) + allocated points + equipment attribute flat`

Each effective attribute is:

`effective = raw * (1 + persisted aptitude / 1000) * (growRate + growRateAdd)`

Base derived stats are truncated to integers by component:

| Stat | Formula |
|---|---|
| HP | `20 + effectiveStamina * 8.50` |
| MP | `5 + effectiveIntelligence * 4.07 + effectiveEnergy * 5.75` |
| SP | `150` |
| Physical attack | `effectiveStrength * 1.90 + effectiveAgility * 0.50` |
| Magic attack | `effectiveIntelligence * 2.00 + effectiveEnergy * 0.60` |
| Physical defense | `effectiveStamina * 2.57 + effectiveAgility * 0.85` |
| Magic defense | `effectiveIntelligence * 2.71 + effectiveEnergy * 4.60` |
| Hit | `100 + effectiveStrength * 0.08 + effectiveAgility * 0.05 + template propHit` |
| Dodge | `effectiveAgility * 0.11 + template propDodge` |
| Speed | `100 + effectiveAgility * 1.70 + template propSpeed` |
| Critical | `effectiveAgility * 0.10 + template propCritical` |

Template combo, counter, defy, reborn, and resistance-to-defy now seed their matching final properties. Equipment applies flat, float, then percent bonuses. Passive skill/talent/stone bonuses apply after equipment.

The coefficients above document current source behavior. The audit did not change the shared `BuildBaseStats` coefficients or the `1 + aptitude/1000` relationship because no authoritative legacy output snapshot is available to distinguish that formula from alternatives.

## Reported Pet breakdown

At the inspected state, the Pet had no allocated aptitude points: every `apt*Ex`, `max_attr_points`, and `distributed_attr_points` field was zero. `finalStrength=129` came from template Strength 50 plus 79 level increments.

The persisted property was explainable without rounding anomalies:

- base physical attack under malformed quality: about `24,492`
- legacy skill 60903 `Man Lực`: direct `+5,000`
- stored final physical attack: `29,492`

Skill 3755 `Kích Thích Tiềm Năng - Nhiệt Huyết` also has exact catalog text but zero `prop1..6`, so the generic passive loader previously ignored it. Its level-3 authority is quality-scaled `MP +510`, physical/magic attack `+290`, and critical `+1.25`. Exact-ID support for all twelve potential skills 3744–3755 is now included.

If only the malformed base quality were corrected from `15.0` to `1.5` while retaining the inspected `growRateAdd=0.83`, the panel aptitude would be about `10,369`, not 70k. Under current coefficients, base physical attack would be about `3,604`; Man Lực and level-3 Nhiệt Huyết would bring it to about `9,280`. This is an explanatory simulation only, not a player-data update or a claim that `growRateAdd=0.83` is canonical.

## Fixed defects

1. `Pet.RecalculateStats` now reads the five persisted aptitude columns. Evolution or authorized aptitude changes are no longer discarded in favor of stale template values.
2. Direct `TBL_CREATURE` hit, dodge, speed, critical, combo, counter, defy, reborn, and resistance-to-defy values are no longer reset to zero.
3. Potential passive skills 3744–3755 apply their exact quality-scaled catalog values.
4. Mixed raw/encoded Pet award quality is normalized without creating `15.0` or `0.15` Pets.

## Remaining gates

- Existing malformed Pet rows require explicit data-remediation permission and a separately reviewed mapping; none were changed.
- Shared coefficients and the additive aptitude term remain unchanged until an authoritative legacy server payload or formula source is available.
- Other passive buffs whose `prop1..6` are empty remain unsupported unless their descriptions provide complete, unambiguous timing and stacking semantics.
- The new binary has not been deployed or Flash-smoke-tested.
