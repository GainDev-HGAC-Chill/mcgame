# Current Work State

## Mục tiêu phiên

- Kiểm tra damage thật, target AoE và số sát thương hiển thị của toàn bộ Pet active skill; mọi hit đã trừ HP phải có playback damage cho đúng target.
- Sửa `MCGAME.bat` Start bị Supabase CLI từ chối migration out-of-order và thêm lựa chọn mở Flash 30 FPS ngay trong menu chính.
- Sửa sáu nhiệm vụ `Thăng Cấp Trung Cấp 3-1` để nhân vật đúng class, đang ở bậc Sơ Cấp và đứng tại Khảo Quan Trung Cấp có thể hoàn thành.
- Cho phép Quest Admin dashboard chỉnh toàn bộ cấu hình nhiệm vụ dù đang có người chơi nhận, vẫn giữ revision/validation/snapshot/audit và không tự sửa tiến độ người chơi.
- Triển khai kế hoạch đã duyệt sửa toàn bộ boss bản đồ, phần thưởng Sổ Tay Ma Thú và boss loot theo mô tả/cấp boss.
- Hoàn tất bộ nhớ AI và bốn lỗi: găng thu thập, túi cứu thương HP/MP, timed transport disconnect, Pet fusion quality/tooltip/random.
- Tiếp tục nhóm lỗi mới: callback/fusion Pet, nâng skill Pet, auto-skill, ranking, Quest Admin, xóa nhân vật và túi chế tạo.
- Kiểm tra toàn bộ skill chủ động Pet theo target count, damage/heal và effect so với mô tả.
- Hoàn tất đúng nhóm voucher `Ngân Phiếu Lễ`/`Hoàng Kim Lễ`/`Thẻ Kim Phiếu` và mở trực tiếp giao diện Quest Admin dashboard không cần phiên đăng nhập; không mở rộng sang item hay trang quản trị khác.
- Đã triển khai completion policy cho registry fingerprint 227 nhiệm vụ đối thoại đã duyệt (67 type 1, 67 type 2, 93 type 4); chưa thay đổi dữ liệu nhiệm vụ hoặc tiến độ người chơi.
- Đã sửa đồng bộ cường hóa Pet: `petStar` roll đúng bảng Flash 100/100/90/80/70/60/50/30/30/30/30/20% cho sao đích 1→12, hỗ trợ Nội Đơn Chúc Phúc giữ sao khi thất bại từ 8+, và phát refresh nested `property` để bảng Pet dùng cùng `GrowRate + GrowRateAdd`; chưa deploy/Flash smoke-test.
- Đã sửa inventory: `addtItemFromBag`/`addmxFromBag` dùng handler riêng để decode vị trí túi tiện dụng thay vì parse thành equipment SID; sort temp/MX giữ index 1-based. Reward sự kiện không còn gửi raw Go `*Item` vào `onAddCharactorSlot`, mà gửi Flash slot DTO đầy đủ; shop path đã có changed-stack DTO snapshot đúng.

## Đã hoàn thành

