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

## Addendum toàn hệ thống — 2026-10-25

### Phạm vi và phương pháp

Addendum này mở rộng audit từ lát cắt nền tảng sang toàn bộ catalog. Kết quả được đối chiếu từ:

- `docs/database/game_data/TBL_ACHIEVEMENT.json` và `TBL_ACHIEVEMENT_REQUIRE.json`;
- contract Flash tại `AchievementPanel.as:749,826,982,1749-1750`, `AchievementComparePanel.as:1352` và `AchievementDetail.as:1268`;
- schema/runtime state hiện có dưới `player.*`, các feature-state JSONB và service Go;
- producer thực tế trong `internal/application/events/bus.go` và các call trực tiếp vào achievement service.

Không suy điều kiện từ câu mô tả nếu thiếu state/event authority. Không backfill, claim, dùng item/Pet hoặc thay đổi dữ liệu người chơi trong bước audit.

### Census đã kiểm chứng

| Chỉ số | Kết quả |
|---|---:|
| Achievement toàn catalog / đang bật | 604 / 566 |
| Requirement toàn catalog / thuộc achievement bật | 739 / 697 |
| Requirement thuộc achievement tắt | 42 |
| Requirement type distinct | 60 (`0–54`, `57–61`; không có `55–56`) |
| Achievement bật có nhiều requirement | 30, tổng 163 requirement |
| Achievement bật không có requirement | 2: `6438`, `6439` |
| Requirement bật có `num=0` | 550 |

Hai dòng không requirement là `6438 Siêu May Mắn` và `6439 Phá Tan Vận Xui`. Chúng cần direct completion event riêng; không được coi tập điều kiện rỗng là tự hoàn thành. `num=0` phổ biến và chỉ phản ánh target không nằm trong field này, không phải target bằng không.

### Phân loại authority

| Nhóm | Số lượng | Quy tắc |
|---|---:|---|
| S — state-provable | 431 | Persisted state hiện tại đủ chứng minh đã đạt; reconcile chỉ đi theo hướng dương. |
| E — event-only | 159 | Chỉ business event authoritative mới đếm đúng; cần event key bền vững và dedupe. |
| U — unsupported | 107 | Chưa có subsystem, persistence hoặc mapping đủ chính xác; tiếp tục fail-closed. |
| **Tổng** | **697** |  |

Absence trong state không được xóa tiến độ đã đạt. Event-only không được suy ngược từ trạng thái hiện tại hoặc callback UI. Login day thuộc E vì ledger mới chỉ chứng minh các ngày kể từ khi bắt đầu ghi.

### Ma trận 60 requirement type

`S/E/U` là số requirement đang bật.

