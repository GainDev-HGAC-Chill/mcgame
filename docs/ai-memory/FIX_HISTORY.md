# MCGame Fix History

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

## FIX-2026-10-14-14 — 52 họ skill chủ động Pet dùng sai damage/effect

- Nguyên nhân: adapter dùng `expSkill`/`dexSkill` progression làm combat formula, buff ID ngoài 1..10 bị bỏ qua, skill support vẫn đi damage path và thiếu cơ chế battle-local cho stat/periodic/shield/control/special.
- Sửa: thêm registry family+level phủ 144 template, domain `SkillPlan`/effect engine, effective-stat overlay, shared shield/invulnerability damage path, target override/dead ally/taunt và các cost/revive/dispel/transfer/rest/sacrifice đã duyệt.
- Test: registry khớp chính xác 144 template/52 family; focused application/domain PASS; `go vet ./...` PASS; build `bin/gameserver` PASS. Broad internal regression chỉ còn baseline ngoài phạm vi ở item/Pet/chat.
- Trạng thái: CODE_FIXED — chưa Flash smoke-test/deploy; không restart server và không đổi dữ liệu production.
