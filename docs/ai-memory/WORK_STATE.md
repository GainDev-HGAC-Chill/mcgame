# Current Work State

## Mục tiêu phiên

- Hoàn tất bộ nhớ AI và bốn lỗi: găng thu thập, túi cứu thương HP/MP, timed transport disconnect, Pet fusion quality/tooltip/random.
- Tiếp tục nhóm lỗi mới: callback/fusion Pet, nâng skill Pet, auto-skill, ranking, Quest Admin, xóa nhân vật và túi chế tạo.
- Kiểm tra toàn bộ skill chủ động Pet theo target count, damage/heal và effect so với mô tả.
- Hoàn tất đúng nhóm voucher `Ngân Phiếu Lễ`/`Hoàng Kim Lễ`/`Thẻ Kim Phiếu` và mở trực tiếp giao diện Quest Admin dashboard không cần phiên đăng nhập; không mở rộng sang item hay trang quản trị khác.
- Đã triển khai completion policy cho registry fingerprint 227 nhiệm vụ đối thoại đã duyệt (67 type 1, 67 type 2, 93 type 4); chưa thay đổi dữ liệu nhiệm vụ hoặc tiến độ người chơi.
- Đã sửa đồng bộ cường hóa Pet: `petStar` roll đúng bảng Flash 100/100/90/80/70/60/50/30/30/30/30/20% cho sao đích 1→12, hỗ trợ Nội Đơn Chúc Phúc giữ sao khi thất bại từ 8+, và phát refresh nested `property` để bảng Pet dùng cùng `GrowRate + GrowRateAdd`; chưa deploy/Flash smoke-test.
- Đã sửa inventory: `addtItemFromBag`/`addmxFromBag` dùng handler riêng để decode vị trí túi tiện dụng thay vì parse thành equipment SID; sort temp/MX giữ index 1-based. Reward sự kiện không còn gửi raw Go `*Item` vào `onAddCharactorSlot`, mà gửi Flash slot DTO đầy đủ; shop path đã có changed-stack DTO snapshot đúng.

## Đã hoàn thành

- Tạo và xác minh `docs/ai-memory/`; bổ sung workflow vào `CLAUDE.md`/`AGENTS.md` symlink.
- `clickShop(275)` tra NPC và mở shop 32; không mua/phát item trực tiếp.
- `buyBloodBag` validate `{bt,bn}`, cap/giá Gold, transaction DB chung Gold + reserve, callback và login `ep`.
- Timed transport giữ item 7/30 ngày, không gia hạn/tái kích hoạt, ưu tiên pass trước consumable.
- Pet fusion validate 1+4 cùng template/tier, tỷ lệ 100/80/50/30, đúng một bậc; DTO/callback tooltip đã đồng bộ.
- Đã sửa lại theo bằng chứng Flash runtime: `onRefreshPetProp` dùng một wrapper `{id,s}`; fusion đổi `GrowRate` cơ bản ngẫu nhiên trong bậc kế tiếp và không cộng fusion vào `GrowRateAdd`.
- Fusion thành công hiện xóa `GrowRateAdd` cũ để tự phục hồi Pet từng bị build cũ lưu sai `+0.11`; test đầu vào có `GrowRateAdd=0.11` PASS.
- Đã thêm RPC `petUpSkill`, tiêu hao manual 1365/1366/1367 và callback inventory/Pet.
- Đã giữ `pi`/`soulInfo` ở root wrapper Pet DTO và bổ sung `bt8`, tránh lỗi `Core.updatePetSetting` sau khi mở Pet Manager.
- Đã thêm application path riêng cho đúng template `654` Ngân Phiếu Lễ, `1064` Hoàng Kim Lễ và `3848` Thẻ Kim Phiếu; hỗ trợ dùng đơn/nhiều, kiểm tra level/target/count/overflow, tiêu hao rồi cộng đúng ví khóa và phát `granted_currencies`.
- Quest Admin API không còn yêu cầu Supabase Auth/admin session; mọi request được cấp quyền editor nội bộ cố định, còn PATCH vẫn bắt buộc same-origin.
- Audit 5.530 quest xác định 227 nhiệm vụ đối thoại tin cậy cao: 67 ID hiện tại (Quest 46 được operator xác nhận) và 160 ứng viên mới type 2/4. Registry lưu exact name/type/startNPC/finishNPC từ checked `TBL_QUEST.json`; catalog drift, requirement, `rt/rn`, thiếu dialogue/NPC tiếp tục fail closed. Giữ chặn 711 ứng viên mơ hồ, 77 ứng viên mâu thuẫn và 669 empty definition khác.