| Type | S/E/U | Ngữ nghĩa chính | Authority / quyết định |
|---:|---:|---|---|
| 0 | 0/55/11 | Boss/quái đặc biệt; Hành Lang Vô Tận | Kill event chính xác; Hành Lang thiếu subsystem. |
| 1 | 12/14/6 | Quest/phụ bản | Exact completed quest state; aggregate dùng event; 6 instance chưa map. |
| 2 | 56/0/0 | Pet, phẩm chất, skill, trang bị, khế ước | Pet/equipment persistence + catalog. |
| 3 | 0/2/0 | Bắt Pet thành công | Capture transaction event. |
| 4 | 29/0/7 | Cấp, kinh tế, slot, chỉ số, PK | Character/wallet/state; PK và điểm âm chưa có authority/model. |
| 5 | 28/0/0 | Skill/công thức đã học | Character/life skill persistence. |
| 6 | 94/0/0 | Trang bị, cánh, dress, phi thăng | Item slot + equipment/dress catalog/state. |
| 7 | 3/0/16 | Danh hiệu, Chi Hồn, câu cá xếp hạng | Title state; hai subsystem sau thiếu authority. |
| 8 | 0/6/0 | Ngày đăng nhập khác nhau | Dated event ledger, theo lịch/reset game. |
| 9 | 6/14/0 | Dùng vật phẩm/hành động; title vĩnh viễn | Title state hoặc exact successful action event. |
| 10–14 | 0/13/0 | Nấu ăn, chế thuốc, câu cá, trồng/hái | Successful production/gather event có template ID. |
| 15–17 | 0/18/0 | Nhóm boss, quest vòng, trộm farm | Exact boss/quest/theft event; RID 294 còn mâu thuẫn actor. |
| 18–19 | 0/0/9 | Ảo Ma Tháp | Chưa có floor/run authority. |
| 20 | 0/5/0 | Tổng trận thắng | Battle result event + battle ID. |
| 21–23 | 8/0/0 | Guild, bạn bè, sư đồ | Guild/member/friends relationship state. |
| 24–26 | 0/14/0 | Donate guild, Arena tham gia/thắng | Successful transaction/match result event. |
| 27–29 | 0/0/5 | Achilles | Chưa có subsystem/match result. |
| 30–31 | 0/8/0 | Pet từ thương nhân/cây ước nguyện | Purchase/wish result event. |
| 32 | 1/0/0 | Một lần mua giá trị lớn | `shop_purchases` positive proof. |
| 33 | 3/0/7 | Chỉ số cuối; Hệ Thống Giác Quan | Final stat authority; Giác Quan chưa có state. |
| 34 | 9/0/0 | Hôn nhân | Marriage state + append-only audit. |
| 35–36 | 0/0/0 | Treasure Arena | Chỉ thuộc achievement tắt. |
| 37–38 | 0/0/5 | Túi tiện dụng; Quiz | Thiếu persisted distinction/subsystem. |
| 39–40 | 6/0/0 | Kết bái; trang viên | Relationship/Magic Estate state. |
| 41 | 0/7/0 | Chuỗi ngày farm/trang viên | Dated action ledger. |
| 42 | 7/7/0 | Đấu Trường Pet | Battle/ranking state; settlement lặp cần event ledger. |
| 43 | 73/0/1 | Tinh Giới/Star Instance | Score grid; RID 462 “Siêu Cấp” chưa có field. |
| 44 | 17/0/0 | Tiểu Tinh Linh | Fairy/skin feature state. |
| 45–47 | 0/0/5 | Đông Huyền; Mê Cung | Chưa có run/score/morale authority. |
| 48–49 | 21/0/0 | Tiến hóa Pet; điểm Thành Tựu | Pet evolution state; tổng điểm từ completion. |
| 50–52 | 0/0/23 | Tầm Bảo, Không Gian Rèn Luyện, Kỳ Thạch | Chưa có stage/rank/action authority. |
| 53–54 | 32/0/0 | Bộ dress; Mật Bảo | Dress book/Mystery Treasure collection state. |
| 57–58 | 26/0/0 | Hắc Diệu Thạch; PRS | Persisted feature state + exact registry. |
| 59–60 | 0/0/8 | Mạo Hiểm Bí Cảnh | Chưa có timed-run/ranking ledger. |
| 61 | 0/0/4 | Huyết Chiến Cổ Bích | Chưa có killer/buff authority. |

### 431 requirement có thể reconcile từ state

| Authority | Count | Requirement IDs |
|---|---:|---|
| Character progression/wallet/slots/final stats | 32 | `1–4,17–20,22,47–49,57,58,100,101,134–137,142–149,395,421,510,535` |
| Exact completed quests | 12 | `259–269,511` |
| Pet roster/equipment/contracts/evolution | 72 | `59–66,71–79,81,82,85–91,97–99,307–319,343,513,539–554,667–674,765–768` |
| Character/life skills/recipes | 28 | `5–14,276,281–286,295–301,303–306` |
| Equipment/loadout/dress activation | 94 | `23–38,289,342,360–375,377,378,499,512,515,516,569,592–594,611–651,710–714,761–764` |
| Titles | 9 | `50–52,7499–7504` |
| Guild/friends/tutor/sworn relationships | 9 | `111–118,398` |
| Historical shop purchase | 1 | `21` |
| Marriage | 9 | `131–133,346,347,356–359` |
| Magic Estate | 5 | `399–403` |
| Pet Arena history | 7 | `411–417` |
| Star Instance score grid | 73 | `425–461,463–498` |
| Fairy/skin | 17 | `500–509,523–525,536,537,560,561` |
| Derived achievement points | 5 | `555–559` |
| Dress-part sets | 20 | `687–698,741–744,754–757` |
| Mystery Treasure | 12 | `675–686` |
| Hắc Diệu Thạch | 11 | `699–709` |
| PRS/Pet forms | 15 | `715–723,725,734–736,758,759` |

### 159 requirement event-only

