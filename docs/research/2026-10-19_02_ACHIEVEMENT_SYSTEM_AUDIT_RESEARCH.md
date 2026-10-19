# Báo cáo kiểm tra hệ thống Thành Tựu

Ngày: 2026-10-19

## Phạm vi

Chỉ kiểm tra tĩnh. Đối chiếu bài hướng dẫn chính thức [Nhận Thưởng Thành Tựu](https://vuaphapthuat.go.vn/bai-viet/nhan-thuong-thanh-tuu/4975.html) với contract Flash đang phục vụ, seed game-data, persistence, application service và RTMP claim handler. Không sửa code, migration, restart server hay thay đổi dữ liệu người chơi/catalog trong bước kiểm tra.

## Hành vi chính thức

Bài hướng dẫn xác nhận:

- Người chơi mở bảng Thành Tựu bằng biểu tượng thành tựu hoặc phím `R`.
- Bảng hiển thị chi tiết, tiến độ điều kiện và điểm thành tựu.
- Thành tựu có thưởng đã hoàn thành chỉ được nhận một lần.
- Phần thưởng gồm danh hiệu, vật phẩm đặc biệt, Ấn Chương nhân vật và Ấn Chương Pet.
- Danh sách minh họa gồm 9 phần thưởng danh hiệu, 9 phần thưởng vật phẩm, 8 Ấn Chương nhân vật và 4 Ấn Chương Pet.
- Phần thưởng đã hủy hoặc phân giải không được nhận lại.

Tên các phần thưởng trong bài viết có trong catalog legacy local, ví dụ item 3984 `Tinh Hồn Chi Thực`, item 4029 `Quả Linh Tê`, các dòng danh hiệu và các họ `TBL_MEDAL`.

## Contract Flash

- `AchievementPanel` đọc thời điểm hoàn thành từ `player.achieveLog[achievementID]` và điểm từ `player.achPnt`.
- Trạng thái điều kiện được đánh key bằng **requirement ID**, không phải achievement ID: `_charAchieveReqLog[requirement.id]`.
- `AchievementDetail` chỉ gọi `getAchieveAward(aid)` sau khi thành tựu hoàn thành.
- Với `isAward == 1`, `takeAchieveAwardLog[aid]` làm ẩn nút nhận sau khi claim. Với `isAward == 2`, nhánh Flash đã kiểm tra không áp dụng điều kiện ẩn này.
- Tiến độ trực tiếp phải dùng đúng callback `onUpdateAchieve(type, id, data)`: type 1 trỏ tới thành tựu, type 2 trỏ tới điều kiện.

## Thống kê catalog

Kết quả phân tích seed tĩnh:

- 604 dòng thành tựu; 566 dòng đang bật.
- 739 dòng điều kiện thuộc 602 thành tựu.
- 33 thành tựu có nhiều điều kiện; nhiều nhất là 14 điều kiện.
- 51 thành tựu có `is_award` khác 0 (`38` dòng giá trị `1`, `13` dòng giá trị `2`).
- Cả 604 dòng đều có `reward_type`, `reward_id`, `reward_qty` bằng `NULL`.

Vì vậy ba cột được thêm bởi `20260505081133_add_achievement_reward_columns.sql` hiện chưa mô tả bất kỳ phần thưởng thật nào.

## Trạng thái server hiện tại

### Phần nền đã có

- Khi đăng nhập, server nạp `achieveLog`, `achieveReqLog`, `takeAchieveAwardLog` và `achPnt` từ `player.character_achievements`.
- RPC `getAchieveAward` đã đăng ký và chặn dòng không tồn tại, chưa hoàn thành hoặc đã nhận.
- Điểm thành tựu được tính từ `TBL_ACHIEVEMENT.award` cho các dòng hoàn thành đã lưu.
- Repository đã có hàm liệt kê, upsert, hoàn thành và đánh dấu đã nhận.

### Thiếu sót và lỗi nghiêm trọng

1. **Chưa có bộ máy tính thành tựu.** `achievement.Service` chỉ có `OnLogin` rỗng. Service không nhận sự kiện đánh quái, nhặt/dùng item, lên cấp, chiến đấu, nhiệm vụ hay cường hóa. Không có production caller gọi `UpsertProgress` hoặc `MarkCompleted`, nên nhân vật mới không thể tự tạo tiến độ hay hoàn thành thành tựu.

2. **Persistence không biểu diễn được mô hình điều kiện của Flash.** `player.character_achievements` chỉ có một dòng cho mỗi `(character_id, achievement_id)` và không có bảng tiến độ từng điều kiện. Mô hình này không thể lưu 33 thành tựu nhiều điều kiện.

3. **Snapshot dùng sai key điều kiện.** `LoadSnapshot` ghi `AchieveReqLog[achievementID]`, nhưng Flash đọc `AchieveReqLog[requirementID]`. Ví dụ thành tựu 1008 dùng requirement ID 8, còn server lại phát key `1008`.

4. **Ý nghĩa `done` trong snapshot đang sai.** Service phát target vào `done` cho mọi dòng tiến độ. Flash coi `done > 0` hoặc `progress > 0` là điều kiện enum đã xong, nên điều kiện mới làm một phần có thể bị hiển thị như đã hoàn thành sau khi sửa key.

5. **Callback tiến độ duy nhất hiện có vừa tạm thời vừa sai ngữ nghĩa.** Thu hoạch nông trại tăng bộ đếm trong RAM, mất khi restart, rồi phát requirement 294 cho người thu hoạch. Requirement 294 trong catalog là `Số lần bị trộm cây`; handler còn chạy cho cả thu hoạch thông thường và không lưu tiến độ.

6. **Không có cấu hình phần thưởng.** Vì toàn bộ cột reward có cấu trúc đều `NULL`, handler không cấp item, Ấn Chương hoặc danh hiệu theo ID cho dữ liệu hiện tại. Trường `desc` legacy chỉ là nội dung hiển thị, không thể dùng làm authority giao dịch.

7. **Tra danh hiệu bằng sai tên.** Ví dụ thành tựu `Đẳng Cấp Đại Gia` phải cấp danh hiệu `Phú Hào`, nhưng handler tìm `TBL_TITLE.n` bằng `Đẳng Cấp Đại Gia`. Phần lớn danh hiệu trong bài hướng dẫn không thể được cấp. Catalog danh hiệu còn có trường hợp trùng/thiếu tên, nên parse chuỗi hiển thị không phải cách sửa an toàn.

8. **Reward dispatcher chưa hỗ trợ Ấn Chương.** Nó chỉ hỗ trợ tiền, vàng, item thường và danh hiệu. Ấn Chương nhân vật/Pet thuộc Medal feature có persistence riêng qua `AddMedalToBag`, không phải item túi thường.

9. **Claim và cấp thưởng không nguyên tử.** `ClaimReward` đặt `is_claimed=true` trước khi gọi `grantStructuredReward`. Các hàm cấp thưởng chỉ ghi log rồi bỏ qua khi lỗi, trong khi handler vẫn trả `flag:true`. Túi đầy hoặc lỗi cấp danh hiệu có thể làm mất vĩnh viễn lượt nhận mà không có thưởng.

10. **Không chặn claim đồng thời.** Service đọc rồi mới update, còn SQL update không có điều kiện `is_claimed=false`. Hai request đồng thời có thể cùng thấy trạng thái chưa nhận và cùng cấp thưởng.

11. **Item thường bị cấp với `ItemTypeQuest`.** Các vật phẩm đặc biệt trong bài hướng dẫn là item sử dụng thông thường; loại nội bộ phải được xác định từ catalog canonical thay vì ép thành quest item.

12. **Hành vi UI của `is_award == 2` chưa được giải quyết.** Flash vẫn để nút claim hiện, trong khi server từ chối lần nhận thứ hai. Dữ liệu còn có dòng `is_award == 2` nhưng mô tả không phải danh hiệu, vì vậy không thể mặc định mọi giá trị 2 đều là reward danh hiệu.

## Đánh giá

Hệ thống **chưa hoạt động end-to-end**. Nó chỉ có thể hiển thị và claim các dòng hoàn thành được tạo thủ công, nhưng chưa thể tính thành tựu, lưu đúng tiến độ điều kiện hoặc cấp phần thưởng theo thiết kế. Cho người chơi thật sử dụng claim hiện tại có nguy cơ mất thưởng âm thầm và nhận lặp.

## Thứ tự triển khai an toàn

1. Thêm cấu hình phần thưởng canonical theo achievement ID, có loại rõ ràng cho danh hiệu, item, Ấn Chương nhân vật và Ấn Chương Pet; validate mọi ID tham chiếu.
2. Thêm persistence từng điều kiện theo `(character_id, requirement_id)` và suy ra trạng thái hoàn thành thành tựu trong transaction.
3. Tạo evaluator registry dựa trên catalog cho các requirement type có đủ bằng chứng, nhận sự kiện domain hiện có và reconcile idempotent khi đăng nhập cho thành tựu dựa trên trạng thái.
4. Thay claim read-then-write bằng một transaction/advisory lock: kiểm tra hoàn thành/chưa nhận, cấp thưởng, rồi mới ghi claimed; lỗi phải rollback toàn bộ.
5. Phát đúng callback Flash và dựng lại login snapshot bằng requirement ID với `done/progress` trung thực.
6. Thêm test evaluator, nhiều điều kiện, reconnect, claim đồng thời, túi đầy và callback contract trước khi smoke-test với người chơi.

Không suy đoán reward ID từ mô tả tiếng Việt và không đánh dấu claimed cho người chơi hiện có khi backfill cấu hình.
