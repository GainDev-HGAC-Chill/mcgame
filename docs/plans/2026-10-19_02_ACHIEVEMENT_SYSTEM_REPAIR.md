# Kế hoạch sửa hệ thống Thành Tựu

**Ngày**: 2026-10-19 · **Cập nhật**: 2026-10-25 · **Trạng thái**: Đã duyệt toàn bộ; Wave 0 code-complete
**Mục tiêu**: hoàn thiện Thành Tựu trên mọi tab cho toàn bộ catalog, dựa trên state hoặc business event authoritative; mọi điều kiện chưa đủ authority phải fail-closed.

Cơ sở nghiên cứu:

- `docs/research/2026-10-19_02_ACHIEVEMENT_SYSTEM_AUDIT_RESEARCH.md`: đối chiếu website, Flash, server và catalog; xác định lỗi schema, snapshot, evaluator và claim.
- `docs/research/2026-10-11_03_QUEST_REWARD_CALLBACK_AUDIT_RESEARCH.md`: xác nhận `onUpdateAchieve(type,id,data)` và ý nghĩa type 1/type 2.
- `docs/memory/achievement-system.md`: trạng thái triển khai hiện tại và các giới hạn đã biết.

## 1. Cơ chế đã xác nhận

- Flash đọc `achieveLog` theo achievement ID và `achieveReqLog` theo requirement ID.
- Callback tiến độ là `onUpdateAchieve(2, requirementID, {done,progress})`; callback hoàn thành là `onUpdateAchieve(1, achievementID, {done:<epoch-ms>})`.
- Một achievement hoàn thành khi toàn bộ requirement thuộc achievement đó hoàn thành.
- Catalog có 604 achievement, 739 requirement; 33 achievement có nhiều requirement.
- `getAchieveAward(aid)` là RPC claim hiện có. `updateAchieveAwardLog({aid,num})` cập nhật log nhận thưởng.
- Website chính thức xác nhận mỗi reward chỉ nhận một lần và gồm title, item, Ấn Chương nhân vật/Pet.
- Item 3984 là `Tinh Hồn Chi Thực`; item 4029 là `Quả Linh Tê`.
- Base medal đã xác minh bằng exact name + `level=0`: 1 Lục Quân, 23 May Mắn, 111 Chúng Thần, 133 Thành Nhân, 155 Quang Huy, 317 Chiến Hồn, 328 Trí Tuệ, 361 Tấn Tiệp, 383 Kỳ Ảo, 394 Cuồng Nộ, 416 Mị Hoặc, 449 Kinh Nhận.
- Title reward đã xác minh: 256 Đại Sư Bắt Pet, 257 Huấn Luyện Sư Đa Công Năng, 273 Độc Cô Cầu Bại, 274 Giáo Sư Biết Tuốt qua liên kết title→buff, 275 Đĩa Hút Máu, 276 Củ Cải Phong Lưu, 277 Phú Hào, 278 Địa Chủ, 279 Thợ Săn Tiền Thưởng.
- Catalog có anomaly ở title 274: tên hiển thị là `Tuổi Trẻ Bồng Bột`, nhưng field `b=2793` liên kết duy nhất tới `TBL_BUFF.id=2793` có exact name `Giáo Sư Biết Tuốt`; achievement 6066 và bài hướng dẫn chính thức cùng ghi exact reward name này. Mapping dùng quan hệ ID trực tiếp, không dùng tên gần giống hay thứ tự dòng.

## 2. Thiết kế

### 2.1 Nguồn dữ liệu chuẩn

- Requirement definition tiếp tục lấy từ `TBL_ACHIEVEMENT_REQUIRE`.
- Tiến độ runtime lưu tại bảng mới `player.character_achievement_requirements`, khóa `(character_id, requirement_id)`.
- Event idempotency lưu tại `player.character_achievement_events`, khóa `(character_id,event_key)`.
- Claim reservation lưu tại `player.character_achievement_claim_reservations`, khóa `(character_id,achievement_id)` để chặn hai request cùng cấp thưởng.
- Reward registry Go chỉ chứa các ánh xạ ID đã xác minh từ website + catalog; achievement không có mapping sẽ fail closed và không bị đánh dấu claimed.

