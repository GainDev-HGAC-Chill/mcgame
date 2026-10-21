# Pet Skill Forum Gap Research

**Date:** 2026-10-21  
**Source requested by user:** <https://kinggame.forumtwilight.com/t88714-tong-hop-danh-sach-tat-ca-skill-pet-vpt>

## Scope and authority

- The KingGame post is a 2012 community compilation. It is useful for discovering omitted families and filling clearly missing tooltip values, but it contains incomplete levels, uncertain numbers, and contradictions with the shipped catalog.
- The checked-in `TBL_SKILL` and `TBL_BUFF` exports identify exact IDs, player-visible descriptions, targets, costs, rates, rounds, and stat properties.
- Decompiled Flash is authoritative only for the wire/playback contract. It sends one selected `tid` with skill ID/level; the server expands area targets and executes every hit, proc, status, resource change, and passive trigger.
- Existing user-approved mechanics in `2026-10-14_02_PET_ACTIVE_SKILL_AUDIT.md` remain authoritative unless this addendum explicitly identifies a corroborated defect.
- No ActionScript, database, player data, or runtime deployment was changed during this research.

## Flash contract findings

- `docs/client/logic/Battle.as:27-42` defines target types and vertical/horizontal/cross/random area codes.
- `docs/client/view/view/compGameStage/BattleCreatureView.as:627-725` uses `targetType` only to validate the clicked unit. Type 11 permits any clicked unit; it does not automatically select the whole battlefield.
- `docs/client/logic/Battle.as:1105-1131` sends only `{tid, action, id, level}`. There is no expanded target list or hit count.
- `docs/client/logic/Battle.as:1177-1209` and `:597-621`, `:821-843`, `:1059-1080` use `targetNum/areaAttack` for previews. Runtime expansion remains server-owned.
- No ActionScript references `attackNum` or `multiAttack`. Repeated damage must be emitted as repeated server playback actions.
- `docs/client/view/view/compDragable/PetManagerPanel.as:851-865` explicitly says Combo affects `Nhịp Đập Trái Tim`, but Flash does not reveal its roll formula, cap, retargeting, or damage progression.

## Existing server coverage

- `pet_active_skill_catalog.go` and `pet_active_skill_specials.go` cover the previously approved 144 book templates/52 families plus three special-source families.
- `pet_active_skill_templates.go` covers exact-ID creature/evolution mechanics including Nhịp Đập Trái Tim, Hào Quang May Mắn, Bách Biến Trùng Kích, U Minh Huyền Vũ, Thú Vương Đột Kích, and Lôi Đình Thần Phạt.
- Passive stat skills with ordinary `TBL_BUFF.prop*` rows are already aggregated by `internal/application/pet/skill_books.go:145-211`.
- Potential passives `3744..3755` are explicitly implemented in `potential_skill_bonuses.go`; the forum's Bão Tố, Ngưng Tụ, and Nhiệt Huyết coefficients agree with the corresponding rows.
- Unplanned complex active skills fail through to legacy `expSkill/dexSkill` behavior. `pet_skill_rules.json` is currently empty, so the families below do not receive their described mechanics.

## Forum family matrix

