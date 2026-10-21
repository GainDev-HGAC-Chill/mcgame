# Pet Super-Skill Learning — Research

**Date:** 2026-10-21  
**Scope:** Learning the 11 `Ma Thú Yếu Quyết Siêu Cấp` books through the existing Pet training panel. No ActionScript, schema, seed, deployment, or player-data changes.

## Sources and authority

1. Official guide: <https://vuaphapthuat.go.vn/bai-viet/ky-nang-pet/4000.html>, fetched and inspected on 2026-10-21.
2. Flash wire/UI contract: `docs/client/view/view/compDragable/PetFuncPanel.as`.
3. Checked-in catalogs: `docs/database/game_data/TBL_ITEM_TEMPLATE.json`, `docs/database/game_data/TBL_SKILL.json`, and `docs/database/game_data/TBL_BUFF.json`.
4. Current server flow: `internal/application/pet/skill_books.go` and `internal/presentation/rtmp/handlers/pet/fusion.go`.
5. Operator override: all three prerequisite skills must be level 3, and the learned Super skill must always start at level 1.

## Confirmed official mechanics

The official guide's “Lưu ý” section states:

1. The Pet must own the matching Sơ, Trung, and Cao skills.
2. Those three skills disappear after learning, leaving only the Super skill.
3. Once the Pet owns the Super skill, it cannot relearn the matching Sơ/Trung/Cao skills.
4. The legacy guide says the resulting level follows the lowest prerequisite level.
5. No empty Pet skill slot is required.

Requirement 4 is intentionally overridden by the operator: all three prerequisites must already be level 3 and the result is always the exact level-1 Super row. The server must not select a result by “lowest available ID” or by the lowest prerequisite level.

## Flash contract

- `PetFuncPanel.petBook()` calls `_core.remote.call("petBook", new Responder(onBook), petId, bookItemInstanceId)` (`PetFuncPanel.as:2323-2328`). There is no separate Super-skill RPC.
- The Pet training bag accepts item `type=506` (`PetFuncPanel.as:3251-3257`).
- The Learn button checks the item `reqClass` against the Pet creature `classId`, and Pet `qLevel >= item.proplNum` (`PetFuncPanel.as:3155-3175`). This is separate from the first segment of `TBL_SKILL.creKind`, which is skill-family metadata and is not a Pet `classIds` gate.
- `onBook` reads only `f`, `rf`, and `n` (`PetFuncPanel.as:3031-3058`):
  - `f=1,rf=false`: learn failure;
  - `f=2`: unusable book;
  - `f=3`: invalid Pet/book input;
  - `n`: remaining book stack; zero cleans the panel slot.
- Skill-slot mutation is callback-driven. The existing server uses `onUpdatePet(petId, "skillN", skillId)` and then `onRefreshPetProp`.

## Exact authoritative catalog

All 11 books are live `type=506,useType=2`, require `proplNum=3`, and resolve to distinct three-level Super chains. The required rows below are exact level-3 prerequisites and exact level-1 results.

| Book | Family | Sơ L3 | Trung L3 | Cao L3 | Super L1 |
|---:|---|---:|---:|---:|---:|
| 3216 | Thụ Bì Ngoại Sáo | 2830 | 3345 | 5156 | 5527 |
| 3217 | Trí Tuệ Thiên Nhiên | 2749 | 3348 | 5159 | 5530 |
| 3218 | Thú Tính Tỉnh Ngộ | 2743 | 3351 | 5162 | 5533 |
| 3219 | Cổ Long Chi Hồn | 2821 | 3354 | 5165 | 5536 |
| 3220 | Ngạnh Xác Hóa | 2699 | 3357 | 5168 | 5539 |
| 3221 | Vẻ Ngoài Yếu Ớt | 2842 | 3360 | 5171 | 5542 |
| 3222 | Tinh Thần Tập Trung | 3329 | 3363 | 5174 | 5545 |
| 3223 | Trực Giác Dã Thú | 2746 | 3366 | 5177 | 5548 |
| 3224 | Báo Chi Tấn Tiệp | 2748 | 3369 | 5180 | 5551 |
| 3225 | Vỏ Ngoài Phóng Điện | 2711 | 3372 | 5183 | 5554 |
| 3226 | Long Lân Chi Giáp | 2818 | 3375 | 5186 | 5557 |

The Super level chains are `5527..5529`, `5530..5532`, ..., `5557..5559`. Existing `petUpSkill` can upgrade the level-1 result through its `codeName` chain using the quality-3 manual once the Super learn operation has completed.

## Required slot mutation

Because no empty slot is required, ordinary first-empty-slot behavior cannot be used. For a successful learn:

1. Locate the three exact level-3 prerequisite IDs in canonical `skill1..skill15` slots.
2. Put the exact Super level-1 ID into the lowest-numbered prerequisite slot.
3. Set the other two prerequisite slots to `0` while preserving locked slots (`-1`) and unrelated skills.
4. Emit all changed `onUpdatePet` callbacks in ascending slot order, then one derived-property refresh.

Choosing the lowest occupied prerequisite slot is deterministic and does not invent an extra slot. All three prerequisite skills disappear semantically, while one of their existing slots becomes the Super skill.

## Failure and lockout behavior

- Missing any prerequisite, owning a prerequisite below level 3, or already owning any level of the matching Super chain: fail with the existing learn-failure shape `f=1,rf=false,n=currentStack`; consume nothing and mutate nothing.
- A Pet owning any level of a matching Super chain cannot learn that family's Sơ/Trung/Cao books again.
- Wrong ownership/input remains `f=3`; invalid/non-book/unresolvable/quality or item-class mismatch remains `f=2`.
- A failed path emits no inventory, skill-slot, or stat-refresh callbacks.

## Atomicity risk

The current flow saves the Pet first and consumes the item second, with best-effort rollback. That is insufficient for a three-slot replacement and can leave duplicated/lost state when item persistence fails or requests race.

Production Super learning should therefore use one PostgreSQL transaction that locks and revalidates the Pet skill snapshot plus the owned book row, writes the final Pet, and decrements/deletes exactly one book. Cache state must update only after commit. This requires no schema migration.

## Current gap

The generic resolver already maps each Super book to its level-1 row, but `LearnSkillFromBook` treats it as an ordinary book: it requires an empty slot, does not require or remove the three level-3 prerequisites, and does not block relearning lower tiers after Super. Existing tests that directly learn item 3216 without prerequisites encode this incomplete behavior and must be corrected.