### 2.2 Tiến độ và hoàn thành

- Repository ghi một event cùng danh sách requirement update trong một Postgres function.
- Event key ngăn RPC retry cộng lặp.
- Function upsert tiến độ, tính `is_done`, rồi chỉ đánh dấu achievement hoàn thành khi đủ mọi requirement catalog.
- Login snapshot join hai bảng: completion/claim theo achievement và progress theo requirement.
- `done` là cờ hoàn thành `0/1`; `progress` là giá trị hiện tại, cap theo target khi target dương.

### 2.3 Evaluator an toàn đợt đầu

- Cấp độ: requirement 1–4 tương ứng level 10/50/100/140; reconcile khi login và cập nhật khi level-up.
- Ngày đăng nhập: requirement 53–56/423/424; mỗi ngày theo lịch chỉ cộng một lần bằng event key ngày.
- Tổng nhiệm vụ hoàn thành: requirement 248–251; event key dựa trên quest ID + completion count.
- Tổng trận thắng: requirement 185–189; event key dựa trên battle ID.
- Thắng Arena: requirement 151–154 chỉ khi `BattleTypeArena`.
- Requirement type khác không được suy đoán. Chúng giữ nguyên chưa hoàn thành cho tới khi có authority từ subsystem tương ứng.

### 2.4 Claim và phần thưởng

- Claim lấy reservation trong DB trước khi cấp để chặn concurrency.
- Nếu cấp thất bại, reservation được release và `is_claimed` không đổi.
- Nếu cấp thành công, finalize cập nhật `is_claimed` có điều kiện và xóa reservation trong cùng DB transaction.
- Nếu finalize mất kết nối sau khi phần thưởng đã cấp, reservation được giữ lại để chặn nhận lặp; server ghi lỗi mức cao, không tự retry phần thưởng.
- Item reward dùng `ItemTypeConsumable`, không dùng `ItemTypeQuest`.
- Medal reward gọi `medal.Service.AddMedalToBag` và phát `updateMedalInBag({i,t,n})`.
- Title reward cấp bằng title ID canonical và phát callback title hiện có.
- Mọi reward chưa ánh xạ trả thất bại rõ ràng, không claim rỗng.

### 2.5 Điểm chạm

| File | Thay đổi |
|---|---|
| `internal/domain/achievement/*` | Thêm model requirement update, event result, claim reservation và repository contract. |
| `internal/application/achievement/*` | Snapshot đúng key, evaluator đợt đầu, reward registry và orchestration claim. |
| `internal/infrastructure/persistence/postgres/achievement_repository.go` | Gọi các function requirement/event/claim mới. |
| `internal/presentation/rtmp/handlers/achievement/*` | Claim reservation, cấp item/title/medal có lỗi trả về, callback đúng. |
| `internal/presentation/rtmp/handlers/{auth,character,quest,combat}` | Reconcile/record event và phát delta `onUpdateAchieve`. |
| `cmd/gameserver/main.go` | Inject medal service và achievement progress recorder. |
| `supabase/data/achievement_system.sql` | Bảng/function draft để apply compose và sinh migration bằng `supabase db diff`. |
| Tests trong các package liên quan | Snapshot multi-requirement, idempotency, completion, reward mapping, claim failure/concurrency và callback. |

## 3. Ngoài phạm vi

- Không tự suy đoán 60 requirement type từ mô tả tiếng Việt; các type chưa có event/state authority tiếp tục fail closed.
- Không sửa ActionScript.
- Không backfill tiến độ người chơi hiện có ngoài reconcile cấp độ/đăng nhập/tổng quest/battle có dữ liệu authority.
- Không cấp title gần giống; trường hợp title 274 chỉ được chấp nhận khi startup validator còn thấy exact link buff 2793 `Giáo Sư Biết Tuốt`.
- Không restart/deploy hoặc kiểm thử bằng item/người chơi thật.