- Pet skill damage/playback: xác định plan executor đã trừ HP đúng nhiều target nhưng chỉ sinh skill/death action, thiếu Hurt/Idle. `BuildPlaybackRounds` xóa `tSObj.hHp` khỏi AoE opener rồi không có Hurt round để gắn lại, gây quái mất HP/chết mà không hiện số; khi chỉ có một target thì không qua cluster nên hiện bình thường. Đã bổ sung canonical Hurt và Die/Idle sau mọi direct/fixed Pet damage hit dương; pure support và miss vẫn không giả damage. Area resolver cũng lọc target chết trước khi cap, tránh slot chết gần làm mất target sống xa hơn.
- Launcher: `cmd/game` luôn truyền `supabase db push --include-all`, nên migration local chưa có trong history dù timestamp cũ hơn migration cuối vẫn được áp mà không reset/reseed database hiện hữu. Wrapper không còn báo nhầm là thiếu CLI cho mọi lỗi push. `MCGAME.bat` thêm mục `F` gọi command `client30`; launcher 30 FPS riêng vẫn được giữ nguyên.
- Quest Trung Cấp 3-1: audit xác định đủ sáu ID `143/287/639/640/643/644` đều là type 1, không có requirement row nên bị completion authority fail-closed. Đã thêm exact fingerprint policy riêng, yêu cầu đúng class, `class_rank == 1` và ở gần NPC 600; rank thấp/cao hơn, sai class/map hoặc catalog drift tiếp tục bị chặn. Persisted objective cũ bị loại và bước 3-1 không tự tăng class rank.
- Quest Admin active-use: đã gỡ khóa structural fields/nút lưu trên trang quest, đổi cảnh báo thành cho phép lưu có rủi ro, và thêm migration thay `quest_admin_save` để bỏ riêng guard `ACTIVE_QUEST_IN_USE`; guard loop dùng chung giữ nguyên. Không tự sửa/reset progress. Migration chưa apply local vì `supabase-db` đang `Exited (0)` và assistant không tự restart.
- Boss map runtime follow-up: log click NPC 707 lúc 07:22 xác nhận `clickNpc` đã vào route mới nhưng connection ở channel 0; validation nhầm chặn `channelID <= 0` dù DB có state `(707,0,is_alive=true)`. Đã cho phép channel 0 đối với scope `all`, vẫn chặn channel âm và vẫn giữ `channel:1` chỉ nhận channel 1.
- Boss map implementation bước 1: schedule chỉ quản lý `ground/flying` normal, có validate map/channel/alive; scene có thể inject boss registry bị thiếu khỏi static map; `clickNpc` được chuyển sang shared combat path cho schedule boss, dựng toàn roster `TBL_NPC_CREATURE`, gắn `BossCtx` schedule và kill hook ưu tiên NID canonical.
- Boss reward bước 2: đã thêm generator `cmd/seed-boss-loot` và sinh `supabase/seeds/data_tbl_schedule_boss_loot.sql` gồm 862 reward row phủ 150 registry boss theo profile ground/flying/mythic và sáu dải level. Generator đã validate item/equipment ID với local catalog. Chưa apply seed vào local DB tại checkpoint này.
- Boss reward bước 3: đã apply seed vào compose DB local sau checkpoint; xác minh 150/150 boss có 3–7 loot row, tổng 862 và 0 reference item/equipment lỗi. Không chạm player data, chưa restart/deploy.
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

