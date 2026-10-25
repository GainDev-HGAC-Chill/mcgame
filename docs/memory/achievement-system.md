# Achievement Snapshot and Claim Baseline

## Callback contract correction (2026-10-11)

Served Flash `onUpdateAchieve(type:int,id:int,data:Object)` requires three arguments and a real delta: type 1 updates achievement `done`; type 2 updates requirement `done/progress`. It is not a zero-argument refresh. Removed the unconditional empty call from quest reward completion that caused #2095/#1063. Existing farm progress pushes with three real arguments remain. No quest-to-achievement evaluator or fabricated completion state was added. See [Quest reward callback audit](../research/2026-10-11_03_QUEST_REWARD_CALLBACK_AUDIT_RESEARCH.md).

The first achievement slice now has both login snapshot hydration and a minimal claim flow that matches the Flash client contract without inventing missing reward metadata:

- `onChooseCharactor` should seed `achieveLog`, `achieveReqLog`, `takeAchieveAwardLog`, and `cData.achPnt` from `achievement.Service.LoadSnapshot()`.
- `data_tbl_achievement.award` currently behaves as achievement points for the panel total (`achPnt`), not as a structured claim reward payload.
- `getAchieveAward` now safely marks `player.character_achievements.is_claimed = true` through the achievement service and pushes `updateAchieveAwardLog({aid, num})` so the client hides the claim button.
- Title-style rewards are safe when `is_award == 2` and the achievement name matches exactly one `TBL_TITLE.n` row. In that case the server grants the title, refreshes either `onAddTitle({f,t,ct})` or `onAddActTitle(cts)` based on the title list bucket, and then pushes `onAchieveTitle(name)`.
- If the title-name lookup is ambiguous, the server still records the achievement as claimed and skips the title grant rather than guessing.

## Structured rewards (added 2026-05-05)

`data.data_tbl_achievement` now has three nullable columns: `reward_type SMALLINT`, `reward_id INTEGER`, `reward_qty INTEGER`. Migration `supabase/migrations/20260505081133_add_achievement_reward_columns.sql`. All existing rows have NULL — new claims with `reward_type` set will dispatch through `handlers/achievement/rewards.go::grantStructuredReward` after the existing flow completes.

Reward type registry (constants in `rewards.go`):

| `reward_type` | Action                                                                                    |
|---------------|-------------------------------------------------------------------------------------------|
| `1` (money)   | `char.Money += reward_qty`, char saved, `onAddMoney(charID, "money", qty, balance)` push |
| `2` (gold)    | `char.Gold += reward_qty`, char saved, `onAddMoney(charID, "gold", qty, balance)` push   |
| `3` (item)    | `itemService.AddItemToSlot(reward_id, reward_qty, ItemTypeQuest, SlotTypeBag)` then `onAddItem` + `onAddCharactorSlot` |
| `4` (title)   | `titleService.GrantTitle(reward_id)` then `SendTitleGrantedCallback` (only if newly added) |

The handler is wired in `cmd/gameserver/main.go` via `achievementHandler.SetItemService(itemService)` and `SetCharacterService(charService)` (two call sites: dev path ~L384, prod path ~L919).

### Out of scope for this slice

- **No backfill from `desc`.** Free-text reward descriptions are not parsed — that's a separate ticket and could invite injection if done carelessly.
- **No EXP reward type.** EXP grants need level-up handling and aren't covered here.
- **No double-grant guard beyond `is_claimed`.** Once `repo.MarkClaimed` flips, the structured reward fires once. If the structured reward step itself partially fails (e.g. inventory full), the claim is still recorded as claimed — players have to contact a GM. A future hardening pass should make the reward-grant + mark-claimed pair transactional.

## End-to-end audit correction (2026-10-19)

The minimal slice is not a complete achievement system. Static catalog/Flash/server comparison is recorded in [Achievement System Audit](../research/2026-10-19_02_ACHIEVEMENT_SYSTEM_AUDIT_RESEARCH.md).

- `achieveReqLog` must be keyed by requirement ID; current login code keys it by achievement ID, and the one-row-per-achievement schema cannot represent 33 multi-requirement achievements.
- No production evaluator writes progress or completion. `achievement.Service.OnLogin` is a no-op and it implements none of the available gameplay event listeners.
- All 604 seeded achievements have null structured reward columns. Current title fallback searches by achievement name rather than the distinct reward-title name; medal rewards are unsupported.
- Claim currently records `is_claimed` before best-effort grant, is not atomic, and is vulnerable to concurrent duplicate grant. Do not expose it as production-complete until claim+grant is transactional.
- The farm requirement-294 callback is volatile and semantically mismatched; it is not valid persisted achievement progress.

## Triển khai sửa nền tảng (2026-10-19)

Phần code mới thay thế flow claim tối thiểu ở trên:

