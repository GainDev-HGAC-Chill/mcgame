from pathlib import Path
import sys, shutil

root=Path(sys.argv[1] if len(sys.argv)>1 else r'I:\mcgame')
changes={}

def patch(rel, old, new):
    p=root/rel
    original=changes.get(p, p.read_text(encoding='utf-8-sig'))
    if old not in original:
        if new in original:
            print('Already patched:',rel)
            return
        raise RuntimeError(f'Expected code not found in {rel}: {old[:90]!r}')
    if original.count(old)!=1:
        raise RuntimeError(f'Ambiguous patch in {rel}: {original.count(old)} matches')
    changes[p]=original.replace(old,new,1)

plan='internal/domain/combat/skill_plan.go'
patch(plan,'\tRepeatChance  float64 // chance for exactly one extra cast, paid once', '\tRepeatChance  float64 // chance for exactly one extra cast, paid once\n\tChainChance   float64 // chance after every hit to continue, geometric chain')

adapter='internal/application/combat/pet_active_skill_templates.go'
anchor='var petActiveSkillTemplateCatalog = map[int]domaincombat.SkillPlan{'
addition='''var petActiveSkillTemplateCatalog = map[int]domaincombat.SkillPlan{
    // Tho Baby: each landed magical pulse deals exactly 1 HP damage;
    // after each pulse another one occurs with 30% chance.
    5363: rabbitHeartPlan(1),
    5362: rabbitHeartPlan(2),
    5361: rabbitHeartPlan(3),
    // Tho Baby lucky shield. Data buff ID is level-specific.
    5366: rabbitLuckyShieldPlan(1, .30, 1934),
    5365: rabbitLuckyShieldPlan(2, .40, 1933),
    5364: rabbitLuckyShieldPlan(3, .60, 1932),'''
patch(adapter,anchor,addition)
append='''

func rabbitHeartPlan(level int) domaincombat.SkillPlan {
    return domaincombat.SkillPlan{Family: "Nhịp Đập Trái Tim", Level: level,
        Target: domaincombat.TargetEnemy, TargetCount: 1,
        ChainChance: .30,
        Effects: []domaincombat.SkillEffect{{Kind: domaincombat.EffectMagicDamage, Coefficient: 1}},
    }
}

func rabbitLuckyShieldPlan(level int, ratio float64, buffID int) domaincombat.SkillPlan {
    return domaincombat.SkillPlan{Family: "Hào Quang May Mắn", Level: level,
        Target: domaincombat.TargetTeam, TargetCount: 2,
        Effects: []domaincombat.SkillEffect{{
            Kind: domaincombat.EffectStatus, Family: "Hào Quang May Mắn",
            Duration: 4, Polarity: domaincombat.PolarityBeneficial,
            Removable: true, BuffID: buffID,
            ShieldPercentMaxMP: ratio, ShieldHits: 3,
        }},
    }
}
'''
p=root/adapter
changes[p]=changes[p]+append

execfile='internal/domain/combat/skill_executor.go'
old='''\tactions := make([]*BattleAction, 0, len(targets)*2+1)
\tkilledTarget := false'''
new='''\tif plan.ChainChance > 0 && isRabbitHeartSkill(cmd.SkillID) {
        return tp.executeRabbitHeartChain(actor, cmd, info, targets[0], plan.ChainChance)
    }
\tactions := make([]*BattleAction, 0, len(targets)*2+1)
\tkilledTarget := false'''
patch(execfile,old,new)
extra='''

func isRabbitHeartSkill(id int) bool {
    return id == 5363 || id == 5362 || id == 5361
}

// Each hit independently rolls for another hit; there is no gameplay combo cap.
// A successful strike is resolved as magical damage, including shields/resistance.
func (tp *TurnProcessor) executeRabbitHeartChain(actor *Participant, cmd *BattleCommand, info *SkillInfo, target *Participant, chance float64) []*BattleAction {
    actions := make([]*BattleAction, 0, 8)
    for target != nil {
        if !target.IsAlive {
            target = SelectRandomTarget(tp.battle, getOppositeSide(actor.Side))
            if target == nil { break }
        }
        before := target.CurrentHP
        dealt := tp.applyDamage(target, 1, true)
        action := tp.damageAction(actor, target, cmd.SkillID, info.Template, dealt, false, false, before)
        actions = tp.appendPlannedDamagePlayback(actions, actor, target, action)
        if dealt > 0 {
            actions = append(actions, tp.petReflectActions(actor, target, dealt)...)
        }
        if tp.rng.Float64() >= chance { break }
    }
    if shouldTurnBackAfterStationarySkill(info.Template, false) {
        actions = append(actions, buildTurnBackAction(tp.battle.CurrentRound, actor))
    }
    return actions
}
'''
p=root/execfile
changes[p]=changes[p]+extra

# Perform all checks before writing anything.
for p,content in changes.items():
    if not p.is_file(): raise RuntimeError(f'Missing: {p}')
print('PATCH PLAN:')
for p in changes: print(' -',p)
if '--apply' not in sys.argv:
    print('DRY RUN only. Add --apply to modify files.')
else:
    for p,content in changes.items():
        backup=p.with_suffix(p.suffix+'.rabbit.bak')
        if not backup.exists(): shutil.copy2(p,backup)
        p.write_text(content,encoding='utf-8')
    print('Applied. Backups *.rabbit.bak. Next: gofmt -w and go test.')