- Pet skill damage/playback: full `internal/domain/combat`, `internal/application/combat` và combat handler PASS; AoE damage+dispel ba target đều mất HP và có popup riêng; fixed lethal AoE có popup trước death cho từng target; dead-slot target-cap regression PASS; pure support vẫn không gây damage; legacy multi-target playback vẫn PASS. Catalog invariant, targeted vet, build `bin/gameserver.exe` và scoped diff-check PASS.
- Launcher: `go test ./cmd/game`, `go vet ./cmd/game`, build `bin/game.exe` PASS; CLI `db push --include-all --dry-run` xác nhận chỉ pending `20261016040000_findback_bounty_verification.sql` và `20261016090000_sth_progress.sql`, seeds rỗng; hai asset Flash 30 FPS tồn tại; scoped diff-check sạch.
- Quest Trung Cấp 3-1 focused policy/runtime regressions PASS cho đủ sáu class, gồm completion thật và xác nhận rank vẫn là 1 sau bước 3-1; quest/NPC handler PASS; targeted vet và build `bin/gameserver.exe` PASS; scoped diff-check sạch (chỉ cảnh báo line-ending). Full application quest và `cmd/questaudit` vẫn fail baseline conversation registry `234 want 227` từ thay đổi ngoài phạm vi.
- Quest Admin active-use: dashboard `corepack pnpm typecheck` PASS; quest editor/validation/permission tests 20 PASS, 1 live test SKIP; scoped Prettier và diff-check PASS.
- Sau fix channel 0: domain/application scheduleboss, application combat, combat/NPC/scene handlers PASS; targeted vet PASS; `go build -o bin/gameserver.exe ./cmd/gameserver` PASS.
- Sau bước battle routing boss: domain/application scheduleboss, application combat, scene/NPC/combat handler packages đều PASS.
- `go test ./cmd/seed-boss-loot` PASS; generator chạy thành công với local compose catalog và sinh 862 row.
- Targeted boss packages + generator PASS; targeted `go vet` PASS; `go build -o bin/seed-boss-loot.exe ./cmd/seed-boss-loot` và `go build -o bin/gameserver.exe ./cmd/gameserver` PASS; scoped/full `git diff --check` sạch (chỉ có cảnh báo line-ending).
- Broad `go test ./internal/... -count=1` không có lỗi boss; còn lỗi ngoài phạm vi ở item award quality, Quest registry 234-vs-227, 2 chat-stat, và một center timing test đã PASS ngay khi chạy riêng.
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
- Đã code-fix Thuốc Tăng Trưởng Pet theo exact Flash registry `PET_GROW_ITEM_NUM`: 961=30k, 962=150k, 964/1585=300k, 1586=2m, 1587=6.4m, 1836=12.5m, 3339=100m. Type501 exact IDs nay nhận Pet target, kiểm tra level Pet (không nhầm level nhân vật), cộng/persist EXP qua Pet service, hỗ trợ single/multi count, không tiêu hao khi sai target/thiếu level/max level, và callback exp/level/property; item/Pet packages, targeted vet và gameserver build PASS.
- Đã chỉnh failure cường hóa theo xác nhận operator: dưới 8 sao giữ sao/hệ số; từ 8+ không protection thì sao và `GrowRateAdd` cùng về 0 rồi recalculate stats; có protection giữ cả hai. Pet application/domain/handler tests, targeted vet và gameserver build PASS; `petStarClear` thủ công không đổi.
- Đã code-fix ba item runtime từ log 2026-10-15: Sách Triệu Hồi 2514/2543/2798 cấp đúng timed buff hiển thị 1709/1979, chọn Pet đã khóa đúng ngưỡng cam/tím, persist follow và broadcast scene; không có Pet hợp lệ thì không tiêu hao. Sách Mật Đạo 128/2137 chặn group/battle trước tiêu hao rồi đưa vào Mật Đạo 1/map 140106. `useIntimacyItem` 32/1302/1303/1315 nay validate exact 60/660, giới tính Hoa Hồng, quan hệ bạn bè, persist hai chiều khi có row đối ứng, refresh stack và `onInitViewImC`. Đồng thời sửa snapshot stack callback của `addPopNum`/`delPopNum`, là wire Hoa/Trứng Gà; nhấp đôi 724/783 không consume và chỉ dẫn dùng nút mục tiêu. Item/social/Pet application/domain/handler tests, targeted vet và gameserver build PASS; chưa deploy/Flash smoke-test.
- Broad internal sau nhóm inventory: Pet suite PASS; còn 2 chat-stat baseline và Quest registry count `234 want 227` do thay đổi Quest ngoài phạm vi đang có trong working tree (`internal/application/quest/service.go` cũng có thay đổi người dùng). Không sửa/revert nhóm Quest đó.

## Runtime/database

- Trước khi thử migration launcher: compose DB đang chạy; history đã có đến `20261016073610`, còn thiếu out-of-order `20261016040000_findback_bounty_verification.sql` và migration mới hơn `20261016090000_sth_progress.sql`. Chưa apply hai migration này trong bước code/checkpoint; không reset hoặc sửa player data.
- `bin/gameserver.exe` đã build gồm fix channel 0; container `mcgame-server` chưa được assistant restart và cần deploy/restart được cho phép để nhận fix mới.
- Compose DB local có 862 row `data.data_tbl_boss_loot` từ seed mới; gameserver đang chạy chưa reload cache nên chưa dùng các row này. Không thay đổi bảng dữ liệu người chơi.
- Không có migration mới: blood bag dùng bảng wallet/stat-feature hiện có.
- Không live-test và không thay đổi dữ liệu người chơi.
- Nhóm voucher/Quest Admin mới không có migration hoặc thay đổi dữ liệu; chưa deploy/restart/smoke-test.

## Rủi ro/chưa xác minh