## File đang sửa/chưa commit

- HEAD `2b550d0` đã chứa nhóm Pet/blood-bag/transport/shop và bộ nhớ trước đó; không commit/push thêm.
- Working tree còn nhóm character-delete/client artifact do luồng khác để lại và cập nhật bộ nhớ phiên này. Không reset hoặc ghi đè nhóm này.

## Kiểm tra đã chạy

- Focused bloodbag/shop/statfeature/item/transport/equip/pet/fusion/domain/cache/postgres tests PASS.
- Full shop và item/pet handler packages PASS.
- Targeted `go vet` PASS.
- `go build -o bin/gameserver ./cmd/gameserver` PASS.
- `git diff --check` hiện chỉ fail vì trailing whitespace trong `docs/client/.../CharSelectCanvas.as` do thay đổi ngoài luồng; không sửa decompiled ActionScript.
- Full `internal/application/pet` còn 4 baseline failures về stat/equipment/skill; full `internal/application/item` còn baseline failures về award quality. Không sửa ngoài phạm vi.
- Focused fusion/random/DTO/`petUpSkill`/pet handler tests PASS sau khi khôi phục phiên.
- Auth handler + PostgreSQL tests PASS cho source character-delete hiện có.
- Full `internal/application/pet` chạy lại: vẫn đúng 4 baseline failures về stat/equipment/skill; các package domain/pet, pet handler, auth và postgres PASS.
- Targeted `go vet` PASS; `go build -o bin/gameserver.exe ./cmd/gameserver` PASS.
- Sau sửa stale `+0.11`: focused `TestFusePets`, toàn bộ pet handler và domain/pet PASS; diff-check hai file code sạch.
- Focused voucher application tests và toàn bộ item handler PASS; Quest editor/validation/permission tests PASS; dashboard `corepack pnpm typecheck` PASS. Go targeted vet và `go build -o bin/gameserver ./cmd/gameserver` PASS.
- Dashboard production build đã compile/typecheck và generate đủ page nhưng bước copy `.next/standalone` fail do Windows không cho tạo pnpm symlink (`EPERM`), không phải lỗi TypeScript/source. Full `go vet ./...` bị các package trong thư mục backup ngoài source `backup_bloodbag_before_atomic`; targeted vet của package sửa PASS.
- Baseline trước audit nhiệm vụ đối thoại: application quest, quest/NPC handlers và targeted vet PASS; `cmd/questaudit` build PASS. `go test ./internal/...` chỉ còn đúng 4 baseline Pet và 2 baseline chat-stat, không có lỗi quest.
- Registry policy tests, quest/quest-handler/NPC-handler focused tests và targeted vet PASS. `bin/questaudit.exe` báo đúng 227 (67/67/93), zero drift/NPC/requirement/currency/dialogue diagnostics; partition hiện tại: catalog 3682, conversation 227, special 90, missing-target 4, currency 215, unconfigured 1312.
- Broad `go test ./internal/... -count=1` sau implementation chỉ còn đúng 4 baseline Pet và 2 baseline chat-stat đã biết; toàn bộ quest/NPC packages PASS. `go build -o bin/questaudit.exe ./cmd/questaudit` và `go build -o bin/gameserver ./cmd/gameserver` PASS.
- Independent verification sửa test `cmd/questaudit` từng phụ thuộc artifact `.local/research` bị Git ignore; test registry nay chỉ đọc ba catalog checked-in `TBL_QUEST`, `TBL_QUEST_REQUIRE`, `TBL_NPC` và PASS trên focused run. Full `go vet ./...` vẫn chỉ bị thư mục backup ngoài source `backup_bloodbag_before_atomic`; targeted vet của toàn bộ package sửa PASS.
- Pet enhancement live refresh: thêm `onRefreshPetProp` sau nâng/xóa sao và regression payload `1.56 + 0.40 = 1.96`; Pet handler/domain, targeted enhancement, targeted vet và gameserver build PASS; broad internal còn đúng baseline 4 Pet + 2 chat-stat ngoài phạm vi.
- Pet star probability: bỏ hành vi thành công 100% đến 12 sao; server dùng exact base table mà Flash hiển thị. Failure giữ sao dưới 8, từ 8 trở lên về 0 nếu không có protection; Nội Đơn Chúc Phúc 2276 được validate/consume và trả `flag/starSafeNum`. Focused probability/failure/protection tests, targeted vet và gameserver build PASS; broad internal giữ nguyên baseline 4 Pet + 2 chat-stat.
- Inventory/temp bag follow-up: event rewards (`TakeNewGift`, `TakeSPGift`, `BuySendCombine`) trước đây callback raw struct nên DB có item nhưng Flash slot có thể không render tới khi sort. Đã chuẩn hóa notice + `BuildClientItemDTO`; shop snapshot path đã đúng. `addtItemFromBag`/`addmxFromBag` nay map client page 24 ô thành normal 1-based/MX 101-based, validate capacity/type, xóa source icon rồi refresh `updateTempBags`; focused item/activity/shop tests, targeted vet và gameserver build PASS.
- Follow-up ngay sau test người dùng: item chuyển vào temp bag bị ẩn vì `BuildTBagPayload` gửi inventory DTO, trong khi `TempBagSlot` đọc compact `{t,n,b,c}`. Đã đổi cả normal/MX `tempList` sang exact compact payload và thêm assertion template/stack/bind/color; focused item/activity/shop, targeted vet và gameserver build PASS.
- Chiều lấy ra từng báo `method not found`: đã đăng ký `tempToBag`/`mxtempToBag`, decode cùng page position, move về first empty regular-bag slot, gửi `onAddCharactorSlot` và full `updateTempBags`; normal/MX application regressions PASS.
- Runtime 19:13:47 xác nhận registration trước đó sai wire name: Flash thực gửi `addItemFromTemp(position, targetSID[, count])` (log args `[1,2106]`), và source Flash dùng case-sensitive `addMXFromTemp`. Đã đăng ký/triển khai cả hai, hỗ trợ full/partial stack tới SID đích, chặn occupied non-stack thay vì swap vào temp; focused application/handler regressions, targeted vet và gameserver build PASS.
- Broad internal sau nhóm inventory: Pet suite PASS; còn 2 chat-stat baseline và Quest registry count `234 want 227` do thay đổi Quest ngoài phạm vi đang có trong working tree (`internal/application/quest/service.go` cũng có thay đổi người dùng). Không sửa/revert nhóm Quest đó.

