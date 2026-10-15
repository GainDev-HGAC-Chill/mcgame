# MCGame Fix History

## FIX-2026-10-15-21 — Thuốc Tăng Trưởng Pet không cộng EXP

- Hiện tượng: dùng Thuốc Tăng Trưởng lên Pet không thay đổi kinh nghiệm/cấp Pet.
- Nguyên nhân: cả tám template là `type=501`, `useType=2` nhưng server route vào HP/MP potion; RTMP handler chỉ truyền `PetID` cho type507. Lượng EXP không nằm ở `i1/i2`, mà ở exact Flash registry `PET_GROW_ITEM_NUM`.
- Sửa: exact-ID growth effect cho 961/962/964/1585/1586/1587/1836/3339 với 30k/150k/300k/300k/2m/6.4m/12.5m/100m EXP; validate Pet target/ownership/Pet level/max level, persist qua `pet.Service.GainExperience`, hỗ trợ single/multi count và phát `exp`, changed `level`, `onRefreshPetProp`.
- Test: exact registry, quantity, Pet-vs-character level gate, wrong target, requested handler flow/item consumption và progress callbacks PASS; item/Pet packages, targeted vet và gameserver build PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test; không thay đổi dữ liệu Pet/người chơi thật.

## FIX-2026-10-15-20 — Item sự kiện ẩn và túi tiện dụng route nhầm

- Hiện tượng: item từ Shop/sự kiện đôi khi chỉ hiện sau khi sắp xếp túi; kéo vật phẩm vào túi tiện dụng báo `Internal error: cannot equip item`.
- Nguyên nhân: ba activity reward callback gửi raw Go `*Item` thay vì Flash slot DTO. `addtItemFromBag`/`addmxFromBag` bị đăng ký vào generic `MoveItem`, khiến target position 1..22 bị parse thành equipment SID.
- Sửa: chuẩn hóa activity notice + `BuildClientItemDTO`; giữ shop changed-stack snapshot hiện có. Thêm handler riêng decode page 24 ô, map normal index 1-based/MX 101-based, validate capacity/type, xóa icon nguồn và refresh `updateTempBags`; temp sort giữ cùng index contract. Follow-up sửa `tempList` từ inventory DTO sang compact `{t,n,b,c}` để item đã chuyển render ngay thay vì trông như bị mất. Hỗ trợ alias `tempToBag`/`mxtempToBag` và đúng wire RPC `addItemFromTemp(position,targetSID[,count])`/case-sensitive `addMXFromTemp`; full/partial stack dùng SID đích, không swap item không tương thích vào temp bag.
- Test: exact normal/MX compact payload, actual runtime args `[1,2106]`, move-out normal/MX, requested destination/no-swap, item application và item handler PASS; targeted vet và gameserver build PASS. Broad internal trước đó có lỗi registry Quest 234-vs-227 từ thay đổi quest ngoài phạm vi và 2 chat-stat baseline; Pet suite hiện PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-15-19 — Đồng bộ tỷ lệ và hệ số cường hóa Pet

- Hiện tượng: server luôn nâng thành công tới 12 sao dù Flash hiển thị tỷ lệ giảm dần; tổng hệ số đã lưu đúng, ví dụ `1.56 + 0.40 = 1.96`, nhưng bảng Pet vẫn có thể hiện `1.56`.
- Nguyên nhân: application không roll xác suất; responder chỉ cập nhật `growRateAdd` ở root Pet trong khi `PetPanel`/`PetManagerPanel` tính từ nested `property.growRateAdd`.
- Sửa: dùng exact base rates Flash `100/100/90/80/70/60/50/30/30/30/30/20%` cho sao đích 1→12. Failure giữ sao dưới 8; từ 8+ về 0 nếu không có Nội Đơn Chúc Phúc hợp lệ, có protection thì giữ sao và consume item. `petStar`/`petStarClear` đồng thời phát `onRefreshPetProp({id,s})` từ DTO mới nhất.
- Test: exact rate table, deterministic success/failure, mốc 8 có/không protection, response `flag/starSafeNum`, regression `1.56 + 0.40 = 1.96`, Pet handler/domain, targeted vet và gameserver build PASS. Broad internal giữ nguyên baseline 4 Pet + 2 chat-stat.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-14-18 — Hoàn tất registry nhiệm vụ đối thoại

