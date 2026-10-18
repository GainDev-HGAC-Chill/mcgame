# Unified Stat Formula (2026-04-23)

Creature template `life` still acts as a final max-HP percentage multiplier in the unified stat pipeline: `10000 = 100%`, `3500 = 35%`, and missing/non-positive values fall back to `100%`. Pet combat HP is no longer scaled by any `life` value.

All creature-based pets loaded from `TBL_CREATURE` now use the flat stat-growth rule: keep `growthScale = growBase + growRateAdd` without multiplying by level, and add `level - 1` to each raw `attStrength`, `attAgility`, `attStamina`, `attIntelligence`, and `attEnergy` input before the shared stat builder runs. Non-creature/default pet templates still use the original `(growRate + growRateAdd) * level` scaling path.

All character, pet, creature, and boss stat calculations now share a single formula via `internal/domain/stats.BuildBaseStats(Profile)`.

**Shared Coefficients:**

- HP from stamina: 8.50
- MP from intelligence: 4.07; from energy/spirit: 5.75
- SP from stamina: 5.00
- Attack from strength: 1.90; from agility: 0.50
- MagicAttack from intelligence: 2.00; from energy/spirit: 0.60
- Defense from stamina: 2.57; from agility: 0.85
- MagicDefense from intelligence: 2.71; from energy/spirit: 4.60
- Hit from strength: 0.08; from agility: 0.05
- Dodge from agility: 0.11
- Speed from agility: 1.70
- Critical from agility: 0.10

**Characters:** `Character.RecalculateStats()` populates a `stats.Profile` with `Strength = att_* + distributed Apt*` and `AptStrength = ClassAptStrength`. Class `apt_*` values are applied as per-mille efficiency multipliers in the shared stat engine (`effective_attr = attr + attr * apt / 1000`), not `/100`. The `ClassApt*` fields stay separate so distributed `Apt*` points (from aptitude items) remain independent of class aptitude. `applyClassDefaultStats` also sets `ClassApt*` from the class template's `apt_*` fields.

**Pets:** `Pet.RecalculateStats()` reads template `att*` and direct combat props from `CreatureData`, but uses the five persisted Pet aptitude columns so evolution/authorized aptitude changes affect results. `Strength = template attStrength + (level-1) + AptStrengthEx + equipBonuses`. Persisted `AptStrength` uses the same per-mille shared-engine scaling (`/1000`). Creature-backed Pet `GrowthScale = GrowRate + GrowRateAdd`; non-creature fallback templates retain multiplication by level. `AptStrengthEx` and similar Ex fields remain additive raw attribute bonuses. Template hit/dodge/speed/critical flow through `Profile.Prop*`; combo/counter/defy/reborn/resiDefy seed their matching Pet final properties before equipment and passive bonuses.

Pet award quality is mixed-format in checked data: encoded `15/19/20/25` means `1.5/1.9/2.0/2.5`, while at least one row stores raw `1.5`. Normalize values `>=10` by `/10` and preserve smaller positive decimals. Existing rows created by the old raw-encoded path are not rewritten automatically.

**Creatures:** `CalculateCreatureStats()` sends `att_*` and `apt_*` directly into `stats.BuildBaseStats()`, and creature aptitude also uses per-mille scaling (`effective_attr = att + att * apt / 1000`) before growth/coefficients are applied. `GrowthScale = max(1, GrowBase)`. `SeedHP = normalizeCreatureLife(Life) + level*20`. `PropHit`, `PropSpeed`, `PropDodge`, and `PropCritical` stay as additive direct overrides via `Profile.Prop*`.

**Bosses:** Follow the creature flow via `CalculateCreatureStats`; no separate boss multiplier was introduced.

References:

- Plan: `docs/plans/2026-04-23_01_UNIFIED_STAT_FORMULA_PLAN.md`
- Research: `docs/research/2026-04-22_02_ATTRIBUTE_POINT_BASE_STAT_FORMULA_RESEARCH.md`