| Family/group | Exact seed IDs | Current status | Conclusion |
|---|---:|---|---|
| Báo Chi Tấn Tiệp Cao Cấp | 5178–5180 | Buff-backed passive | Already covered by generic passive stats. |
| Chính Xác Sơ/Trung Cấp | 3523–3525, 3568–3570 | Buff-backed passive | Already covered. `Chính Xác Cao Cấp` has no matching seed family and must not be invented. |
| Kích Thích Tiềm Năng — Bão Tố/Ngưng Tụ/Nhiệt Huyết | 3747–3755 | Explicit potential registry | Already covered; forum's incomplete rows do not replace seed/registry. |
| Liên Tỏa Phản Ứng | 3650, 3676, 3677 | Buff-backed passive | Already adds 7/10/14 Combo points. |
| Long Lân Chi Giáp, Long Giáp, Ngạnh Xác Hóa, Thụ Bì Ngoại Sáo, Phản Kích, Tinh Thần Tập Trung, Trực Giác Dã Thú, Vẻ Ngoài Yếu Ớt | catalog families | Buff-backed passives | Already covered from authoritative buff properties. |
| Cân Bằng Chi Thược | 3763, 3771, 3772 | Buff has no `prop*`; runtime adds nothing | Missing. Seed and forum agree that each final Energy point increases MagicAttack by 0.05/0.10/0.15%. |
| Da Kháng Tính | 3808, 3812, 3813 | Buff has no `prop*`; no turn hook | Missing. At own turn start, independently roll 50/75/100% and remove one removable harmful effect. |
| Cận Vệ Thánh Nữ | 3848–3850 | No passive shield hook | Blocked conflict: forum says shield during the first three battle rounds; seed says activate before every turn. |
| Ảo Hoa Kỳ Hương | 2835, 2834 | Exact plan exists | Keep seed/current 30%×2 and 40%×3. Forum says level 3 lasts four turns and mentions an absent level 2; no correction without a decision. |
| Cao Tần Long Hống Trung Cấp | 2961–2963 | No exact plan | Add M95/110/125% plus independent 15/25/35% MagicAttack −10/16/22% for two turns. |
| Cung Chúc Tân Hỷ | 3245 | Legacy heal with no coefficient | Blocked: seed/forum say heal ten allies but provide no amount. |
| Dã Thú Cuồng Vũ | 3300, 3301, 3303 | Exact family plan | Already covered as 2/2/3 full hits. |
| Diệt Vong Chi Ca | catalog | Exact family plan | Already covered. |
| Duy Trì Tốc Độ | 4020, 4031, 4032 | Exact family plan | Already covered as Dodge +10/20/30. |
| Độc Thích Đơn/Quần Thể | 3547–3561 | Exact family plans | Already covered. |
| Đột Thích Xung Phong | 2666, 2735, 2736 | No exact plan | Add P80/90/100% plus independent 10/15/20% Attack −15/24/33% for two turns. |
| Hấp Thu Năng Lượng | 4022, 4035, 4036 | Exact family plan | Already covered. |
| Kích Trạng Loạn Tâm | 2845, 2839, 2846 | No exact plan | Add one-target confusion at 5/10/15% for two turns. |
| Lạc Thạch Vẫn Tinh | 2692, 2777, 2784 | No exact plan | Add horizontal-three M30/40/50%. |
| Lời Ngọt Ngào | 3319, 3322, 3323 | User-approved exact plan | Keep M100/110/120%. Forum's 99/“500?”/999 fixed values are explicitly uncertain and seed has no fixed damage value. |
| Ma Pháp Thuẫn | 3651, 3674, 3675 | Exact plan | Keep authoritative four-turn seed duration, not forum's three turns. |
| Mê Tâm | 4739, 4741, 4742 | No exact plan | Add random 2/3/4 targets, independent confusion 60/65/70%, duration 2/2/3 from seed. |
| Nhịp Đập Trái Tim | 5363, 5362, 5361 | Exact corrected plan | Keep seed coefficients 120/150/180 unless user explicitly chooses forum's 130% level 1. Combo chain correction remains 5/10/15 casts with reroll/retarget/increasing damage. |
| Nuốt Chửng | 4021, 4033, 4034 | Exact family plan | Already covered by prior user-approved selected-target semantics. |
| Phản Ứng Trì Độn | 2838, 2843, 2844 | No exact plan | Add one-target 5/10/15% proc, Speed −5/7/10% for two turns. |
| Phản Xạ | 5144, 5143, 5142 | No plan | Blocked: shield/reflect magnitudes are absent; “increases with level” is not executable data. |
| Pháo Kích | 3820–3822 | Exact family plan | Already covered as cross-four, independent 60/70/80% same-turn stun. |
| Quyến Luyến Không Rời | 3678–3680 | Exact family plan | Already covered. |
| Sủng Vật Hiệp Hộ | 3379–3381 | No redirect primitive/plan | Blocked: defense/counter values are known, but duration and physical owner-damage redirection lifecycle are unspecified. |
| Sương Đông Đơn/Quần Thể | 3516–3522 | Exact family plans | Already covered. |
| Suy Yếu Linh Hồn | 4019, 4029, 4030 | Prior user-approved plan | Already covered. |
| Tàn Nhẫn Tư Liệt | catalog | Exact family plan | Already covered. |
| Táng Thần Đơn Thể | 3541, 3543, 3545 | Exact family plan | Already covered. |
| Táng Thần Quần Thể | 3542, 3544, 3546 | No exact plan | Add cross-four P90/108/128%, then rest one turn. |
| Thích Nhĩ Hô Khiếu | 2723, 2724 | Exact family plan | Seed contains only levels 2/3 at P120/130%; do not invent level 1 from forum. |
| Thích Vũ | 5133–5135 | No plan | Blocked conflict: forum says outgoing damage −40/50/65%; seed skill says −30/40/60%, while buff text says incoming damage. |
| Thạch Hóa Công Kích / forum “Thạch Hoá Chi Xúc” | 2615 | No exact plan and no stone control field | Add only exact level-1 seed behavior after adding stone control: one target, 5%, two turns. Do not invent levels 2/3. |
| Thuốc Kích Thích | 3590, 3672, 3673 | Exact corrected plan | Keep seed HP costs 30/40/50%, not forum's blanket 50%; caster is the Pet. |
| Tích Lực Sơ/Trung Cấp | 3526–3531 | Exact family plans | Already covered. |
| Tiếng Sét Phẫn Nộ | 5417, 5416, 5415 | No plan | Blocked: forum gives approximate fixed ranges and seed gives no damage amount. |
| Trào Phúng Đơn Thể | 3539, 3563, 3564 | Exact family plan | Keep visible seed description duration 1/3/4; forum conflicts at level 1. |
| Trào Phúng Quần Thể | 3540, 3566, 3567 | No exact plan | Add vertical 2/2 then cross 4, independent 66% taunt, duration 1/2/2 from visible seed descriptions. |
| Tự Nhiên Thuẫn Đơn Thể | 3562, 3573, 3574 | Existing plan partly wrong | Keep seed Defense +1000/+2000/+3000 and targets 1/1/horizontal 5. Correct HoT from 65/65/65% to forum's coherent 25/45/65% MagicAttack sequence. |
| Tự Nhiên Thuẫn Quần Thể | 3575–3577 | Exact family plan | Already matches forum: +1000 Defense, HoT 25%, 4/5/10 allies. |
| Uy Nhiếp Ý Chí | 2725, 2690, 2727 | No exact plan | Add horizontal-three M30/40/50% from complete seed descriptions; forum only has a contradictory/incomplete level-1 line. |

