# Pet Skill Book Catalog Audit

**Date:** 2026-10-21  
**Scope:** every checked-in `TBL_ITEM_TEMPLATE` row with `type=506,useType=2`, plus the reported `Cận Vệ Thánh Nữ` passive.

## Result

- Total book-like item templates: **199**.
- Deterministic exact-name resolution in canonical Pet environments `useEnv=0..2`: **154**.
- Ambiguous lowest-level matches: **0**.
- Missing direct `skillId` references: **0**; all 199 rows currently have `skillId=0` and rely on catalog resolution.
- Intentionally unresolved/non-exact rows: **45**.

The five reported books all resolve uniquely. Their failure happened after resolution: the server incorrectly treated the first `creKind` segment as a required Pet `classIds`. Runtime evidence from the reported Pet shows that `creKind` is `family|minimumQLevel`; only the second segment gates book learning.

| Book item | Skill level-1 | Family metadata |
|---:|---:|---:|
| 1382 Báo Chi Tấn Tiệp | 2741 | 2 |
| 1377 Cổ Long Chi Hồn | 2820 | 6 |
| 1378 Ngạnh Xác Hóa | 2696 | 4 |
| 1379 Vẻ Ngoài Yếu Ớt | 2841 | 5 |
| 1374 Thụ Bì Ngoại Sáo | 2825 | 3 |

The fix removes the family/class gate and retains minimum quality validation from the second segment. Read-only runtime evidence confirms Pet 278 is Thỏ Baby `classIds=10,qLevel=3`, while it already owns Trí Tuệ Thiên Nhiên skills carrying family segment `1`; the failed Cổ Long and Ngạnh Xác books carry segments `6` and `4` and therefore cannot be interpreted as class restrictions.

## The 45 fail-closed rows

These rows are not silently mapped because the checked-in item/skill catalog does not provide one unique canonical skill ID.

### Random or container books (23)

- 1370 `Bí Kíp Kỹ Năng Bị Động Của Ma Thú Sơ Cấp`.
- Seven Sơ/Trung/Cao `Yếu Quyết` groups for Ác Ma, BOSS, Dã Thú, Long, Máy, Người and Thực Vật: item IDs 804/873/880, 803/874/881, 435/869/876, 437/871/878, 434/868/875, 436/870/877 and 802/872/879.
- 889 `Yếu Quyết Ma Thú Đặc Cấp Hệ Long`.

Their names describe a pool/container rather than one exact learned skill. Resolving one arbitrary member would consume the item incorrectly.

### Test or separate subsystem rows (7)

- 2248 `Năng Lượng Hộ Giáp fortest` and 2249 `Kim Cang Hộ Thể fortest`.
- 4775–4779 `Thánh Thú Hủy Diệt` Nộ/Tiếu/Cương/Linh/Bích are guardian books, not proven canonical `skill1..skill15` books.

### Exact names whose only skill rows are special `useEnv=-1` (3)

- 2539 `Thần Thánh Phủ Định Sơ Cấp`.
- 2701 `Thần Thánh Phủ Định Trung Cấp`.
- 2709 `Thần Thánh Phủ Định Cao Cấp`.

The current resolver deliberately excludes `useEnv=-1`. These cannot be enabled globally without also admitting unrelated creature/evolution skills.

### Legacy/localized names without a unique exact skill row (12)

- 1646 `Chí Mệnh Xúc Giác`
- 1148 `Đột Thứ Xung Phong`
- 1322 `Hào Quang Thần Tình Yêu`
- 1130 `Long Nhan Huyết Mạch` (`description=Undefined`)
- 1321 `Mũi Tên Của Thần Cupid`
- 1552 `Nhóm Liều Lĩnh`
- 1558 `Nhóm Trào Phúng`
- 1176 `Sách Kỹ Năng “Cung Chúc Tân Hỷ“`
- 1293 `Tiếng Nói Ác Ma`
- 2247 `Toàn Năng Hộ Thuẫn`
- 1145 `Tường Độc Vô Hình`
- 1320 `爱情诡计`

Several have plausible translated or renamed candidates, but item descriptions alone do not prove a complete item→skill ID mapping. They remain fail-closed rather than learning the wrong family. An alias can be added later only with Flash/runtime or another authoritative mapping.

## Cận Vệ Thánh Nữ

- Skills: 3848/3849/3850.
- Buffs: 2015/2014/2013.
- Seed skill text says the shield activates before every owner turn.
- Seed buff rows specify three blocked hits and pools equal to 50/75/100% caster MaxMP.
- Runtime now replaces the same-family shield before each turn, preserving one instance, and emits `buffAdd` with the exact level buff ID so Flash can render icon `4140200100250`.
- The implementation follows checked-in seed's every-turn semantics, not the community forum's conflicting “first three rounds” wording.

## Verification policy

- No real book, Pet or player item is consumed.
- Catalog tests read the checked-in JSON and pin `199 = 154 resolved + 45 fail-closed`.
- End-to-end mocked learn tests use a `classIds=10,qLevel=3` Pet and cover all five reported families plus the exact Cổ Long/Ngạnh Xác trung cấp items observed in runtime logs.
- Combat tests cover shield ownership, exact icon ID, three hits, MaxMP pool and per-turn refresh.
