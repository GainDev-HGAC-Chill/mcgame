# Plan: sửa năm nhóm gameplay/progression

**Trạng thái:** Implemented — authority gaps deferred

**Research:** [2026-10-22_01_GAMEPLAY_PROGRESSION_REPAIR_RESEARCH.md](../research/2026-10-22_01_GAMEPLAY_PROGRESSION_REPAIR_RESEARCH.md)

## Kết quả cần đạt

1. Trả được đúng quest Thần Khí cho đủ sáu class và nhận đúng main magic weapon.
2. Học/nâng được bốn nhánh Tu Luyện theo catalog và đúng gate/chi phí.
3. Pet cấp 80 mở tám slot Mệnh Hồn cơ bản; tám slot phụ vẫn dùng Soul EXP đúng giá.
4. Huy Chương Mạo Hiểm dùng đúng `explorerPnt`, có progress thật và mua 50/500 bằng Gold hoạt động.
5. TXKC có state thật, lịch ngày/tuần đúng giờ Việt Nam, mua premium/EXP, task và claim reward.

## Unit A — Quest Thần Khí sáu class

- Thêm exact fingerprint registry `4707..4712` vào completion policy.
- Chỉ complete khi đúng class và ở NPC 272; catalog drift/missing dialogue fail-closed.
- Regression table-driven đủ sáu class, sai class, sai NPC và reward `1348..1353`.

## Unit B — Nhánh kỹ năng Tu Luyện

- Đăng ký `addTrainingSkill` trong handler skill/combat, giữ method có log.
- Thêm service path học level 1 cho `useEnv=3,kind=5`.
- Mở exact-next upgrade cho cùng training family; tái dùng EXP, bạc đã chọn, compensation và callback hiện có.
- Test bốn nhánh × ba bậc đại diện, sáu class, level/class-rank/cost, duplicate và concurrent mutation.
- Không triển khai stat formula thiếu authority trong batch này.

## Unit C — Mệnh Hồn Pet cấp 80

- Reconcile `openNum=8` khi Pet owner có level >=80; dưới 80 giữ 0 trừ state legacy cao hơn.
- Validate ownership + Pet level trong `OpenPetSoulBag`.
- Không sửa `openNum2`; giữ giá Soul EXP `20000..55000`.
- Test level 79/80, foreign Pet, legacy state, slot 9 đầu tiên và không tiêu EXP khi reject.

## Unit D — Huy Chương Mạo Hiểm

- Đổi state parser/serializer thành `level|progress` và panel response đúng bốn phần tử.
- Inject character repository/currency accessor; dùng currency ID 54 `explorerPnt` cho normal progression.
- Thêm `levelUpEMedalByGold(cid,type)` với type 0 = 50 Gold/50 progress, type 1 = 500 Gold/500 progress.
- Xử lý carry qua nhiều level, max 100, overflow an toàn, per-character lock và compensation persistence.
- Push `updateEMPanel`, wallet callback và stat refresh sau mutation.
- Không thêm random critical khi thiếu probability/multiplier authority.

## Unit E — TXKC core

- Tạo package application TXKC, state versioned dưới `feature_key='special_training_course'`.
- State gồm season key/start/end, level/exp, premium, free/premium claimed level, daily/weekly task progress+claimed, day/week reset keys.
- Dùng `Asia/Ho_Chi_Minh`; daily reset 00:00 và weekly reset thứ Hai 00:00.
- Implement/register sáu RPC mutation còn thiếu và thay starter `getTXKCData` bằng persisted read model.
- Gold mutations dùng per-character lock, validate count, overflow và compensation; premium 680 Gold, EXP 300 Gold/500 EXP.
- Snapshot reward map 70+70 từ Flash vào Go, validate mọi template khi startup/test; claim theo level, preflight capacity, chống nhận lặp.
- Task progress đi qua event API idempotent. Chỉ wire các nguồn đã có business event authoritative (quest completion/loop completion, boss handbook, farm/dungeon/Pet event khi có exact hook); task không có hook giữ trạng thái chưa hoàn thành.
- `openTXKCShop` fail-closed kèm system message cho tới khi có shop ID/slot catalog, không mở shop đoán.

## Lịch mùa đã duyệt

Mùa toàn server bắt đầu **2026-10-22 00:00 Asia/Ho_Chi_Minh**, kéo dài **70 ngày**, kết thúc **2026-12-31 00:00** (end-exclusive). Lịch đặt trong runtime configuration để mùa sau đổi không cần sửa code. Không dùng cửa sổ trượt theo từng lần mở panel.

## Verification

- Focused tests: quest, skill/combat handler, soul/Pet, explorer medal/activity, TXKC/event hooks.
- Full affected application/handler suites, targeted `go vet`.
- `go build -o bin/gameserver.exe ./cmd/gameserver` và Linux `CGO_ENABLED=0` build vào `bin/`.
- `git diff --check`; xác nhận không có `.as` thay đổi.
- Không restart/deploy, không tiêu vàng/vật phẩm thật, không mutate player data, không commit/push.

## Ngoài phạm vi / deferred

- Công thức stat Tu Luyện phụ thuộc class/Pet nhưng catalog buff bằng 0.
- Bạo kích tiến độ Huy Chương thiếu probability/multiplier.
- TXKC shop thiếu shop ID/slot authority; task source chưa tồn tại không được giả hoàn thành.