- `achieveReqLog` được dựng từ `player.character_achievement_requirements` và đánh key bằng requirement ID; `done` là cờ `0/1`, `progress` là giá trị thật.
- `supabase/data/achievement_system.sql` thêm tiến độ từng requirement, event key idempotent, completion nhiều điều kiện và claim reservation/finalize/release. Schema đã apply vào compose DB và được capture ở migration `20261025033410_achievement_system.sql`; ba bảng runtime bật RLS, không có client policy. Shadow replay qua Supabase CLI 2.122.0 PASS và migration history đã mark applied.
- Requirement cấp độ 1–4 có catalog `num=0`, nên target phải lấy giá trị server-owned 10/50/100/140 từ event update thay vì xem 0 là canonical. Runtime login level 100 đã tạo đúng ba completion 1001/1002/1003 và giữ 1004 ở 100/140; không tự claim reward.
- Evaluator có bằng chứng hiện hỗ trợ cấp độ, ngày đăng nhập, tổng nhiệm vụ hoàn thành, tổng trận thắng và thắng Arena. Login reconcile trước snapshot; quest/battle phát `onUpdateAchieve(type,id,data)` đúng contract.
- Reward registry dùng ID canonical cho 29 reward xác minh được từ website + catalog: item 3984/4029, 8 title và 12 base medal. Achievement 6066 không có exact title `Giáo Sư Biết Tuốt` nên fail closed.
- Claim lấy reservation trước grant, chỉ finalize sau grant thành công; lỗi grant release reservation, lỗi finalize giữ reservation để chặn nhận lặp. Item dùng `ItemTypeConsumable`; medal dùng `AddMedalToBag`.
- Callback farm requirement 294 và bộ đếm RAM đã bị xóa vì tăng sai cho người thu hoạch thay vì chủ vườn bị trộm.

## Full-system census và kế hoạch mở rộng (2026-10-25)

- Catalog có 566 achievement bật và 697 requirement bật trên 60 type. Trong đó 431 requirement state-provable, 159 event-only và 107 chưa có authority; production hiện mới có 23 evaluator.
- Có 30 achievement bật nhiều requirement; completion tiếp tục là AND theo requirement ID. Hai achievement `6438/6439` không có requirement nên chỉ được direct-complete bằng exact business event, không được empty-set auto-complete.
- `num=0` xuất hiện ở 550 requirement bật và không mang nghĩa target 0. Registry phải keyed theo requirement ID, khai báo target source và được validate với catalog lúc startup.
- State reconcile chỉ đi theo hướng dương. Event-only cần durable fact/outbox với stable business ID; callback UI không phải authority. Requirement thiếu subsystem tiếp tục fail-closed.
- Addendum đầy đủ S/E/U và ID list nằm trong `docs/research/2026-10-19_02_ACHIEVEMENT_SYSTEM_AUDIT_RESEARCH.md`; implementation waves và reward-ledger recovery nằm trong `docs/plans/2026-10-19_02_ACHIEVEMENT_SYSTEM_REPAIR.md`.
- Wave 0 registry/validator đã code-complete với catalog fingerprint và exact census guard. Wave 1 hiện phủ 421/431 requirement S. Sau batch maker-set/Thần Khí/cánh vợ chồng, batch mới dùng durable marriage audit/current duration cho RID `131/346/347/356–359` và Pet Arena battle history cho RID `411–417`; daily streak tính theo giờ Việt Nam. Positive-only evaluator và hai aggregate function đã apply/capture ở migration `20261025224346_achievement_wave1_state_aggregates.sql`; privilege, rollback và shadow replay PASS, nhưng gameserver chưa restart/deploy. Còn 10 S: RID `21/132/133/289/7499–7504`, đều thiếu exact mapping hoặc durable shop/wedding authority.
- Wave 2 đã harden evaluator level/login/quest/battle: quest cần persisted progress ID + completion count dương; battle cần UUID lowercase stable và một trong 5 battle type canonical. Login-day v2 dùng function DB suy ngày `Asia/Ho_Chi_Minh`, serialize theo character và dedupe event v1/v2 bằng `occurred_at`, không xóa hay giảm progress cũ. Function/index đã apply/capture ở migration `20261025225516_achievement_wave2_login_game_day.sql`; privilege, v1 cutover, same-game-day retry, rollback và shadow replay PASS. Hai event v1 lịch sử cùng game-day giữ nguyên progress 2 theo positive-only policy; gameserver chưa deploy.
- Wave 3 bắt đầu bằng quest durable fact: append-only table + pending checkpoint; `update_quest_progress` ghi atomic fact `quest-completed:v1:<progressID>:<completionCount>` khi count tăng đúng một. Consumer kiểm tra payload/character/ID, hiện chỉ map 4 quest-total RID `248–251`, dùng achievement event dedupe rồi mark processed; login replay xử lý crash gap. Migration `20261025230510_achievement_wave3_quest_facts.sql`, privilege, retry rollback và shadow replay PASS; không để lại test mutation. Code chưa deploy; các quest E còn lại chưa có exact matcher tiếp tục fail-closed.