## 4. Kiểm thử

1. Unit test snapshot dùng requirement ID, `done` boolean và multi-requirement completion.
2. Unit test event trùng không cộng lặp; level/login/quest/battle mapping đúng và unsupported type không thay đổi.
3. Unit test 29 reward mapping; achievement 6066 và reward chưa biết fail closed.
4. Unit test claim: chưa hoàn thành, đã claim, grant lỗi release reservation, finalize lỗi giữ reservation, hai claim đồng thời chỉ một grant.
5. SQL rollback verification trên compose DB không chứa player mutation tồn tại; fresh migration replay.
6. `go test` package achievement/handler và các handler chạm tới; targeted `go vet`; `go build -o bin/gameserver.exe ./cmd/gameserver`.
7. Smoke test Flash do người dùng thực hiện sau deploy: mở `R`, quan sát tiến độ, hoàn thành fixture không tiêu hao dữ liệu thật, reconnect, claim trên fixture test và xác nhận không nhận lần hai.

## 5. Trạng thái nền tảng sau đợt đầu

Đợt đầu đã hoàn tất và deploy:

- persistence theo requirement ID, event dedupe và phép AND cho nhiều requirement;
- login snapshot/callback đúng Flash contract;
- claim reservation/finalize/release;
- migration `20261025033410_achievement_system.sql`, RLS và shadow replay;
- 23 evaluator production cho level, login day, quest total, battle wins và Arena wins;
- runtime xác nhận các mốc cấp 10/50/100 hiện nhận thưởng và vẫn chưa claim.

Phần còn thiếu là phủ toàn catalog. Census authoritative nằm trong addendum của `docs/research/2026-10-19_02_ACHIEVEMENT_SYSTEM_AUDIT_RESEARCH.md`: 431 state-provable, 159 event-only và 107 unsupported trên 697 requirement bật.

## 6. Kiến trúc toàn hệ thống

### 6.1 Requirement registry tĩnh

Thêm registry theo **requirement ID**. Mỗi entry phải khai báo:

- achievement ID và requirement type dự kiến;
- evaluator kind: `state`, `event`, `direct-completion` hoặc `unsupported`;
- target/threshold canonical và target source;
- authority adapter cùng version;
- event matching fields nếu là event-only.

Startup validator đối chiếu registry với `TBL_ACHIEVEMENT_REQUIRE` và `TBL_ACHIEVEMENT`. Server fail startup hoặc quarantine entry nếu ID, parent, type, target source hay dependency drift. Không dùng riêng `type`, không parse `name/description` ở runtime.

Hai achievement `6438/6439` không có requirement dùng direct-completion registry và chỉ được hoàn thành bởi exact business event đã xác minh. SQL completion không được coi empty set là true.

### 6.2 State reconcilers

Tạo các reader theo domain, trả về positive proofs cho một snapshot nhất quán:

- character/progression/wallet/final stats;
- quest completion;
- Pet roster, equipment, contract, evolution;
- character/life skills và recipe;
- equipment/loadout/dress;
- title, guild, relationship, marriage;
- shop history, Magic Estate, Pet Arena;
- Star Instance, Fairy, Mystery Treasure, Hắc Diệu Thạch, PRS.

Reconcile chạy lúc login sau khi state authoritative đã load và tại post-commit hook của mutation liên quan. Kết quả chỉ dùng max/mark-done; state vắng mặt không giảm/xóa achievement progress. Mỗi batch dùng event key dạng `state:<registry-version>:<authority-revision>` để retry an toàn.

Không nhồi SQL/query từng subsystem vào handler Thành Tựu. Adapter đọc qua domain repository/service hiện có hoặc Postgres function schema-qualified khi cần aggregate dài.

### 6.3 Durable event facts

Event-only phải được tạo tại transaction/business boundary thành công, không phải từ callback UI. Mỗi fact gồm tối thiểu:

