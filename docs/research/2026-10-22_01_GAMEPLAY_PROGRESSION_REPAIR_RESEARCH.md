# Research: sửa năm nhóm gameplay/progression

## Phạm vi và an toàn

Đối chiếu Flash client chỉ đọc, Go server và catalog cho: nhiệm vụ Thần Khí sáu class, kỹ năng nhánh Tu Luyện, slot Mệnh Hồn Pet cấp 80, Huy Chương Mạo Hiểm và Khóa Học Đặc Huấn. Không sửa ActionScript, không tiêu vàng/vật phẩm thật, không mutate dữ liệu người chơi, không restart/deploy.

## 1. Nhiệm vụ Thần Khí

Sáu nhiệm vụ chính xác là `4707..4712`. Tất cả đều:

- `type=10`, cấp tối thiểu 50, không có `TBL_QUEST_REQUIRE` và không có `TBL_QUEST_PRE`;
- có `startText`, `completeText`, `finishNpc=272` và class riêng;
- thưởng đúng một main magic weapon `kind=8/type=800`, bound, quality `q=1`.

| Class | Quest | Trang bị thưởng |
|---|---:|---:|
| 1 | 4707 | 1348 |
| 2 | 4709 | 1349 |
| 3 | 4708 | 1350 |
| 4 | 4711 | 1351 |
| 5 | 4710 | 1352 |
| 6 | 4712 | 1353 |

Root cause: `ClassifyQuestCompletion` chỉ cho type-10 rỗng đi qua các policy đặc biệt đã biết; sáu quest trên rơi vào `PolicyUnconfigured`, nên `CompletionBlocked=true` và không thể trả. Reward path type 19 đã hỗ trợ trang bị và không phải nguyên nhân.

Hướng sửa an toàn: thêm registry exact-fingerprint cho cả sáu quest, yêu cầu đúng ID/type/class/start NPC/finish NPC/tên và dialogue, rồi dùng conversation completion tại NPC 272. Catalog drift tiếp tục fail-closed.

## 2. Kỹ năng nhánh Tu Luyện

Flash `SkillUseSlot.learn()` phân nhánh theo `useEnv`:

- học level đầu của `useEnv=3`: RPC `addTrainingSkill(skillId)`;
- nâng cấp: RPC `skillLearnByClient(nextSkillId)`.

Server không đăng ký `addTrainingSkill`. `skillLearnByClient` có tồn tại nhưng `UpgradeSkillWithResult` chỉ chấp nhận ordinary skill `(useEnv=0,kind=1)` hoặc `(useEnv=1,kind=2)`, nên từ chối toàn bộ Tu Luyện.

Catalog có 120 row `useEnv=3,kind=5`, dùng chung cho sáu class:

- bốn nhánh: Sinh Mệnh, Phòng Vật Lý, Phòng Ma Pháp, Tốc Độ;
- ba bậc Sơ/Trung/Cao, mỗi bậc 10 level;
- `reqClass=|1|2|3|4|5|6|`, gate `reqCL` lần lượt 1/2/3 và gate level riêng;
- chi phí nâng nằm ở `price` và `expSkill`; từng family có `codeName` ổn định.

Hướng sửa: RPC học level 1 riêng, validation exact `useEnv=3/kind=5/level=1`; mở progression exact-next cho cùng `codeName`; giữ level/class-rank/EXP/bạc và mutation lock hiện có. Không gộp các bậc có `codeName` khác nhau.

Các buff Tu Luyện có mô tả nhưng `TBL_BUFF.prop*` đều bằng 0 và hiệu lực phụ thuộc cấp/class/Pet; catalog hiện không đủ công thức để tự bổ sung stat. Lượt sửa này chỉ khôi phục học/nâng và persistence, không bịa công thức hiệu lực.

## 3. Mệnh Hồn Pet cấp 80

Flash hiển thị 16 slot Pet:

- slot 1..8 mở theo `soulInfo.openNum`;
- slot 9..16 mở trả Soul EXP theo `openNum2`, giá lần lượt `20000..55000`;
- tooltip khẳng định Pet cấp 80 trở lên có thể dùng Mệnh Hồn;
- `openPetSoulBag` chỉ cho mua slot phụ khi `openNum >= 5`.

Server hiện tạo state Pet mặc định `openNum=0`, chỉ có test/seed helper ghi `openNum`; không có production writer khi Pet đạt cấp 80. DTO vì vậy tiếp tục khóa toàn bộ slot dù `pet.level >= 80`. RPC mua slot phụ kiểm tra state stale này, nên cũng trả `ErrPetBagLocked`.