- Pet damage popup/AoE fix chưa deploy/restart hoặc Flash smoke-test; cần thử đại diện all-enemy, horizontal/cross, damage+debuff, lethal AoE, miss và DoT tick để xác nhận animation thực tế.
- Quest Trung Cấp 3-1 chưa deploy/restart hoặc Flash smoke-test; hai bước dùng chung 3-2 (645) và 3-3 (677) chưa được thay đổi vì có semantics riêng cần kiểm chứng nếu tiếp tục lỗi.
- Quest Admin active-use migration chưa được parser/live RPC verify vì local `supabase-db` dừng trước lúc apply; cần apply migration rồi chạy transactional rollback test trên một quest active không phải Quest 166 trước deploy.
- Runtime 07:22 boss 707 thất bại do channel 0 đã được code-fix và regression-test, nhưng chưa Flash retest sau khi deploy binary mới.
- Audit boss 2026-10-16 xác nhận 71 boss bản đồ và 79 boss Sổ Tay đều có roster hợp lệ, nhưng `data_tbl_boss_loot` có 0 dòng. `clickNpc` bị NPC handler ghi đè nên boss type 10 không bắt đầu trận khi click; 13 boss mythic map dùng `pos_map_id=200` nên không xuất hiện; roster map boss và định danh kill/loot hiện cũng sai. Research tại `docs/research/2026-10-16_01_MAP_BOSS_HANDBOOK_REWARD_AUDIT.md`; plan `docs/plans/2026-10-16_01_MAP_BOSS_HANDBOOK_REWARDS.md` đã được operator duyệt, chưa sửa code hoặc áp seed.
- Audit boss 2026-10-16 xác nhận 71 boss bản đồ và 79 boss Sổ Tay đều có roster hợp lệ. Research tại `docs/research/2026-10-16_01_MAP_BOSS_HANDBOOK_REWARD_AUDIT.md`; plan `docs/plans/2026-10-16_01_MAP_BOSS_HANDBOOK_REWARDS.md` đã triển khai. Code battle/spawn và reward seed hoàn tất, local DB đã apply; chưa restart/deploy hoặc Flash smoke-test nên runtime hiện tại vẫn có thể đang dùng binary/cache loot cũ.
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

1. Khi được operator cho phép deploy/restart gameserver: Flash smoke-test một quest Trung Cấp 3-1 trên nhân vật test đúng class/rank tại NPC 600, rồi kiểm tra tiếp 3-2/3-3; không dùng Quest 166 hoặc sửa tiến độ người chơi thật.
2. Khi local Supabase được operator khởi động lại: apply `20261016073610_allow_edit_active_quests.sql`, xác minh transactional rollback rằng structural patch trên quest active được chấp nhận và audit/revision vẫn hoạt động; sau đó deploy dashboard theo yêu cầu riêng.
3. Khi được operator cho phép deploy/restart: restart riêng gameserver để load binary và cache 862 loot row, rồi Flash smoke-test đại diện ground normal, flying normal, mythic Kênh 1 và Sổ Tay normal/mythic; không dùng Quest 166, Pet Nấm Lục hoặc item/người chơi thật.
4. Cần Flash smoke-test callback/fusion, `petUpSkill`, auto-skill và character-delete sau khi được phép deploy/restart riêng server.
5. Cần runtime ID cho row/template làm crash túi chế tạo và yêu cầu cụ thể cho ranking/Quest Admin trước khi triển khai tiếp.
6. Ranking cần repository/query và định nghĩa chính xác cho category tuần/tháng trước khi thay stub bằng dữ liệu thật.
7. Skill Pet cần Flash smoke-test sau khi được phép deploy/restart riêng server; kiểm tra đại diện damage đơn/AoE/multi-hit, buff, DoT/HoT, revive, shield, Nuốt Chửng và fixed-HP skill.
8. Sau khi được phép deploy, smoke-test riêng ba voucher bằng item test và mở/lưu Quest Admin khi không có session đăng nhập; không dùng vật phẩm người chơi thật.
9. Sau khi được phép deploy/restart riêng server, Flash smoke-test quest 46 cùng một quest type 2/type 4 mới tại/xa finish NPC và xác nhận một quest mơ hồ vẫn bị chặn.

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