- `fact_id` hoặc business transaction ID unique;
- character/actor ID, event kind, occurred-at;
- payload version và exact dimensions cần match: template, quest, battle context, role, quantity, result;
- dedupe key ổn định qua retry/reconnect/restart.

Thêm append-only `player.character_achievement_facts` hoặc outbox tương đương và consumer checkpoint. Mutation domain ghi business state + outbox trong cùng transaction nếu cùng DB; consumer chuyển fact thành requirement updates idempotent. Với service chưa thể transactional-outbox ngay, hook chỉ được mở khi có persisted business ID đủ để replay.

Không dùng generic `BattleEnded(charID,battleType,won)` cho boss, instance hoặc party attribution. Event mới phải mang battle ID, encounter/template/context và winner/contribution policy. Login day phải dùng timezone/reset-day canonical của game, không dùng UTC thô.

### 6.4 Completion và callback

- Requirement update và parent completion vẫn nằm trong một Postgres transaction.
- Multi-requirement dùng AND trên toàn bộ requirement đang bật của parent.
- Callback chỉ phát sau commit: requirement delta type 2 trước, completion type 1 sau.
- Retry, duplicate fact hoặc reconnect không được phát tăng lặp.
- Snapshot luôn là nguồn phục hồi nếu callback bị mất.

### 6.5 Reward ledger và recovery

Giữ fail-closed cho achievement chưa có reward mapping canonical. Đối với mapping đã xác minh:

- tạo durable grant ledger keyed `(character_id, achievement_id)` với trạng thái `reserved/granting/granted/finalized/failed`;
- reward operation phải có idempotency key truyền xuống item/title/medal grant;
- lưu grant receipt trước finalize claim;
- startup/worker recovery chỉ finalize receipt đã grant hoặc retry operation idempotent; không cấp mù lần hai;
- callback claim chỉ trả success khi ledger ở `finalized`.

Không parse `desc`, không dùng tên gần giống và không tự cấu hình 51 reward claimable chưa xác minh đầy đủ.

## 7. Các wave triển khai

### Wave 0 — Registry và guardrail

Phạm vi:

1. Sinh/duy trì registry cho đủ 697 requirement bật và 2 direct achievements.
2. Startup validation: unique ID, parent/type match, coverage S/E/U đủ 697, target source hợp lệ, không auto-complete `num=0`/empty set.
3. Test census cố định: `431 + 159 + 107 = 697`; 30 parent nhiều requirement; 2 parent không requirement.
4. Sửa login-day calendar authority và siết battle event hiện có trước khi mở thêm progress.

Kết quả: catalog drift không thể âm thầm cộng sai.

Trạng thái 2026-10-25: code-complete. `registry.go` phân loại đủ 697 requirement, khóa fingerprint catalog, census S/E/U, cấu trúc multi/empty parent và target authority; `cmd/gameserver` fail startup nếu drift. Fixture catalog thật và drift regression PASS. Chưa deploy runtime mới.

### Wave 1 — 431 state-provable requirements

Triển khai theo nhóm để giữ review/test nhỏ:

1. Character, exact quest, skill/recipe, title, guild/social/marriage/shop: 95 requirement.
2. Pet roster/equipment/contracts/evolution: 72 requirement.
3. Equipment/loadout/dress activation và dress sets: 114 requirement.
4. Magic Estate, Pet Arena history, Star Instance: 85 requirement.
5. Fairy, achievement points, Mystery Treasure, Hắc Diệu Thạch, PRS: 65 requirement.

Mỗi nhóm phải có table-driven registry tests, positive/negative state fixtures, multi-requirement completion và reconnect snapshot. Không backfill bằng suy đoán; login reconcile từ state hiện có là hợp lệ vì state tự chứng minh điều kiện.

