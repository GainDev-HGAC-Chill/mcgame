// Pet active skill plans describe battle-local costs, targeting, and reusable effects.
package combat

type SkillPlan struct {
	Family        string
	Level         int
	Target        TargetType
	Area          AreaType
	TargetCount   int
	Effects       []SkillEffect
	Costs         []SkillCost
	CostsOnTarget bool
	RequireHPGT   float64
	RequireHPMax  float64
	Sacrifice     bool
	RestTurns     int
	OnKill        *SkillOnKillFollowUp
}

type SkillOnKillFollowUp struct {
	PassiveTargetCounts map[int]int
	TemplateIDs         map[int]int
	Target              TargetType
}

type SkillCost struct {
	Resource ResourceType
	Percent  float64
	Fixed    int
}

type ResourceType int

const (
	ResourceHP ResourceType = iota + 1
	ResourceMP
)

type EffectKind int

const (
	EffectPhysicalDamage EffectKind = iota + 1
	EffectMagicDamage
	EffectFixedCurrentHPDamage
	EffectFixedMaxHPDamage
	EffectHeal
	EffectRestoreHP
	EffectRestoreMP
	EffectStatus
	EffectRevive
	EffectRemoveOne
	EffectRemoveAll
	EffectTransfer
)

type EffectPolarity int

const (
	PolarityNeutral EffectPolarity = iota
	PolarityBeneficial
	PolarityHarmful
)

type CombatStat int

const (
	StatAttack CombatStat = iota + 1
	StatMagicAttack
	StatDefense
	StatMagicDefense
	StatSpeed
	StatDodge
	StatAccuracy
	StatCombo
	StatCritical
	StatReducePhysical
	StatReduceMagic
	StatIncomingDamage
	StatEnergy
	StatDebuffResistance
)

type StatModifier struct {
	Stat    CombatStat
	Percent float64
	Flat    int
}

type PeriodicEffect struct {
	Damage      bool
	Magic       bool
	Resource    ResourceType
	Coefficient float64
	Fixed       int
	Basis       CombatStat
}

type ShieldEffect struct {
	MagicOnly bool
	Pool      int
	Hits      int
}

type ControlEffect struct {
	Stun          bool
	Sleep         bool
	Confusion     bool
	Invulnerable  bool
	CannotAct     bool
	TauntSourceID string
}

type SkillEffect struct {
	Kind                    EffectKind
	Coefficient             float64
	Fixed                   int
	Chance                  float64
	Duration                int
	Polarity                EffectPolarity
	Removable               bool
	Family                  string
	Modifiers               []StatModifier
	Periodic                *PeriodicEffect
	ShieldPercentMaxMP      float64
	ShieldMagicOnly         bool
	ShieldHits              int
	Control                 ControlEffect
	BonusCurrentHPChance    float64
	BonusCurrentHPRatio     float64
	TransferCount           int
	RequiresPreviousSuccess bool
	ResourceBasis           ResourceType
	ApplyToTarget           bool
}
