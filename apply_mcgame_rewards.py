#!/usr/bin/env python3
"""Conservative source patch for MCGame reward options.
Run: py apply_mcgame_rewards.py I:\\mcgame --apply
Without --apply only validates patterns. Backups are made before editing.
"""
import sys, pathlib, re, shutil, datetime

ROOT=pathlib.Path(sys.argv[1] if len(sys.argv)>1 else '.').resolve()
APPLY='--apply' in sys.argv
changes={}

def load(p):
    f=ROOT/p
    if not f.is_file(): raise RuntimeError(f'Missing {p}')
    return f.read_text(encoding='utf-8-sig')
def replace(p,old,new, count=1):
    s=changes.get(p,load(p))
    n=s.count(old)
    if new in s and n==0:return
    if n!=count:raise RuntimeError(f'{p}: expected {count} matches, got {n}: {old[:75]!r}')
    changes[p]=s.replace(old,new,count)
def insert_once(p,needle,block):
    s=changes.get(p,load(p))
    if block in s:return
    replace(p,needle,block+needle)

lib='dashboard/src/lib/box-item-awards.ts'
ui='dashboard/src/app/dashboard/box-items/_components/box-item-award-workspace.tsx'
service='internal/application/item/service.go'
award='internal/application/item/effect_award.go'
main='cmd/gameserver/main.go'
fairy='internal/application/fairy/adopt.go'

replace(lib,'12 | 19 | 29 | 30 | 31 | 32 | 33 | 34 | 35','12 | 19 | 29 | 30 | 31 | 32 | 33 | 34 | 35 | 37 | 38')
insert_once(lib,'    { value: "35", label: "Điểm / tiền tệ game" },', '    { value: "37", label: "Skin thú cưỡi" },\n    { value: "38", label: "Tinh linh" },\n')
# Basic IDs deliberately remain 0/1/2. Duplicated UI options carry payload bound=true.
insert_once(lib,'export const BOX_ITEM_GAME_CURRENCY_OPTIONS = [', '''export const BOX_ITEM_BOUND_CURRENCY_OPTIONS = [
    { value: "0", label: "Bạc khóa", payload: { bound: true } },
    { value: "1", label: "Vàng khóa", payload: { bound: true } },
] as const

export const BOX_ITEM_MOUNT_OPTIONS = [
    { value: "1", label: "Bích Hổ" },
    { value: "2", label: "U Linh Hổ" },
    { value: "3", label: "Hổ Vàng Răng Kiếm" },
    { value: "6", label: "Bông Gòn" },
    { value: "7", label: "Viêm Lang" },
    { value: "8", label: "Cửu Vĩ Hồ" },
    { value: "9", label: "Hồng Liên Bôn Lôi" },
    { value: "10", label: "Hàn Sương Thiểm Điện" },
    { value: "11", label: "Viễn Cổ Chiến Hùng" },
    { value: "12", label: "Hàn Băng Nha" },
    { value: "13", label: "Thương Lam Nha" },
    { value: "14", label: "Hấp Huyết Nha" },
    { value: "15", label: "Cơ Giáp Ma Long" },
    { value: "16", label: "Tê Giác Thiết Giáp" },
    { value: "17", label: "Chiến Hoàng Bá Thiên Hổ" },
    { value: "18", label: "Kim Khôi Chiến Tướng" },
    { value: "19", label: "Niên Thú" },
    { value: "20", label: "Burin" },
] as const

export const BOX_ITEM_FAIRY_OPTIONS = [
    { value: "1", label: "Thiên Sứ Poli" },
    { value: "2", label: "Rocky" },
    { value: "3", label: "Băng Nữ Koura" },
    { value: "4", label: "Puca" },
    { value: "5", label: "Rồng Xanh Abu" },
    { value: "6", label: "Yêu Tinh Nicole" },
    { value: "7", label: "Tuần Lộc Miya" },
    { value: "8", label: "Thiên Sứ Bọt" },
    { value: "9", label: "Minions" },
    { value: "10", label: "Casper" },
    { value: "11", label: "Thần Chết" },
    { value: "12", label: "Hàn Sương Băng Long" },
    { value: "13", label: "U Độc Minh Phượng" },
    { value: "14", label: "Thánh Quang Thiên Sứ" },
    { value: "15", label: "Bông Ú Ú" },
    { value: "16", label: "Thỏ Nắng Nịnh" },
    { value: "17", label: "Cặp Ma Lực" },
] as const

''')
replace(lib,'        case 35:\n            return numeric','        case 35:\n        case 37:\n        case 38:\n            return numeric')
replace(lib,'        case 35:\n            return "Điểm game"','        case 35:\n            return "Điểm game"\n        case 37:\n            return "Skin thú cưỡi"\n        case 38:\n            return "Tinh linh"')
replace(lib,'    return normalized === 33 || normalized === 34','    return normalized === 33 || normalized === 34 || normalized === 37 || normalized === 38')
replace(lib,'        case 35:\n            return boxItemGameCurrencyName(awardId)','        case 35:\n            return boxItemGameCurrencyName(awardId)\n        case 37:\n            return BOX_ITEM_MOUNT_OPTIONS.find(x => Number(x.value) === awardId)?.label || `Skin thú cưỡi #${awardId}`\n        case 38:\n            return BOX_ITEM_FAIRY_OPTIONS.find(x => Number(x.value) === awardId)?.label || `Tinh linh #${awardId}`')