Trạng thái 2026-10-25: framework reconcile và 11 reader code-complete, phủ 177/431 requirement S ở character, 7 exact quest, 14 life skill/recipe, social, guild, title, achievement points, Star Instance, Hắc Diệu Thạch, PRS và Mật Bảo. Reader khai báo ownership khi startup; duplicate/non-state/unknown và proof ngoài ownership fail-closed. Character final stat `47–49`, daily act `421`, 5 quest nghề và phần skill/Pet/equipment còn lại chờ authority mapping tiếp theo. SQL positive-only đã sửa trong canonical draft nhưng chưa apply/capture migration, nên chưa deploy Wave 1.

### Wave 2 — Harden 23 evaluator hiện có

1. Level giữ target 10/50/100/140.
2. Login day dùng game-day/reset boundary và event key versioned.
3. Quest total dùng unique quest-completion business ID, không lấy UI callback.
4. Battle total/Arena dùng exact match ID, result và battle context; loại encounter không đủ authority.
5. Chạy migration dữ liệu chỉ nếu cần ledger metadata, không sửa progress hiện có theo hướng giảm.

Trạng thái 2026-10-26: code + DB complete, chưa deploy. Level giữ target server-owned; quest yêu cầu persisted progress identity; battle yêu cầu UUID canonical + exact enum. Login-day v2 dùng ngày `Asia/Ho_Chi_Minh`, advisory lock và dedupe event v1/v2 theo `occurred_at`; migration `20261025225516_achievement_wave2_login_game_day.sql`, privilege, v1 cutover, rollback retry và shadow replay đã PASS.

### Wave 3 — 159 event-only requirements

Triển khai outbox/fact producer theo domain:

1. battle/boss/Arena: 67;
2. quest/loop/checkpoint: 30;
3. life skills/farm/cooking/medicine/fishing: 15;
4. item/action/capture/merchant/wishing tree: 24;
5. guild donation/estate dated actions/Pet Arena settlement: 17;
6. login day: 6, đã harden ở Wave 2.

Mỗi producer cần: post-success placement, stable business ID, retry test, duplicate test, crash/replay test và actor attribution test. Requirement 294 giữ disabled cho tới khi giải quyết mâu thuẫn actor bằng source contract/runtime evidence.

Trạng thái bước đầu 2026-10-26: quest completion có outbox atomic keyed progress ID + completion count và consumer checkpoint/replay; mới map 4 RID tổng nhiệm vụ `248–251`. Migration `20261025230510_achievement_wave3_quest_facts.sql`, privilege/retry rollback/shadow replay PASS; code chưa deploy. 26 quest event RID còn lại giữ fail-closed tới khi có exact matcher theo quest catalog/business context.

### Wave 4 — 107 unsupported requirements

Không mở bằng stub hoặc fabricated progress. Thứ tự bổ sung authority đề xuất:

1. các subsystem đã có UI/game-data nhưng thiếu persistence rõ ràng: Ảo Ma Tháp, Hành Lang Vô Tận, Quiz, Mê Cung;
2. các feature-state còn thiếu: Chi Hồn, Hệ Thống Giác Quan, utility bag, PK/popularity;
3. các instance/event lớn: Đông Huyền, Ảo Cảnh Tầm Bảo, Không Gian Rèn Luyện, Kỳ Thạch, Mạo Hiểm Bí Cảnh, Huyết Chiến Cổ Bích, Achilles;
4. sáu legacy instance chưa map và các ranking/settlement còn lại.

Mỗi subsystem phải có plan riêng về wire contract, persistence, reset semantics và event IDs trước khi nối achievement evaluator. Khi authority mới hoàn tất, entry registry chuyển U→S/E cùng regression census.

### Wave 5 — Reward completion và recovery

1. Kiểm kê 51 achievement claimable bằng nguồn canonical ngoài `desc`.
2. Thêm reward registry cho từng mapping đã xác minh, gồm item/title/character medal/Pet medal.
3. Thêm ledger/idempotent grant/recovery.
4. SQL concurrency và crash-point tests tại trước grant, sau grant, trước finalize, sau finalize.
5. Mapping chưa chứng minh tiếp tục trả lỗi rõ ràng và không claim.

