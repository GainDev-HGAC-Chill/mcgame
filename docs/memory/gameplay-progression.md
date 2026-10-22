# Gameplay progression systems

## Quest Thần Khí

- Quest `4707..4712` là conversation quest type 10 có class riêng, cùng trả tại NPC 272.
- Mapping class/quest/reward: `1/4707/1348`, `2/4709/1349`, `3/4708/1350`, `4/4711/1351`, `5/4710/1352`, `6/4712/1353`.
- Completion phải qua exact catalog fingerprint và runtime class/NPC gate; không coi objective rỗng là tự hoàn thành.

## Kỹ năng Tu Luyện

- Flash học level đầu qua `addTrainingSkill(skillId)`, nâng qua `skillLearnByClient(nextSkillId)`.
- Authority là `useEnv=3,kind=5`, exact-next cùng `codeName`; catalog có 12 family × 10 level = 120 row và dùng chung sáu class.
- Buff Tu Luyện hiện không có `prop*` authority, nên học/nâng được persist nhưng không được bịa stat formula.

## Mệnh Hồn Pet

- Pet level 80+ có tám slot cơ bản (`openNum=8`); slot 9–16 dùng `openNum2` và Soul EXP.
- Mọi mutation slot phụ phải xác minh Pet owner và level trước khi tiêu tài nguyên. Reconcile chỉ tăng base slots, không giảm state legacy và không đổi `openNum2`.

## Huy Chương Mạo Hiểm

- Feature state `explorer_medal.info` là `level|progress`, không phải điểm tích lũy.
- Ví client là `explorerPnt` (`CurrencyShowTime`); panel là `[level,cumulativePropMap,wallet,progress]`.
- Gold type 0 = 50 Gold/50 progress; type 1 = 500 Gold/500 progress. `TBL_EXPLORER_MEDAL.cost` là threshold progress, không phải giá Gold.

## Khóa Học Đặc Huấn

- Feature key `special_training_course`; season key ngăn state mùa cũ chảy sang mùa mới.
- Mùa 2026 đã duyệt: `2026-10-22 00:00` đến `2026-12-31 00:00`, end-exclusive, UTC+7 Việt Nam.
- Level đầu là 1, mỗi level 500 EXP, max 70; premium 680 Gold; mỗi gói EXP 500 EXP/300 Gold.
- Daily reset 00:00, weekly reset thứ Hai 00:00; daily/weekly claim lần lượt 120/280 EXP.
- Reward map free/premium 70 cấp lấy nguyên từ Flash và item phải tồn tại trong catalog. Shop và special-task bonus phải fail-closed khi thiếu authority.
- Không tự cộng task chỉ từ RPC/UI; progress phải đến từ business event authoritative qua `RecordTask`.
