# Gameplay progression systems

## Quest Thần Khí

- Quest `4707..4712` là conversation quest type 10 có class riêng, cùng trả tại NPC 272.
- Mapping class/quest/reward: `1/4707/1348`, `2/4709/1349`, `3/4708/1350`, `4/4711/1351`, `5/4710/1352`, `6/4712/1353`.
- Completion phải qua exact catalog fingerprint và runtime class/NPC gate; không coi objective rỗng là tự hoàn thành.

## Kỹ năng Tu Luyện

- Flash học level đầu qua `addTrainingSkill(skillId)`, nâng qua `skillLearnByClient(nextSkillId)`.
- Authority là `useEnv=3,kind=5`, exact-next cùng `codeName`; catalog có 12 family × 10 level = 120 row và dùng chung sáu class.
- Buff Tu Luyện không có `prop*` authority. Người dùng đã duyệt curve balance thay thế, ghi rõ không phải công thức gốc: giữ base Sơ cấp khôi phục được, multiplier Sơ/Trung/Cao `1/1.5/2.25`, hệ số nghề từ aptitude `TBL_CLASS`, hệ Pet từ bảng median cố định, clamp `0.75..1.25`. Runtime chỉ nhận exact fingerprint 120 row, lấy level cao nhất trong từng family, cộng dồn 12 family độc lập vào HP/Vật Phòng/Ma Phòng/Tốc Độ; catalog/class/family/level drift trả zero.
- Sau khi học hoặc nâng thành công, handler phải phát lại final sheet ngay: `onUPP` cho nhân vật và `onRefreshPetProp` cho mọi Pet đã enrich. Chỉ refresh skill list không đủ vì client giữ property sheet cũ đến lần đăng nhập sau.

## Mệnh Hồn Pet

- Pet level 80+ có tám slot cơ bản (`openNum=8`); slot 9–16 dùng `openNum2` và Soul EXP.
- Mọi mutation slot phụ phải xác minh Pet owner và level trước khi tiêu tài nguyên. Reconcile chỉ tăng base slots, không giảm state legacy và không đổi `openNum2`.
- Mutation EXP/dissolve/movement dùng chung lock theo character owner. `putInToExp` nhận fraction Flash của `min(upExp-slot.exp,soulExp)`; level-up chỉ tiêu phần còn thiếu, yêu cầu exact same-family `sid+1`, rồi reset invested EXP. Level 10/catalog drift bị từ chối trước debit.
- Dissolve bag trả `template.exp + slot.exp` và `template.chip`; quick mode chỉ lấy unlocked row có `color < threshold` với threshold `1..3`. Temp panel có single/threshold bulk qua `transformExpInPanel`.
- Equipped Soul dùng exact `TBL_PET_SOUL.propType/propVal`; integer base stat và decimal feature property được cộng trước `RecalculateStats`, giữ decimal trong `soulProp<id>`/canonical property và đưa vào `EffectiveStats` khi có mapping. Row `751..760` canonical là `Vĩnh Sinh`, mô tả `Tăng Miễn Tử <propVal>`.
- `moveSoul` type 2–4 thành công phải reload Pet qua service sau persist để Soul provider được enrich lại, rồi phát `onRefreshPetProp`; move type 1 chỉ giữa các slot nhân vật không refresh Pet.

## Văn Chỉ Pet PVE

- Văn Chỉ là `PPVEState.KP`, không phải character currency `CurrencyPvePoint=238`. Box award type 35 dùng reserved ID `242` và phải được intercept trước generic character-currency lookup.
- Grant tải và persist toàn bộ PPVE state dưới mutation lock theo character. Persistence lỗi phải trả lỗi trước khi box bị consume; amount không dương hoặc overflow fail closed.
- Callback sau grant thành công là `onUpdatePPVEPanel(kpData, ppveData, nil, false)`. Không phát `onAddMoney` cho Văn Chỉ.

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