| Domain | Count | Requirement IDs |
|---|---:|---|
| Boss/quái đặc biệt | 55 | `190–207,209–211,231–238,240–243,326,329–332,514,518,526–534,570–575` |
| Quest aggregate/daily/special/loop | 30 | `122–128,221–223,226,244–251,254–258,324,325,327,328,337,338` |
| Bắt Pet | 2 | `83,84` |
| Ngày đăng nhập | 6 | `53–56,423,424` |
| Dùng item/đổi tên/giới tính/cưới | 14 | `150,270–273,277,278,348–353,376` |
| Chế tạo/câu cá/trồng/hái | 13 | `274,275,279,280,287,288,290–292,379–382` |
| Trộm farm | 2 | `293,294` |
| Trận thắng | 5 | `185–189` |
| Donate guild | 3 | `119–121` |
| Arena | 7 | `151–157` |
| Pet từ merchant/wishing tree | 8 | `102–109` |
| Farm/estate theo ngày | 7 | `404–410` |
| Pet Arena settlement | 7 | `745–750,753` |

### 107 requirement tiếp tục fail-closed

| Thiếu authority | Count | Requirement IDs |
|---|---:|---|
| Hành Lang Vô Tận | 11 | `659–666,751,752,760` |
| Legacy instance chưa map | 6 | `213–215,396,397,517` |
| PK point / negative popularity | 7 | `138–141,392–394` |
| Chi Hồn / fishing contest | 16 | `170–182,383–385` |
| Ảo Ma Tháp | 9 | `216–220,333–336` |
| Achilles | 5 | `161–165` |
| Hệ Thống Giác Quan | 7 | `652–658` |
| Túi tiện dụng / Quiz | 5 | `386–391` |
| Star Instance “Siêu Cấp” | 1 | `462` |
| Đông Huyền / Mê Cung | 5 | `519–522,538` |
| Ảo Cảnh Tầm Bảo | 7 | `562–568` |
| Không Gian Rèn Luyện | 5 | `576–580` |
| Kỳ Thạch | 11 | `581–591` |
| Mạo Hiểm Bí Cảnh | 8 | `726–733` |
| Huyết Chiến Cổ Bích | 4 | `737–740` |

### Anomaly và ràng buộc triển khai

- Production hiện chỉ có evaluator cho 23/697 requirement bật: level, login day, quest total, battle wins và Arena wins.
- Generic battle-win hiện có nguy cơ đếm quá rộng; mọi event phải mang exact battle context và business ID.
- Login day hiện dựa UTC; phải đổi sang ngày/reset timezone của game trước khi coi là hoàn tất.
- Requirement `294` có name/description mâu thuẫn actor trộm/bị trộm; không nối lại callback farm cũ cho tới khi có contract chắc chắn.
- 30 achievement nhiều requirement phải hoàn thành bằng phép AND trên requirement ID; danh sách: `1015,1036,2012,2013,2044,5005,5010,5013,5017,5035,5037,6010,6020,6045,6046,6047,6101,6102,6103,6187,6210,6250,6251,6252,6253,6305,6306,6307,6414,6424`.
- Reward canonical vẫn chỉ xác minh một phần trong 51 achievement claimable. Không parse `desc` để cấp thưởng.
- Reservation hiện chặn double grant nhưng grant và finalize chưa cùng transaction; cần durable reward ledger để phục hồi crash sau grant.

## Addendum reward chính thức — 2026-10-26