insert_once(ui,'    BOX_ITEM_CURRENCY_OPTIONS,','    BOX_ITEM_BOUND_CURRENCY_OPTIONS,\n    BOX_ITEM_MOUNT_OPTIONS,\n    BOX_ITEM_FAIRY_OPTIONS,\n')
replace(ui,'''    if (boxItemAwardUsesCurrencyOptions(type)) {
        return BOX_ITEM_CURRENCY_OPTIONS.map(option => ({''','''    if (boxItemAwardUsesCurrencyOptions(type)) {
        const basicItems = BOX_ITEM_CURRENCY_OPTIONS.map(option => ({''')
replace(ui,'''        })).filter(
            item =>
                !normalizedSearch ||
                item.name.toLowerCase().includes(normalizedSearch)
        )
    }

    if (boxItemAwardUsesGameCurrencyOptions(type))''','''        }))
        const boundItems = BOX_ITEM_BOUND_CURRENCY_OPTIONS.map(option => ({
            key: `${type}:bound:${option.value}`,
            awardId: Number(option.value), type, name: option.label,
            subtitle: `Thêm phần thưởng ${option.label.toLowerCase()}`,
            iconId: 0, iconDataUrl: "", count: 1, rate: 100,
            quality: 0, preNameType: 0,
            payloadText: JSON.stringify(option.payload),
        }))
        return [...basicItems, ...boundItems].filter(item =>
            !normalizedSearch || item.name.toLowerCase().includes(normalizedSearch)
        )
    }

    if (type === 37 || type === 38) {
        const options = type === 37 ? BOX_ITEM_MOUNT_OPTIONS : BOX_ITEM_FAIRY_OPTIONS
        return options.map(option => ({
            key: `${type}:${option.value}`, awardId: Number(option.value),
            type, name: option.label, subtitle: type === 37 ? 'Mở khóa vĩnh viễn' : 'Nhận tinh linh',
            iconId: 0, iconDataUrl: "", count: 1, rate: 100,
            quality: 0, preNameType: 0, payloadText: "",
        })).filter(item => !normalizedSearch ||
            item.name.toLowerCase().includes(normalizedSearch) ||
            String(item.awardId).includes(normalizedSearch))
    }

    if (boxItemAwardUsesGameCurrencyOptions(type))''')
# Add static types to effect condition (format independent)
s=changes.get(ui,load(ui))
needle='boxItemAwardUsesGameCurrencyOptions(selectedAvailableType) ||'
if needle not in s:raise RuntimeError('Missing static game currency condition; first fix the existing currency tab')
replace(ui,needle,needle+'\n            selectedAvailableType === 37 ||\n            selectedAvailableType === 38 ||')