## Confirmed implementation batch

The evidence is sufficient for these changes without inventing mechanics:

1. Exact active plans for 28 templates across ten families: Cao Tần Long Hống Trung Cấp, Đột Thích Xung Phong, Kích Trạng Loạn Tâm, Lạc Thạch Vẫn Tinh, Mê Tâm, Phản Ứng Trì Độn, Táng Thần Quần Thể, Trào Phúng Quần Thể, Uy Nhiếp Ý Chí, and level-1 Thạch Hóa Công Kích.
2. Turn-start passive cleanse for Da Kháng Tính 3808/3812/3813.
3. Dynamic Energy-to-MagicAttack passive scaling for Cân Bằng Chi Thược 3763/3771/3772.
4. Correct Tự Nhiên Thuẫn Đơn Thể HoT coefficients to 25/45/65% while retaining seed defense/target values.
5. Add exact-ID coverage tests ensuring these skills never fall back to legacy damage/status behavior.

## Explicitly deferred

- Cận Vệ Thánh Nữ: first-three-round versus every-turn conflict.
- Cung Chúc Tân Hỷ: missing heal amount.
- Phản Xạ: missing shield and reflection values.
- Sủng Vật Hiệp Hộ: unspecified status lifecycle and redirect semantics.
- Thích Vũ: conflicting values and incoming/outgoing direction.
- Tiếng Sét Phẫn Nộ: approximate forum values only.
- Chính Xác Cao Cấp and missing levels such as Thích Nhĩ Hô Khiếu level 1: no matching seed templates.
- Existing conflict choices (NĐTT level-1 120 vs 130, shield/control durations, Thuốc Kích Thích cost) remain unchanged unless the user explicitly overrides seed authority.