- Hiện tượng: completion policy chỉ cho phép 67 quest type 1 theo ID, nên 160 quest đối thoại type 2/4 đã audit vẫn luôn state 103 và không thể turn-in.
- Sửa: thay allowlist ID-only bằng registry 227 fingerprint exact `ID/name/type/startNPC/finishNPC`; chỉ cho phép khi không có ordinary requirement, `rt/rn=0`, dialogue đầy đủ và finish NPC dương. Quest 46 được giữ nguyên; catalog drift và nhóm mơ hồ tiếp tục fail closed.
- Audit: `cmd/questaudit` kiểm tra type counts, fingerprint/NPC/dialogue drift, requirement/currency overlap và exact policy partition. Checked catalogs báo 227 (67/67/93), zero diagnostics; partition 3682 catalog, 227 conversation, 90 special, 4 missing-target, 215 currency, 1312 unconfigured.
- Test: focused quest/handler/NPC/audit và targeted vet PASS; audit registry test chỉ dùng catalog checked-in, không phụ thuộc `.local`; broad internal chỉ còn 4 baseline Pet + 2 chat-stat; `questaudit` và `gameserver` build PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test; không migration và không thay đổi player data.

## FIX-2026-10-14-16 — Phiếu lễ và Thẻ Kim Phiếu

- Hiện tượng: `Ngân Phiếu Lễ` 654, `Hoàng Kim Lễ` 1064 và `Thẻ Kim Phiếu` 3848 đi vào generic type-550 và không đổi được tiền theo mô tả.
- Sửa: application path riêng cho đúng ba template; 654 cộng 10.000 `moneyBind`, 1064 cộng 50 `goldBind`, 3848 chọn ngẫu nhiên 88/188/288/588/888/1888 `goldBind`; hỗ trợ dùng đơn/nhiều và callback `onAddMoney`.
- An toàn: validate template/level/target/count/overflow; lỗi persist ví sau consume dừng trước success callback và không tự replay.
- Test: focused application và toàn bộ item handler PASS; targeted vet/build PASS.
- Trạng thái: CODE_FIXED — chưa deploy hoặc Flash smoke-test.

## FIX-2026-10-14-17 — Quest Admin không cần đăng nhập

- Hiện tượng: giao diện chỉnh sửa nhiệm vụ phụ thuộc Supabase Auth/admin session.
- Sửa: Quest API cấp editor identity nội bộ cố định, không gọi Auth; PATCH vẫn bắt buộc exact same-origin. Validation/revision/active-use/audit giữ nguyên.
- Test: quest editor/validation/permission tests và dashboard typecheck PASS; production build compile/typecheck xong nhưng copy standalone bị Windows symlink `EPERM`.
- Trạng thái: CODE_FIXED — chưa deploy/smoke-test dashboard.

## FIX-2026-10-13-01 — Shop điểm thưởng

- Hiện tượng: WebPanel hiện có điểm nhưng System Shop báo `not enough points`.
- Nguyên nhân: Shop kiểm tra/trừ `Character.Honor`, trong khi WebPanel/currency registry dùng `ExPoint` (`currency_type=223`).
- Sửa: `internal/application/shop/service.go`, `handlers/shop/config.go`; dùng `ExPoint`, callback key `exPoint`, khóa purchase theo character.
- Test: shop package PASS; test concurrent chỉ một purchase thành công; build/vet PASS. `-race` chưa chạy vì môi trường tắt CGO.
- Deploy: rebuild/restart gameserver; không migration.
- Trạng thái: PARTIAL — chưa có transaction DB chung cho ví và inventory, chưa live-test Flash.

## FIX-2026-10-13-02 — Dung hợp Pet tiêu hao sai

- Hiện tượng: gửi 1 chính + 4 phụ nhưng server chỉ xử lý một Pet phụ; UI chỉ đúng sau relog.
- Nguyên nhân: handler chỉ đọc hai args; service xóa từng Pet và handler không phát callback đầy đủ.
- Sửa: `application/pet/service.go`, `domain/pet/repository.go`, cached/postgres pet repository, `handlers/pet/fusion.go`.
- Cách sửa: validate đủ/không trùng/đúng owner; transaction update chính + delete phụ; callbacks `onDelPet`/`onUpdatePet`.
- Test: success/failure/replay focused tests, cache/postgres tests, build/vet PASS.
- Deploy: rebuild/restart; không migration.
- Trạng thái: CODE_FIXED — chưa kiểm chứng trực tiếp trong Flash.

