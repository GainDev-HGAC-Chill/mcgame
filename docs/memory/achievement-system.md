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
- Reward registry dùng ID canonical cho đủ 30 reward xác minh từ website + catalog: item 3984/4029, 9 title và 12 base medal. Achievement 6066 map title 274 bằng liên kết catalog trực tiếp `TBL_TITLE.b=2793` tới exact `TBL_BUFF.name=Giáo Sư Biết Tuốt`; startup validator khóa quan hệ này thay vì tin tên hiển thị title đang lệch.
- Claim reward dùng durable ledger keyed `(character_id,achievement_id)` với state `reserved/granting/granted/finalized/failed`, grant lease 5 phút và receipt JSONB. Item grant idempotent bằng `achievement_reward_key` trên stack riêng; medal giữ receipt trong `character_stat_features.medal.achievementGrants`; title grant idempotent theo unique ownership. Receipt phải persisted trước finalize; login tự finalize ledger `granted`, còn `granting` hết lease được RPC retry bằng cùng business key. Callback inventory/title/medal chỉ phát sau finalize. DB migration `20261026045011_achievement_reward_ledger.sql` đã apply, rollback-smoke và full shadow replay PASS; code chưa deploy.
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
- Authority completion 2026-10-26: census implementation là `472 S + 74 E + 151 U`. State readers phủ đúng `472/472`; event producers phủ đúng `74/74`. Bổ sung persisted TempBag/Pop/Awaken, 32 exact title-backed RID, Magic Estate steal-log aggregate, derived tứ thần/tứ ma, arena participation, 50 exact creature-kill RID và exact quest facts `226/254`. Các RID thiếu durable producer, subsystem hoặc exact mapping chuyển sang U, không tạo tiến độ giả. Focused suites, full vet, Windows build và diff-check PASS; full test chỉ còn các baseline questaudit/activity/chat đã tồn tại trước batch. Chưa deploy/restart hoặc runtime smoke.
- Pet acquisition authority follow-up 2026-10-26 code-complete: RID `83` dùng fact `pet-captured:v1:<petID>` được ghi cùng transaction tạo Pet; RID `106–109` dùng fact `wish-pet-drawn:v1:<drawUUID>:<ordinal>` được ghi cùng Gold, free quota và Lotto holding-bag state. Wish `TBL_PLAN.ti=12`, `q>=10` normalize `/10`, raw positive decimal dưới 10 giữ nguyên, sau đó dùng `domain/pet.ColorByGrowRate`; mọi Pet mở `106`, màu lam/tím/cam mở `107/108/109`. Registry mới `472 S + 79 E + 146 U`. Combat/Lotto handler xử lý fact ngay và login vẫn recovery; retry dùng event key `fact:v1:<factID>`. Mystery Merchant `102–105` vẫn U; chỉ có fact payload contract `mystery-merchant-pet-purchased`, chờ purchase ledger/currency/Pet transaction thật. Wish Pet claim `ti=12` vẫn là defect riêng. Migration append-fact draft `20261026120000_pet_acquisition_achievement_facts.sql` chưa apply; không deploy/restart/runtime smoke/backfill hoặc mutate player.
- PK/rename authority follow-up 2026-10-27 code-complete: Flash dùng `honor` làm mức tên đỏ nên persisted `Character.Honor` là state authority RID `138–141`, target 10/25/100/200. RID `150` dùng fact `character-renamed:v1:<operationUUID>`; transaction trực tiếp lock/consume đúng card instance template 2126, update `characters.name` và append fact cùng commit, sau đó mới sync cache/callback. Census mới `476 S + 80 E + 141 U`. Focused và broad vet/build PASS; full test chỉ còn baseline questaudit/activity/chat. Generic append-fact migration vẫn chưa apply, nên chưa deploy/restart/runtime smoke hoặc mutate player.
- Runtime update 2026-10-28: migration `20261026120000_pet_acquisition_achievement_facts.sql` đã apply/history-mark, shadow replay và ACL PASS. Natural capture template179 live-created Pet385 plus durable processed fact `pet-captured:v1:385`, event dedupe, RID83 and AID2016 completion in the same acquisition flow. Guild-loop turn-in later consumed Pet385 while the fact remained durable, proving lifecycle separation. All test fact/event/requirement/achievement rows were removed during exact baseline cleanup; no reward was claimed.
- RID84 correction 2026-10-29: catalog RID84/AID2017 `Bắt pet thành công` target5000 is capture-event authority. Title256 `Đại Sư Bắt Pet` is the AID2017 reward and must never be used as completion state. A validated `pet-captured:v1:<petID>` fact increments RID83 target1 and RID84 target5000 under the same durable dedupe event. Registry census is now `475 state / 81 event / 141 unsupported`; no historical RID84 backfill is inferred from mutable title or Pet roster state.