## Runtime/database

- `bin/gameserver` đã build bản mới; container `mcgame-server` chưa restart và vẫn cần deploy được cho phép.
- Không có migration mới: blood bag dùng bảng wallet/stat-feature hiện có.
- Không live-test và không thay đổi dữ liệu người chơi.
- Nhóm voucher/Quest Admin mới không có migration hoặc thay đổi dữ liệu; chưa deploy/restart/smoke-test.

## Rủi ro/chưa xác minh

- Cần Flash smoke-test cả bốn luồng; riêng transport log cũ chứng minh client disconnect nhưng không có stack trace Flash nên exception cụ thể vẫn UNKNOWN.
- Chưa kiểm chứng tự động tiêu hao reserve blood bag sau battle; RPC `useBloodBag` cũ ngoài phạm vi mua và cần contract riêng trước khi đổi.
- Shop purchase chung vẫn chưa atomic xuyên DB giữa ví và inventory; không liên quan transaction blood-bag mới.
- Quest 166/Pet Nấm Lục và Quest Admin giữ nguyên trạng thái cũ, không dùng để test.
- Ranking đã xác nhận `rankGet` là stub và nhiều category thiếu semantics; chưa sửa bằng dữ liệu suy đoán.
- Túi chế tạo crash tại template lookup của shop catalog runtime; catalog checked-in không có row hỏng nên cần ID row/template từ runtime trước khi sửa.
- Quest Admin “chưa hoàn chỉnh” chưa nêu field/workflow cụ thể; source hiện có editor và có thể deployment đang cũ.
- Audit 144 template/52 họ skill Pet tại `docs/research/2026-10-14_02_PET_ACTIVE_SKILL_AUDIT.md` đã được triển khai bằng registry/effect engine battle-local; trạng thái CODE_FIXED, chưa Flash smoke-test.
- Đã chốt: multiplier tính trên công Pet trước mitigation; Đoạt Mệnh/Liêu Hồn đơn mục tiêu và proc theo HP hiện tại mục tiêu; Nuốt Chửng xử lý một mục tiêu; mâu thuẫn data ưu tiên mô tả. Còn thiếu trị số/formula cho một số skill catalog không định lượng.
- Đã chốt nốt trị số thiếu: Đoạt Mệnh/Liêu Hồn proc cộng 5/10/15% HP hiện tại; Ác Ma Chi Kích/Lời Ngọt Ngào 100/110/120%; Hiệp Ước Tử Thần +50% công/thủ trong 3 lượt; Duy Trì Tốc Độ và Suy Yếu Linh Hồn dùng 10/20/30%.
- Đã chốt Nuốt Chửng hồi 10/15/20% max HP+MP; Tự Nhiên Thuẫn đơn/quần HoT bằng 65%/25% MagicAttack; Pháo Kích chỉ P100% + stun. Plan triển khai đã Approved tại `docs/plans/2026-10-14_01_PET_ACTIVE_SKILLS_COMBAT.md`.
- Trong lúc audit xuất hiện thay đổi ngoài luồng ở `turn_executor.go`, item/quest/EXP helpers; không ghi đè. Diff combat mới chỉ đổi cách resolve target cho đánh thường, không thay đổi kết luận skill audit.