## FIX-2026-10-13-03 — Item luyện Pet không cập nhật

- Nguyên nhân: nhiều RPC bỏ qua item instance hoặc chỉ đổi Pet, không trừ stack/không trả field mà responder Flash đọc.
- Sửa: `application/pet/item_consumption.go`, `service.go`, `handlers/pet/enhancement.go`, `fusion.go`, `helpers.go`.
- Cách sửa: validate owner/bag/stack, tiêu hao một lần, rollback Pet khi consume lỗi, trả remaining stack và callback inventory.
- Test: XSD consume/rollback tests, handler package, build/vet PASS.
- Trạng thái: PARTIAL — cần live-test từng tab và vật phẩm bảo hộ tăng sao.

## FIX-2026-10-13-04 — Tiêu Dao Phù thời hạn bị mất

- Nguyên nhân: transport dùng helper luôn giảm stack, không phân biệt `TBL_ITEM_TEMPLATE.t > 0`.
- Sửa: `application/item/timed_reusable.go`, `handlers/item/transport.go`.
- Cách sửa: kích hoạt expiry một lần trong item properties; tái sử dụng không gia hạn/không tiêu hao; item `t <= 0` vẫn tiêu hao.
- Test: timed reuse/no-extension và permanent consume PASS; item handler/build/vet PASS.
- Trạng thái: CODE_FIXED — chưa live-test; disconnect mới được điều tra trong phiên hiện tại.

## FIX-2026-10-13-05 — Túi tiện dụng 20/50 ô

- Nguyên nhân: `SetTempBag` là handler chặn chủ đích; payload không lưu bag instance/capacity/expiry.
- Sửa: `application/item/tempbag_install.go`, `tempbag_login.go`, `handlers/item/tempbag_install.go`, sort filter.
- Cách sửa: lưu bag đã lắp bằng item slot nội bộ, capacity theo `proplNum`, expiry theo `t`, block thay túi nếu làm ẩn contents.
- Test: install/replace/expiry/login payload/content-safety/sort PASS; build/vet PASS.
- Trạng thái: PARTIAL — chưa xác minh RPC tháo túi và chưa live-test Flash.

## FIX-2026-10-13-06 — Shop phát lại thông báo nhận quest

- Nguyên nhân: mọi purchase gọi full `PushQuestProgress`; callback `onAddChaQuest` luôn hiện toast nhận quest trong `QuestPanel.as`.
- Sửa: `handlers/shop/`, `handlers/quest/notify.go`.
- Cách sửa: refresh objective/NPC state im lặng, không phát `onAddChaQuest`.
- Test: silent refresh không phát callback add-quest; handler/build/vet PASS.
- Trạng thái: CODE_FIXED — chưa live-test.

## FIX-2026-10-12-QUEST166 — Bắt Nấm Lục

- Quest 166 yêu cầu pet template 56, kind 3/type 12/q 1/num 1.
- Pet của character 6 đã tồn tại và không được chỉnh sửa/tiêu hao khi điều tra.
- Nguyên nhân hiển thị 0/1: capture lưu Pet nhưng thiếu quest refresh callback.
- Sửa: `handlers/combat/commands.go`; test reward-count và focused combat/quest PASS.
- Trạng thái: CODE_FIXED — binary host đã build, container vẫn chạy bản cũ; cần relog và capture mới sau khi được phép restart.

## FIX-2026-10-12-QUESTADMIN — Quest Admin

- Dashboard/API/schema/audit/revision/active-use guard đã triển khai; auth tests, typecheck, dashboard build và focused Go tests đã PASS ở checkpoint trước.
- Flash catalog activation vẫn `database-only`; không có publication/rebuild an toàn.
- Hai migration local đã áp dụng nhưng checkpoint ghi migration history chưa đăng ký.
- Trạng thái: PARTIAL/BLOCKED cho Flash activation; không replay DDL hoặc sửa dữ liệu người chơi.

## FIX-2026-10-13-07 — Nút mua găng thu thập báo method not found