# Minimal ports to avoid Go import cycles.
insert_once(service,'type Service struct {','''type MountRewardGranter interface {
    GrantDress(ctx context.Context, charID int64, dressID int, expiryMs int64) error
    DressResCode(dressID int) int64
}
type FairyRewardGranter interface {
    GrantReward(ctx context.Context, charID int64, tid int) (map[string]interface{}, error)
}

''')
insert_once(service,'\tawardHandler        *awardEffectHandler','\tmountRewardGranter  MountRewardGranter\n\tfairyRewardGranter  FairyRewardGranter\n')
insert_once(service,'func (s *Service) SetBloodBagService','''func (s *Service) SetMountRewardGranter(v MountRewardGranter) { s.mountRewardGranter = v }
func (s *Service) SetFairyRewardGranter(v FairyRewardGranter) { s.fairyRewardGranter = v }

''')
insert_once(fairy,'func (s *Service) Adopt(','''// GrantReward is the award-only path; it validates a real template before granting.
func (s *Service) GrantReward(ctx context.Context, charID int64, tid int) (map[string]interface{}, error) {
    if s == nil || s.gameData == nil || s.gameData.GetFairyTempalte(tid) == nil {
        return nil, ErrFairyMissing
    }
    return s.Adopt(ctx, charID, tid)
}

''')
insert_once(main,'\tfairyHandler := fairyhandler.NewHandler(fairyService, log)','\titemService.SetFairyRewardGranter(fairyService)\n')
insert_once(main,'\tmountHandler := mounthandler.NewHandler(mountService, log)','\titemService.SetMountRewardGranter(mountService)\n')

# Bound currency: payload data is already carried in resolvedItemAward.
replace(award,'''			newTotal, applied := char.AddBasicCurrency(award.AwardID, int64(award.Count))''','''            bound, _ := award.Payload["bound"].(bool)
            if bound && (award.AwardID == 0 || award.AwardID == 1) {
                var newTotal int64
                var key, label string
                if award.AwardID == 0 {
                    char.MoneyBind += int64(award.Count)
                    newTotal, key, label = char.MoneyBind, "moneyBind", "Bạc khóa"
                } else {
                    char.GoldBind += int64(award.Count)
                    newTotal, key, label = char.GoldBind, "goldBind", "Vàng khóa"
                }
                result.GrantedCurrencies = append(result.GrantedCurrencies, GrantedCurrencyResult{
                    ClientKey: key, Delta: int64(award.Count), Total: float64(newTotal), Label: label,
                })
                result.RefreshCharacterView = true
                break
            }
            newTotal, applied := char.AddBasicCurrency(award.AwardID, int64(award.Count))''')
insert_once(award,'\t\tcase awardTypeMedal:', '''\t\tcase 37:
            if award.Count != 1 || h.itemService == nil || h.itemService.mountRewardGranter == nil {
                return nil, fmt.Errorf("mount reward requires count=1 and configured mount service")
            }
            if h.itemService.mountRewardGranter.DressResCode(award.AwardID) <= 0 {
                return nil, fmt.Errorf("invalid mount dress ID %d", award.AwardID)
            }
            if err := h.itemService.mountRewardGranter.GrantDress(ctx, char.ID, award.AwardID, 1); err != nil {
                return nil, err
            }
            result.RefreshCharacterView = true
            result.Message = fmt.Sprintf("Đã mở khóa skin thú cưỡi %d. Đăng nhập lại để đồng bộ danh sách.", award.AwardID)
        case 38:
            if award.Count != 1 || h.itemService == nil || h.itemService.fairyRewardGranter == nil {
                return nil, fmt.Errorf("fairy reward requires count=1 and configured fairy service")
            }
            if _, err := h.itemService.fairyRewardGranter.GrantReward(ctx, char.ID, award.AwardID); err != nil {
                return nil, err
            }
            result.RefreshCharacterView = true
            result.Message = fmt.Sprintf("Đã nhận tinh linh %d. Đăng nhập lại để đồng bộ danh sách.", award.AwardID)
''')
replace(award,'case 30, 31, 32, 33, 34, 35, awardTypeMedal:', 'case 30, 31, 32, 33, 34, 35, 37, 38, awardTypeMedal:')

# Output patch as backups + writing, with validation upfront.
for name,content in changes.items():
    if '\ufffd' in content: raise RuntimeError(f'Unexpected replacement characters in {name}')
print('Patch targets:')
for name in changes:print(' -',name)
if not APPLY:
    print('Dry run OK. Re-run with --apply to write changes.')
else:
    stamp=datetime.datetime.now().strftime('%Y%m%d_%H%M%S')
    for name,new in changes.items():
        f=ROOT/name
        shutil.copy2(f,f.with_name(f.name+'.backup_'+stamp))
        f.write_text(new,encoding='utf-8',newline='')
    print('Patched. Backups: *.backup_'+stamp)
    print('NEXT: gofmt -w internal/application/item/service.go internal/application/item/effect_award.go internal/application/fairy/adopt.go cmd/gameserver/main.go')
    print('NEXT: go test ./internal/application/item/... ./internal/application/fairy/... ./internal/application/mount/... ./cmd/gameserver/...')
    print('NEXT: cd dashboard; npm run typecheck')
