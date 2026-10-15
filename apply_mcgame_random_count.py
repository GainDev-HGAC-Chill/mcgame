#!/usr/bin/env python3
"""Add quantity ranges to MCGame box rewards, preserving DB integer count. Dry-run default."""
import pathlib,sys,datetime,shutil,re
ROOT=pathlib.Path(sys.argv[1] if len(sys.argv)>1 else '.').resolve()
APPLY='--apply' in sys.argv
changes={}
def edit(p,fn):
 f=ROOT/p
 if not f.exists(): raise RuntimeError('Missing '+str(f))
 old=f.read_text(encoding='utf-8-sig'); new=fn(old)
 if old!=new:changes[p]=new

def sub1(s,old,new):
 if new in s and old not in s:return s
 n=s.count(old)
 if n!=1:raise RuntimeError(f'Expected exactly 1 occurrence, found {n}: {old[:100]!r}')
 return s.replace(old,new,1)

ui='dashboard/src/app/dashboard/box-items/_components/box-item-award-workspace.tsx'
go='internal/application/item/effect_award.go'

def patch_ui(s):
 if 'countText?: string' in s:return s
 s=sub1(s,'    payloadText: string\n    guaranteed: boolean','    payloadText: string\n    countText?: string\n    guaranteed: boolean')
 s=sub1(s,'        count: row.count,','        count: row.count,\n        countText: (typeof payload.countMin === "number" && typeof payload.countMax === "number")\n            ? `${payload.countMin}-${payload.countMax}` : String(row.count),')
 s=sub1(s,'        count: item.count,','        count: item.count,\n        countText: String(item.count),')
 s=sub1(s,'                    payloadText,\n                    ...draft','                    payloadText,\n                    countText,\n                    ...draft')
 key='''                    if (guaranteed) {
                        normalizedPayload[BOX_ITEM_AWARD_GUARANTEED_KEY] = true'''
 val='''                    // Range is stored in payload; DB count stays an integer.
                    const countInput = (countText ?? String(draft.count)).trim()
                    if (!boxItemAwardUsesFixedCount(draft.type)) {
                        const match = /^(\\d+)(?:\\s*-\\s*(\\d+))?$/.exec(countInput)
                        if (!match) throw new Error(`Số lượng của ${awardName} không hợp lệ: ${countInput}`)
                        const min = Number(match[1])
                        const max = Number(match[2] ?? match[1])
                        if (!Number.isSafeInteger(min) || !Number.isSafeInteger(max) ||
                            min < 1 || max < min || max > 100000000) {
                            throw new Error(`Khoảng số lượng không hợp lệ cho ${awardName}.`)
                        }
                        if (min === max) {
                            delete normalizedPayload.countMin
                            delete normalizedPayload.countMax
                        } else {
                            normalizedPayload.countMin = min
                            normalizedPayload.countMax = max
                        }
                        draft.count = min
                    } else {
                        delete normalizedPayload.countMin
                        delete normalizedPayload.countMax
                    }

'''+key
 s=sub1(s,key,val)
 old='''                                                <Input
                                                    type="number"
                                                    min={1}
                                                    value={draft.count}
                                                    disabled={boxItemAwardUsesFixedCount(
                                                        draft.type
                                                    )}
                                                    onChange={event =>
                                                        updateDraft(
                                                            draft._key,
                                                            "count",
                                                            Math.max(
                                                                1,
                                                                Number(
                                                                    event.target
                                                                        .value
                                                                ) || 1
                                                            )
                                                        )
                                                    }
                                                />'''
 new='''                                                <Input
                                                    type="text"
                                                    inputMode="numeric"
                                                    placeholder="10 hoặc 1-100"
                                                    value={draft.countText ?? String(draft.count)}
                                                    disabled={boxItemAwardUsesFixedCount(draft.type)}
                                                    onChange={event => {
                                                        const value = event.target.value
                                                        setDrafts(prev => prev.map(d =>
                                                            d._key === draft._key ? { ...d, countText: value } : d
                                                        ))
                                                    }}
                                                />'''
 s=sub1(s,old,new)
 return s

def patch_go(s):
 if 'func resolveAwardQuantityRange(' in s:return s
 old='''		count := int(award.Count)
		if count <= 0 {
			count = 1
		}
'''
 new='''		count := int(award.Count)
		if count <= 0 {
			count = 1
		}
		// Roll once per box open, after choosing the reward.
		if !boxAwardFixedQuantity(int(award.Type)) {
			if minQty, maxQty, hasRange := resolveAwardQuantityRange(award.Payload); hasRange {
				if rng != nil {
					count = rng(minQty, maxQty)
				} else {
					count = minQty // defensive fallback if service RNG is unavailable
				}
			}
		}
'''
 s=sub1(s,old,new)
 marker='func (h *awardEffectHandler) resolveAwards('
 helper='''// Fixed-count unlocks are not stackable; do not randomize their quantity.
func boxAwardFixedQuantity(awardType int) bool {
    return awardType == 33 || awardType == 34 || awardType == 37 || awardType == 38
}

func resolveAwardQuantityRange(payload map[string]interface{}) (int, int, bool) {
    minQty, minOK := awardPayloadInt(payload, "countMin")
    maxQty, maxOK := awardPayloadInt(payload, "countMax")
    if !minOK || !maxOK || minQty < 1 || maxQty < minQty || maxQty > 100000000 {
        return 0, 0, false
    }
    return minQty, maxQty, true
}

'''
 s=sub1(s,marker,helper+marker)
 return s

edit(ui,patch_ui)
edit(go,patch_go)
print('Targets:',*(str(x) for x in changes),sep='\n - ')
if not APPLY:
 print('Dry run OK. Use --apply to write changes.')
else:
 stamp=datetime.datetime.now().strftime('%Y%m%d_%H%M%S')
 for p,c in changes.items():
  f=ROOT/p; shutil.copy2(f,f.with_name(f.name+'.backup_'+stamp));f.write_text(c,encoding='utf-8',newline='')
 print('Patched. Backup suffix .backup_'+stamp)