Trạng thái 2026-10-26: code + DB complete, chưa deploy. Đã xác minh đủ 30 reward trong nguồn chính thức; title 274 được khóa qua exact linked buff 2793. Durable ledger có lease, persisted receipt, on-login finalize recovery và idempotent operation key xuống item/medal/title. Migration `20261026045011_achievement_reward_ledger.sql`, rollback-smoke các crash point, exact service-role privilege và full shadow replay PASS; không claim reward thật trong verification.

## 8. Điểm chạm dự kiến

| Khu vực | Thay đổi |
|---|---|
| `internal/application/achievement/` | Registry, validator, state reconcilers, fact consumer, delta orchestration. |
| `internal/domain/achievement/` | Definition/proof/fact/ledger contracts và typed context. |
| `internal/infrastructure/persistence/postgres/achievement_repository.go` | Batch reconcile, fact/outbox/checkpoint và reward ledger function calls. |
| `internal/application/events/` | Thay generic events bằng typed/versioned business facts; không phụ thuộc UI. |
| Domain services liên quan | Ghi outbox tại post-success boundary; không đặt achievement logic trong RTMP handlers. |
| `internal/presentation/rtmp/handlers/achievement/` | Claim ledger/recovery response và callback sau commit. |
| `cmd/gameserver/main.go` | Wiring adapters, validator, consumer/recovery worker. |
| `supabase/data/achievement_system.sql` | Batch reconcile, fact/outbox/checkpoint, reward ledger và recovery functions. |
| `supabase/migrations/` | Chỉ sinh bằng Supabase CLI sau apply/verify compose DB. |

## 9. Kiểm thử và tiêu chí hoàn thành

### Tự động

- registry coverage đủ 697 enabled requirement + 2 direct achievements;
- target `num=0`, disabled parent, empty requirements và catalog drift đều fail-closed;
- positive-only state reconcile, concurrent login và repeated reconcile idempotent;
- multi-requirement AND, callback ordering và reconnect snapshot;
- event producer post-success, stable dedupe, outbox replay và consumer restart;
- SQL concurrency cho requirement updates, completion và claim/reward ledger;
- reward crash recovery không mất thưởng/không cấp lặp;
- focused tests từng domain, `go test` affected packages, targeted `go vet`, build `bin/gameserver.exe`, `git diff --check`;
- Supabase shadow migration replay và RLS/advisor review.

### Runtime an toàn

- dùng read-only snapshot trước/sau login để xác minh Wave 1, không tạo tiến độ giả;
- hành động event-only chỉ kiểm tra khi operator chủ động thực hiện gameplay bình thường;
- không tự nhận thưởng, dùng item/Pet, mua bán hoặc sửa dữ liệu để ép progress;
- Flash smoke trên cả 7 nhóm tab: progress, completion, reconnect và claim chỉ với reward đã xác minh;
- log không có failed reconcile, duplicate grant hoặc callback contract error.

### Định nghĩa “làm hết”

Hệ thống hoàn tất khi mọi achievement bật thuộc một trong hai trạng thái minh bạch:

1. **Supported**: có registry + authority + persistence + test + runtime verification; hoặc
2. **Explicitly blocked**: fail-closed với subsystem/authority thiếu được ghi rõ, không hiển thị completion giả.

Không thể coi 107 requirement U là implemented chỉ bằng evaluator đoán. Chúng chỉ chuyển sang supported sau khi subsystem tương ứng được xây dựng và kiểm chứng.

## 10. Approval gate

Trước khi sửa code cần operator duyệt:

- triển khai Wave 0–2 và Wave 1 cho 431 state-provable trước;
- sau đó Wave 3 theo từng domain, vì chạm nhiều transaction/economy path;
- Wave 4 được tách thành các feature plan riêng cho 107 requirement thiếu authority;
- reward ledger/mapping chỉ mở cho ID canonical đã xác minh.

Không migration, restart/deploy, backfill hoặc player mutation chỉ dựa trên việc duyệt plan; các thao tác rủi ro vẫn cần checkpoint và xác nhận riêng theo quy tắc repository.