Hướng sửa: derive/reconcile base slots từ Pet level tại read/mutation boundary; Pet dưới 80 có 0 base slot, Pet từ 80 có đủ 8 base slot. Chỉ tăng derived `openNum`, không giảm state cũ, không đụng `openNum2` hay Soul EXP. `OpenPetSoulBag` phải kiểm tra ownership và level Pet authoritative thay vì tin pet ID/state đơn lẻ.

## 4. Huy Chương Mạo Hiểm

Flash có ba thao tác:

- `levelUpEMedal`: dùng Điểm Mạo Hiểm;
- `levelUpEMedalByGold(cid,0)`: 50 Vàng, cộng 50 tiến độ;
- `levelUpEMedalByGold(cid,1)`: 500 Vàng, cộng 500 tiến độ.

Panel response là `[level, cumulativePropMap, currentEMP, currentProgress]`. Tooltip nói normal upgrade dùng Điểm Mạo Hiểm và nạp điểm thiếu vào tiến độ. `TBL_EXPLORER_MEDAL.cost` là yêu cầu tiến độ cấp kế; `costGold` trong catalog bằng `cost`, không phải giá 50/500 Vàng.

Root causes server:

- thiếu đăng ký/handler `levelUpEMedalByGold`;
- state `info="level|score"` đang hiểu trường hai là score tích lũy, trong khi response luôn gửi progress bằng 0;
- normal upgrade so score như threshold và không tiêu/nạp Điểm Mạo Hiểm;
- điểm client `explorerPnt` thực tế là currency ID 54 (`CurrencyShowTime`, persisted ở `ShowTimePoint`) nhưng service hiện không đọc ví này.

Hướng sửa: hiểu state là `level|progress`, đọc/ghi ví `explorerPnt`, normal button chuyển tối đa số điểm cần thiết vào progress, gold button trừ 50/500 Gold và cộng đúng 50/500 progress, tự xử lý qua nhiều level nếu đủ. Giữ per-character lock và bù trừ khi save lỗi. Hiệu ứng “bạo kích” không có xác suất/hệ số authority nên giữ fail-closed, không tự tạo bonus ngẫu nhiên.

Thuộc tính Huy Chương đã có hiệu lực qua stat-feature provider; vấn đề hiện tại là mutation/progress, không phải aggregate stat.

## 5. Khóa Học Đặc Huấn (TXKC)

Server hiện chỉ có `getTXKCData`, trả level 1/exp 0 và cửa sổ `now..now+7 ngày` không persist. Flash chờ đầy đủ RPC:

- `getTXKCData`
- `openTXKCAdvanceCourse` — 680 Gold
- `buyTXKCEXP(count)` — mỗi count là 500 EXP, giá 300 Gold
- `takeTXKCDailyQuest`
- `TXKCGetProgressEXP(1,dailyTaskId)` / `(2,weeklyTaskId)`
- `takeTXKCFreeAward`, `takeTXKCAdvanAward`
- `openTXKCShop`

Wire state `flag` gồm `lev`, `exp`, `p`, `awd1`, `awd2`, optional `dd`, `spIndex`, `wd`. Mỗi level cần 500 EXP, max 70. Daily thường cấp 120 EXP, daily đặc biệt 150 EXP, weekly 280 EXP. Reset daily 00:00, weekly 00:00 thứ Hai.

Client chứa authority đầy đủ cho 70 phần thưởng free và 70 phần thưởng premium. Item 6734 hoàn trả 680 Gold ở premium level 1, phù hợp giá mở khóa. Các item reward đều tồn tại trong catalog. Client cũng chứa 10 mô tả daily và 6 mô tả weekly.

Những khoảng trống authority:

- không có lịch mùa toàn cục trong catalog; cửa sổ 7 ngày hiện tại là placeholder;
- không có shop/slot catalog tham chiếu item điểm TXKC, nên `openTXKCShop` chưa thể mở shop thật mà không đoán shop ID;
- cơ chế/xác suất chọn `spIndex` chưa được mô tả;
- một số nguồn tiến độ task (Ước nguyện, đấu Pet, trang viên, dungeon) chưa có event chuẩn ở mọi runtime path.

Hướng sửa không đoán:

- lưu state TXKC trong `character_stat_features`, reset theo `Asia/Ho_Chi_Minh`;
- dùng lịch mùa explicit/configured, không dùng moving window;
- hỗ trợ mua premium/EXP, level carry, claim reward theo map client và inventory capacity;
- tạo task tracker theo event authority; trước mắt chỉ phát task có nguồn sự kiện server xác minh được, task chưa có hook không được báo hoàn thành giả;
- `openTXKCShop` trả thông báo unavailable cho tới khi có shop ID/slot authority.

## Kết luận

Bốn nhóm đầu đã có root cause và contract đủ để triển khai an toàn. TXKC đủ contract cho state, mua, reset, task UI và reward, nhưng cần chốt lịch mùa; shop và critical/special-task randomness vẫn phải fail-closed nếu chưa có authority.