- Client gọi fire-and-forget `clickShop(275)`; 275 là NPC Tiệm Tạp Hóa Ông Mộ, có `shop_id=32`, không phải shop/slot ID.
- Nguyên nhân: server thiếu registration/handler. Shop 32 đã có găng 1123–1126; luồng mua chuẩn xử lý giá, tiền và inventory.
- Sửa: `handlers/shop/convenience.go`, `handler.go`, wiring `cmd/gameserver/main.go`; tra NPC game-data rồi push `onOpenShop(32)`. Không phát item miễn phí.
- Test: route 275→32, unknown NPC, equipment position 13 callback; shop/item handlers, vet và build PASS.
- Trạng thái: CODE_FIXED — cần Flash smoke-test mở shop, mua và trang bị/thu thập.

## FIX-2026-10-13-08 — Túi cứu thương HP/MP báo method not found

- Client gọi `buyBloodBag({bt,bn})`; loại 1..4, cap 10,000,000, giá `ceil(bn/5000)` Gold, callback `setBloodBag("|...|")`.
- Nguyên nhân: server thiếu toàn bộ RPC/state/login payload; `ep` luôn là zero.
- Sửa: `application/bloodbag/`, shop convenience handler, `FeatureBloodBag`, stat-feature login `ep`, PostgreSQL transaction Gold + feature state.
- Test: bốn state mapping, giá boundary, cap, thiếu Gold, rollback, concurrent replay, atomic repository path và login/stat-feature packages PASS.
- Migration: không; dùng `player.character_wallet` và `player.character_stat_features` hiện có.
- Trạng thái: CODE_FIXED — chưa live-test mua đầy/một phần và tự hồi phục sau battle.

## FIX-2026-10-13-09 — Tiêu Dao Phù 7/30 ngày gây disconnect

- Log xác nhận client đóng kết nối ngay sau `useTransport`; không có panic/server crash. Bản deploy cũ đi qua consume path cho item thời hạn.
- Sửa code hiện tại giữ item `T>0`, lưu expiry một lần, không gia hạn khi reuse, không tái kích hoạt item hết hạn; ưu tiên pass thời hạn trước item dùng một lần cùng nhóm.
- Test: 7 ngày, 30 ngày, single-use, priority, expiry/no-reactivation và item handler PASS.
- Trạng thái: CODE_FIXED — nguyên nhân client exception cụ thể UNKNOWN; cần smoke-test Flash sau deploy để xác nhận không còn disconnect.

## FIX-2026-10-13-10 — Pet fusion quality/tooltip/random

- Client contract: 1 chính + 4 phụ cùng template/cùng tier; success rates 100/80/50/30; responder `{flag,pid,grow}`.
- Nguyên nhân tooltip: root growth đổi nhưng nested `property` không refresh. Runtime #1063 xác nhận `CallBack.onRefreshPetProp` nhận đúng một wrapper `{id,s}`, không phải hai positional args. Fusion trước đó còn cộng phần tăng bậc vào `GrowRateAdd`, khiến client hiển thị `1.50 + 0.11` thay vì phẩm chất cơ bản tím.
- Sửa: validate template/tier, chặn max tier, giữ random success; khi thành công đặt `GrowRate` cơ bản ngẫu nhiên theo từng 0.01 trong bậc kế tiếp và không đổi `GrowRateAdd`; DTO tách base/add; callback dùng wrapper `{id,s}`.
- Hồi quy bổ sung: build cũ từng lưu phần tăng bậc vào `GrowRateAdd` (điển hình `1.50 + 0.11`). Fusion thành công nay đặt `GrowRateAdd=0` để không giữ lại hệ số cường hóa giả trên Pet đã bị dữ liệu cũ làm bẩn.
- Test: failure/success/replay, incompatible materials, max tier, rates, DTO growth contract, pet handler/cache/postgres focused tests PASS.
- Trạng thái: CODE_FIXED — chưa live-test tooltip/fusion animation trong Flash.

## FIX-2026-10-13-11 — Nâng cấp kỹ năng Pet báo method not found

- Nguyên nhân: Flash gọi fire-and-forget `petUpSkill(petID, slot)` nhưng dispatcher chưa đăng ký RPC.
- Sửa: thêm application flow tìm skill cùng `codeName` ở level kế tiếp, tiêu hao manual theo phẩm chất 1365/1366/1367 (level 1→2: 1; level 2→3: 3), lưu Pet và gửi callback inventory/Pet.
- Test: focused upgrade service và toàn bộ pet handler PASS; targeted vet/build PASS.
- Trạng thái: CODE_FIXED — chưa smoke-test Flash và chưa kiểm chứng transaction thật khi persistence item thất bại.