## Bước tiếp theo

1. Cần Flash smoke-test callback/fusion, `petUpSkill`, auto-skill và character-delete sau khi được phép deploy/restart riêng server.
2. Cần runtime ID cho row/template làm crash túi chế tạo và yêu cầu cụ thể cho ranking/Quest Admin trước khi triển khai tiếp.
3. Ranking cần repository/query và định nghĩa chính xác cho category tuần/tháng trước khi thay stub bằng dữ liệu thật.
4. Skill Pet cần Flash smoke-test sau khi được phép deploy/restart riêng server; kiểm tra đại diện damage đơn/AoE/multi-hit, buff, DoT/HoT, revive, shield, Nuốt Chửng và fixed-HP skill.
5. Sau khi được phép deploy, smoke-test riêng ba voucher bằng item test và mở/lưu Quest Admin khi không có session đăng nhập; không dùng vật phẩm người chơi thật.
6. Sau khi được phép deploy/restart riêng server, Flash smoke-test quest 46 cùng một quest type 2/type 4 mới tại/xa finish NPC và xác nhận một quest mơ hồ vẫn bị chặn.

## 2026-10-14 - Pet active skill implementation checkpoint

- Đã thêm domain `SkillPlan`, catalog registry phủ 153 template/55 family và effect engine battle-local cho damage/heal, stat/control, periodic, shield, revive, dispel, transfer, cost/rest/sacrifice. Chín template `Hổ Khiếu Sơn Lâm`, `Thợ Săn Sát Thủ`, `Tứ Phương Khiếp Đảm` được bổ sung sau audit vì thuộc nguồn Pet đặc thù, không nằm trong roster sách Pet ban đầu.
- Đã nối adapter vào catalog trước legacy behavior và route skill plan qua shared damage/status handling; registry hiện khớp chính xác 144 family+level record, không nhận thừa level hoặc skill không active.
- Không chạm ba file item/quest/EXP ngoài luồng; hunk resolve target đánh thường hiện có trong `turn_executor.go` được giữ nguyên.
- Kiểm tra độc lập đã sửa thêm: bỏ level bonus khỏi base damage Pet, dùng percentage-point cho né/liên kích/bạo kích, sửa confusion redirect, Quang Hợp lấy max MP của người nhận, tách DoT vật lý/ma pháp cho shield, giữ death-immune state ở permille và áp buff Combo/Counter lên normal combat path.
- Focused `application/combat` + `domain/combat` PASS; `go vet ./...` PASS; `go build -o bin/gameserver ./cmd/gameserver` PASS. Broad `go test ./internal/...` chỉ còn baseline ngoài phạm vi ở item (6), pet (4), chat handler (2).
- Bước tiếp theo: Flash smoke-test sau khi được phê duyệt deploy/restart; chưa thay đổi dữ liệu production.
- Sửa bổ sung `Hổ Khiếu Sơn Lâm`: override thành toàn bộ địch tối đa 10, physical 80/100/120%, giải một buff có lợi với xác suất độc lập 30/40/50% trên từng mục tiêu còn sống; focused test/vet PASS.
- Sửa bổ sung `Thợ Săn Sát Thủ`: toàn đội 10, Speed/Accuracy +10/15/20% trong 5 lượt. `Tứ Phương Khiếp Đảm`: 4/6/8 địch ngẫu nhiên, cả hai Attack và Speed −5/8/12% trong 1 lượt, 10/20/30% xóa toàn bộ buff có lợi theo từng mục tiêu. Focused test/vet PASS.
- Hoàn tất nhóm Pet/inventory mới: `petStar` đọc `TBL_ITEM_TEMPLATE.proplNum` của 10 Nội Đơn checked-in để cộng `GrowRateAdd` và tính lại stats; `gdc(39, petId)` trả detail DTO Pet theo ownership với skill và năm trường `att*` root; template 5351 Mảnh Pasar bị chặn trước generic dispatch, trả no-op im lặng và không tiêu hao khi nhấp đôi. Focused/full handler tests PASS; `go vet ./...` và build `bin/gameserver` PASS sau khi bổ sung import, setter và wiring `combatHandler.SetBuffService(buffService)`. Scoped diff-check sạch; hunk encounter-book ngoài luồng trong `item/service.go` vẫn có trailing whitespace. Trạng thái CODE_FIXED, chưa Flash smoke-test/deploy.
- Broad application chạy lại: Pet vẫn đúng 4 baseline failures cũ; item còn nhóm award-quality baseline và thêm `TestOtherScrollDoesNotExpandContainers` từ thay đổi `effect_scroll.go` ngoài luồng hiện tại (`Encounter buff service unavailable`). Không sửa/ghi đè nhóm ngoài phạm vi này.

## 2026-10-13 - Immediate Character Delete / 30 FPS Shared Link
- Added immediate delete RPC using ownership-guarded SQL, secondary MD5 validation and cache invalidation.
- Kept GameLoaders30.swf public URL, using fresh internal main asset 32 to bypass old main cache.
- Migration and cleanup are staged; live rollout must be independently verified.