Đối chiếu toàn bộ bốn bảng tại [Nhận Thưởng Thành Tựu](https://vuaphapthuat.go.vn/bai-viet/nhan-thuong-thanh-tuu/4975.html) với `TBL_ACHIEVEMENT`, `TBL_ITEM_TEMPLATE`, `TBL_TITLE`, `TBL_BUFF` và `TBL_MEDAL` xác minh đủ 30 dòng: 9 title, 9 item, 8 Ấn Chương nhân vật và 4 Ấn Chương Pet. Registry dùng exact achievement ID, item/title/medal ID và số lượng; không parse mô tả lúc runtime.

Trường hợp duy nhất không exact theo tên hiển thị title là achievement `6066 Giáo Sư Biết Tuốt`. `TBL_TITLE.id=274` hiển thị `Tuổi Trẻ Bồng Bột`, nhưng title này có `b=2793`; `TBL_BUFF.id=2793` có exact name `Giáo Sư Biết Tuốt`, và không có title khác liên kết buff này. Kết hợp với exact reward name trong achievement 6066 và nguồn chính thức, đây là quan hệ catalog trực tiếp đủ để map `6066 → title 274`. Startup validation phải khóa cả `title 274 → buff 2793 → exact buff name`; nếu drift thì fail startup.

Reward ledger mới không dựa vào callback UI để xác nhận cấp thưởng. DB giữ state machine và receipt; item lưu stable key trên một stack riêng có unique expression index, medal lưu stable key cùng JSON state được grant, title dựa trên unique ownership. Crash sau mutation nhưng trước receipt có thể retry sau lease mà operation key trả lại cùng item/medal/title; crash sau receipt nhưng trước claim finalize được login hoặc request kế tiếp finalize mà không grant lại. Callback live chỉ phát sau finalize, còn snapshot/inventory persisted là recovery source khi callback bị mất.

## Addendum authority reclassification — 2026-10-26

43 requirement được nâng sang state authority sau audit lại. RID `386/387` là số ô Túi Tiện Dụng mở qua Flash RPC `openTempSlot`; server mutation tăng `characters.temp_bag_slots` (`Character.TempBagSlots`) với giới hạn 5. RID `392–394` là các mốc nhân khí âm `-500/-5000/-10000`; `Character.Pop` và cột DB là signed state, nên current value thấp hơn ngưỡng là positive proof và reconciliation không bao giờ giảm tiến độ đã đạt. RID `652–658` map bảy giác quan vào `AwakenLevel >= 11/22/33/44/55/66/77`: Flash `AwakenPanel` dùng `NODE=11`, còn server persist level 0–77.

31 RID còn lại dùng title ownership vĩnh viễn đã persist, chỉ khi exact catalog relation còn hợp lệ: Chi Hồn `170–182`, Achilles `163`, Ảo Ma Tháp `220/335/336`, câu cá `383–385`, Kỳ Thạch `582–584`, Điểm Danh `128`, sáu requirement của parent Thần Ma `327–332` và Marathon `337`. Reader khóa title ID, buff ID, exact title/buff name và level (`TBL_TITLE.l`), nên catalog drift fail-closed. 10 RID từng xếp S nhưng chưa có exact authority (`21/132/133/289/7499–7504`) được chuyển sang U thay vì giữ coverage giả. Census implementation là `464 S + 151 E + 82 U`; state-reader coverage đạt đủ `464/464 S`.

Mười state requirement chưa có reader vẫn fail-closed sau audit lại: RID `21` có bảng `shop_purchases` nhưng không có producer trong purchase flow; RID `132/133` cần wedding tier nhưng runtime hiện chỉ tạo `ring_type=1`; RID `289` mô tả `Cỏ Nữ Thần` không có exact equipment name/ID trong catalog; RID `7499–7504` không có exact title/buff mapping. Không dùng tên gần giống hoặc table rỗng làm bằng chứng.

Battle resolution có canonical UUID và authoritative enemy participant template IDs, nên 50 requirement boss/quái được nối trực tiếp bằng exact ID + exact `TBL_CREATURE.name`; drift catalog làm event fail-closed. Các requirement nhiều mục vẫn giữ AND ở parent; riêng Mehdi RID `530` chỉ phát khi đủ hai creature ID trong cùng trận, còn Pollux/Catos là hai RID riêng `572/573`. Hai requirement tổng hợp RID `324/325` không được suy từ một kill: state reader chỉ proof sau khi đủ bốn persisted source requirement `194–197` hoặc `202–205`; RID `326` dùng cùng exact Ác Thần ID 542.

Magic Estate có durable `character_magic_estate_logs`: row người trộm dùng `guest=false`, `cid=character_id`, `result=1`. Aggregate read-only theo ngày `Asia/Ho_Chi_Minh` chứng minh max streak và max thắng trong một ngày cho RID `406–410`; không dùng các log này cho RID `404/405` làm vườn. Title `256 → buff 2692` exact name/level chứng minh RID `84` Đại Sư Bắt Pet. Quest fact consumer map thêm exact quest `2579` cho RID `226` và quest `3202` cho RID `254`, khóa name/type/subtype catalog.

Final authority census của implementation là `472 S + 74 E + 151 U`. Mọi S có đúng một registered reader (`472/472`), mọi E có đúng một producer (`74/74`) và regression khóa hai chiều. 69 requirement trước đây chỉ được mô tả là event-only nhưng thực tế chưa có durable producer/aggregate được chuyển E→U: `83,102–109,119–127,150,221–223,244–247,255–258,270–275,277–280,287–288,290–294,338,348–353,376,379–382,404–405,518,745–750,753`. Các flow tương ứng hoặc chưa tồn tại, hoặc mutation hiện không atomic với fact, hoặc thiếu exact matcher/history; vì vậy giữ E sẽ biểu thị coverage giả. Chúng tiếp tục fail-closed cho tới khi subsystem cung cấp stable business ID và durable fact/state.