## FIX-2026-10-13-12 — Đặt skill Pet vào auto gây #1010

- Nguyên nhân: `PetManagerPanel.onInitViewPetMngP` bỏ qua `data.pi`/`data.soulInfo` khi merge và chờ hai object này ở wrapper root; DTO server chỉ trả `{data: ...}`, làm `Core.updatePetSetting` đọc `pi` undefined.
- Sửa: `Pet.ToDTO` giữ `pi` và `soulInfo` ở wrapper root, đồng thời bổ sung `bt8` vào payload `pi`.
- Test: DTO contract và toàn bộ pet handler PASS; targeted vet/build PASS.
- Trạng thái: CODE_FIXED — chưa kéo-thả skill trong Flash sau deploy.

## FIX-2026-10-13-13 — Nút xóa nhân vật bị vô hiệu hóa

- Working tree hiện có implementation server schedule/restore/purge và các artifact client/SWF từ một luồng khác; không ghi đè hoặc nhận là thay đổi của nhóm này.
- Test server: auth handler và PostgreSQL package PASS; build PASS.
- Trạng thái: PARTIAL/UNVERIFIED — client source có thay đổi ngoài phạm vi và chưa smoke-test; purge mặc định tắt.

## 2026-10-13 - Immediate Character Delete / 30 FPS Shared Link
- Added immediate delete RPC using ownership-guarded SQL, secondary MD5 validation and cache invalidation.
- Kept GameLoaders30.swf public URL, using fresh internal main asset 32 to bypass old main cache.
- Migration and cleanup are staged; live rollout must be independently verified.

## FIX-2026-10-14-14 — 55 họ skill chủ động Pet dùng sai damage/effect

- Nguyên nhân: adapter dùng `expSkill`/`dexSkill` progression làm combat formula, buff ID ngoài 1..10 bị bỏ qua, skill support vẫn đi damage path và thiếu cơ chế battle-local cho stat/periodic/shield/control/special.
- Sửa: thêm registry family+level phủ 153 template/55 family, domain `SkillPlan`/effect engine và các primitive đã duyệt. Bổ sung chín template nguồn Pet đặc thù: `Hổ Khiếu Sơn Lâm`, `Thợ Săn Sát Thủ`, `Tứ Phương Khiếp Đảm`; thêm Accuracy modifier, random-target RNG battle-local và remove-all-beneficial effect.
- Test: registry khớp chính xác 144 template sách Pet cộng 9 template đặc thù; focused application/domain PASS; `go vet ./...` PASS; build `bin/gameserver` PASS. Broad internal regression chỉ còn baseline ngoài phạm vi ở item/Pet/chat.
- Trạng thái: CODE_FIXED — chưa Flash smoke-test/deploy; không restart server và không đổi dữ liệu production.

## FIX-2026-10-14-15 — Hệ số cường hóa/tooltip Pet và Mảnh Pasar

- Nguyên nhân cường hóa: `UpgradeStarWithItem` chỉ tăng `UpgradeNum`, không đọc `proplNum` của Nội Đơn nên `GrowRateAdd` luôn giữ nguyên.
- Nguyên nhân tooltip: ô Pet gọi `gdc(39, petId)` nhưng server trả dữ liệu rỗng; `TipCre` còn cần năm trường `att*` ở root bên cạnh `skill1..skill15`.
- Nguyên nhân Pasar: template 5351 đi vào generic type-550 item effect rồi trả `cannot use item`.
- Sửa: validate Nội Đơn type 507/useType 4/propType 3 và cộng `proplNum/100`; nối `gdc` Pet với detail DTO có ownership; chặn template 5351 thành silent no-op trước generic dispatch, không tiêu hao.
- Test: toàn bộ 10 Nội Đơn checked-in; DTO/`gdc` tooltip; application + handler Pasar; toàn bộ gamedata/item handler và domain Pet PASS. Đã bổ sung import/setter/wiring Buff Service cho combat handler; `go vet ./...` và build `bin/gameserver` PASS. Diff-check các file phạm vi PASS; `item/service.go` hiện còn trailing whitespace ở hunk encounter-book ngoài luồng.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test; không thay đổi dữ liệu production.
