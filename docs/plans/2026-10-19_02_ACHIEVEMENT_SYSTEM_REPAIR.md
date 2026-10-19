# Kế hoạch sửa hệ thống Thành Tựu

**Ngày**: 2026-10-19 · **Trạng thái**: Code hoàn tất, SQL chờ compose verification/migration
**Mục tiêu**: biến phần khung login/claim hiện tại thành một lát cắt Thành Tựu chạy end-to-end, lưu đúng tiến độ từng điều kiện, hoàn thành idempotent, phát đúng callback Flash và cấp an toàn các phần thưởng chính thức có ID đã xác minh.

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
- Title reward đã xác minh: 256 Đại Sư Bắt Pet, 257 Huấn Luyện Sư Đa Công Năng, 273 Độc Cô Cầu Bại, 275 Đĩa Hút Máu, 276 Củ Cải Phong Lưu, 277 Phú Hào, 278 Địa Chủ, 279 Thợ Săn Tiền Thưởng.
- Không có exact title `Giáo Sư Biết Tuốt` trong catalog. Title 274 có tên khác nên không được dùng thay thế.

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
- `Giáo Sư Biết Tuốt` và mọi reward chưa ánh xạ trả thất bại rõ ràng, không claim rỗng.

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

- Không tự suy đoán toàn bộ 57 requirement type từ mô tả tiếng Việt; các type chưa có event/state authority tiếp tục fail closed.
- Không sửa ActionScript.
- Không backfill tiến độ người chơi hiện có ngoài reconcile cấp độ/đăng nhập/tổng quest/battle có dữ liệu authority.
- Không cấp title gần giống cho `Giáo Sư Biết Tuốt`.
- Không restart/deploy hoặc kiểm thử bằng item/người chơi thật.

## 4. Kiểm thử

1. Unit test snapshot dùng requirement ID, `done` boolean và multi-requirement completion.
2. Unit test event trùng không cộng lặp; level/login/quest/battle mapping đúng và unsupported type không thay đổi.
3. Unit test 29 reward mapping; achievement 6066 và reward chưa biết fail closed.
4. Unit test claim: chưa hoàn thành, đã claim, grant lỗi release reservation, finalize lỗi giữ reservation, hai claim đồng thời chỉ một grant.
5. SQL rollback verification trên compose DB không chứa player mutation tồn tại; fresh migration replay.
6. `go test` package achievement/handler và các handler chạm tới; targeted `go vet`; `go build -o bin/gameserver.exe ./cmd/gameserver`.
7. Smoke test Flash do người dùng thực hiện sau deploy: mở `R`, quan sát tiến độ, hoàn thành fixture không tiêu hao dữ liệu thật, reconnect, claim trên fixture test và xác nhận không nhận lần hai.
