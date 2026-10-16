# MCGame Fix History

## FIX-2026-10-16-28 — Không thể chủ động bỏ nhiệm vụ type 1 đã nhận

- Hiện tượng: cả Flash QuestManager và QuestPanel tự khóa nút Hủy khi active DTO có `data.type == 1`; nếu gọi RPC trực tiếp, service cũng từ chối toàn bộ type 1 bằng `ErrQuestNotAvailable`.
- Sửa: bỏ type-1 rejection, serialize abandon bằng mutation lock và vẫn yêu cầu đúng character + progress active trước khi xóa. Active type-1 DTO dùng presentation type 2 để mở nút Hủy; canonical catalog type 1 không đổi nên completion policy, prerequisite và reward vẫn giữ nguyên.
- Tradeoff client: nhiệm vụ type 1 đã nhận tạm nằm trong nhóm nhiệm vụ chính; danh sách nhiệm vụ chưa nhận vẫn giữ nhóm gốc. Không sửa ActionScript, không tạo history/phần thưởng và không thay đổi dữ liệu người chơi.
- Test: focused abandon/presentation regressions PASS; quest/NPC handler PASS; targeted vet, gameserver build và scoped diff-check PASS. Full application quest chỉ còn baseline conversation registry `234 want 227` ngoài phạm vi.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-16-27 — Pet AoE trừ HP nhưng không hiện damage hoặc chỉ tác dụng một mục tiêu

- Hiện tượng: Pet damage plan thay đổi HP thật trên server nhưng không sinh Hurt/Idle actions. AoE playback xóa `tSObj.hHp` khỏi opener để chuyển sang shared Hurt round, nhưng round đó rỗng; quái có thể chết mà không hiện số. Ngoài ra target cap được áp trước khi lọc slot đã chết, nên một slot chết gần có thể làm mất target sống xa hơn.
- Sửa: mọi direct/fixed Pet hit dương sinh chuỗi hit → Hurt → Die/Idle; mỗi target đã nhận damage có `hHp` riêng trước death. Area resolver lọc dead target trước rồi mới cap số lượng.
- Semantics giữ nguyên: miss không gây/hiện damage; pure buff/debuff và lần cast đầu của DoT-only không giả damage; DoT hiển thị khi tick; AoE theo hình học chỉ đánh các ô hợp lệ đang có mục tiêu sống.
- Test: full domain/application combat và combat handler PASS; catalog invariant xác nhận mọi direct-damage plan có hệ số và target dương; targeted vet, build `bin/gameserver.exe` và diff-check PASS.
- Trạng thái: CODE_FIXED — chưa restart/deploy hoặc Flash smoke-test.

## FIX-2026-10-16-26 — Launcher chặn migration out-of-order và thiếu mục Flash 30 FPS

- Hiện tượng: Start dừng tại `supabase db push` vì migration `20261016040000_findback_bounty_verification.sql` có timestamp trước migration cuối đã ghi trong history; thông báo wrapper còn nhầm thành có thể thiếu CLI. Menu chính không có lựa chọn 30 FPS dù command/client riêng đã tồn tại.
- Sửa: launcher truyền `--include-all`, vẫn chỉ `--include-seed` cho database mới và đổi lỗi wrapper thành thông báo trung thực. `MCGAME.bat` thêm phím `F` chạy `game client30`.
- An toàn: không reset/volume delete/reseed database hiện hữu. Dry-run xác nhận đúng hai migration pending `20261016040000` và `20261016090000`, không có seed.
- Test: `go test ./cmd/game`, `go vet ./cmd/game`, build `bin/game.exe`, asset 30 FPS và scoped diff-check PASS.
- Trạng thái: CODE_FIXED/BINARY_BUILT — chưa chạy Start thật, chưa apply migration hoặc restart services.

## FIX-2026-10-16-25 — Sáu class không hoàn thành được Thăng Cấp Trung Cấp 3-1

- Hiện tượng: quest `143/287/639/640/643/644` không có requirement row và không thuộc completion policy đã duyệt, nên luôn giữ state 103/`CompletionBlocked` dù nhân vật đã là Sơ Cấp.
- Sửa: thêm exact fingerprint policy riêng cho sáu quest; completion authority kiểm tra đúng class, `class_rank == 1` và vị trí gần Khảo Quan Trung Cấp NPC 600. Persisted objective cũ không được phép bypass điều kiện.
- Phạm vi: không tự sửa tiến độ, không tăng class rank tại bước 3-1, không suy đoán logic cho bước 3-2/3-3.
- Test: focused policy/runtime regression đủ sáu class PASS; quest/NPC handler PASS; targeted vet và build `bin/gameserver.exe` PASS. Full quest/questaudit vẫn fail baseline ngoài phạm vi vì conversation registry hiện có 234 entry trong working tree nhưng assertion cũ chờ 227.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-16-24 — Quest Admin cho sửa nhiệm vụ đang được nhận

- Hiện tượng: dashboard chỉ cho sửa tên/mô tả/hội thoại và khóa mục tiêu, NPC, thưởng, requirements/prerequisites khi có `character_quests` active; DB RPC cũng trả `ACTIVE_QUEST_IN_USE`.
- Sửa: trang quest không còn khóa structural fields hoặc nút lưu theo active count; cảnh báo vẫn hiển thị rõ thay đổi có thể ảnh hưởng đối chiếu runtime. Migration thay `public.quest_admin_save` để bỏ active-use rejection, giữ revision conflict, validation, transactional before-image/audit, child ownership và cycle detection. Guard của shared loop không đổi.
- An toàn: không tự bỏ/reset/bồi thường tiến độ người chơi; save vẫn yêu cầu reason, request UUID và exact revision.
- Test: dashboard typecheck PASS; editor/validation/permission 20 PASS, 1 live test SKIP; Prettier scoped và diff-check PASS.
- Trạng thái: CODE_FIXED/MIGRATION_STAGED — chưa apply local vì `supabase-db` đang stopped; chưa deploy dashboard.

## FIX-2026-10-16-23 — Boss bản đồ và thưởng Sổ Tay Ma Thú

- Hiện tượng: boss bản đồ click trực tiếp không bắt đầu trận; 13 boss Kênh 1 không xuất hiện; boss map có thể dựng sai quái/level; toàn bộ boss map và Sổ Tay không có loot riêng.
- Nguyên nhân: NPC boss hiện dùng type 10 nên Flash gửi `clickNpc`, nhưng NPC handler đăng ký sau đã ghi đè combat route. Static row của 13 mythic boss dùng sentinel map 200. Battle cũ tra `cid == npcID` thay vì roster `TBL_NPC_CREATURE.nid`, đồng thời mất schedule NID khi enemy giữ creature CID. `data_tbl_boss_loot` trống hoàn toàn và `daily_only` normal còn bị bootstrap thành schedule state.
- Sửa: route `clickNpc`/`hitNpc` vào shared schedule-boss path; validate map/channel/alive fail-closed; dựng đủ roster catalog và gắn canonical `BossCtx`; inject mythic boss theo registry/channel scope; loại `daily_only` khỏi scheduler. Thêm generator/seed thưởng theo ground/flying/mythic và sáu dải level.
- Dữ liệu local: đã apply 862 loot row vào compose DB; phủ 150/150 registry boss, 3–7 row/boss, 0 item/equipment reference lỗi. Không thay đổi player data.
- Test: focused scheduleboss/combat/scene/NPC/generator PASS; targeted vet và build gameserver/generator PASS. Broad internal không có lỗi boss; còn baseline ngoài phạm vi ở item award quality, Quest registry, chat stat và một flaky center test đã PASS khi chạy riêng.
- Trạng thái: CODE_FIXED/LOCAL_DATA_APPLIED — chưa restart/deploy hoặc Flash smoke-test.

### Follow-up channel 0

- Runtime 07:22:18–07:22:20: `clickNpc(707)` trả `[Connection] Not found`; character 3 đang map 2, connection channel 0 và DB có `(nid=707, channel_id=0, is_alive=true)`.
- Nguyên nhân: validation mới dùng `channelID <= 0`, trong khi gateway/runtime hiện coi channel 0 là hợp lệ.
- Sửa: scope `all` chấp nhận channel 0; channel âm vẫn bị chặn; `channel:1` vẫn chỉ nhận 1. Bổ sung regression cho engagement và scene visibility channel 0.
- Test: focused scheduleboss/combat/scene/NPC handlers, targeted vet và gameserver build PASS. Chưa restart/retest Flash.

## FIX-2026-10-15-22 — Sách triệu hồi, Mật Đạo và vật phẩm xã hội

- Hiện tượng: Sách Triệu Hồi bị tiêu hao nhưng không có Pet đi sau; Sách Dịch Chuyển Mật Đạo bị tiêu hao nhưng không đổi map; Hoa Hồng/Chocolate bị tiêu hao nhưng không tăng thân mật; người dùng nhấp đôi Trứng Gà nhưng không thấy tác dụng.
- Nguyên nhân: type-500 chưa biết exact summon/secret-passage IDs nên chỉ trả message generic. Flash chỉ cho render follow Pet khi có long-buff 1709/1979. `useIntimacyItem` cũ chỉ consume stack. Trứng Gà dùng RPC mục tiêu `delPopNum`, không phải generic `useItem`; callback stack của `addPopNum`/`delPopNum` còn có thể trừ quantity hai lần trên cùng object đã mutate.
- Sửa: 2514/2543 cấp buff 1709 trong 1 giờ/30 ngày và chỉ gọi Pet cam đã khóa; 2798 cấp buff 1979 trong 30 ngày và nhận Pet tím/cam đã khóa. Persist follow, gửi `onCancelPet`→`onCreatePet`; không có Pet hợp lệ thì không consume. 128/2137 preflight không group/battle rồi consume và vào map 140106. 1302/1303 tăng 60, 1315/32 tăng 660 thân mật; Hoa Hồng yêu cầu khác giới, chỉ áp dụng với bạn bè, cập nhật hai row chiều nếu tồn tại, refresh stack và phát lại `onInitViewImC`. Nhấp đôi 724/783 không consume và chỉ dẫn dùng nút mục tiêu; wire `addPopNum`/`delPopNum` vẫn là effect thật.
- Test: exact summon buff/duration/quality, bound-quality eligibility, exact intimacy values/gender flags, bidirectional persistence/non-friend rejection, secret-passage/social target registry, item/social/handler/Pet packages, targeted vet và gameserver build PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test; không migration và không sửa dữ liệu người chơi.

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
- Điều chỉnh theo xác nhận operator 2026-10-15: failure từ 8+ không protection đặt cả sao và `GrowRateAdd` về 0 rồi tính lại stats; dưới 8 giữ cả hai, có protection cũng giữ cả hai. `petStarClear` thủ công không đổi trong điều chỉnh này.
- Test: exact rate table, deterministic success/failure, mốc 8 có/không protection và growth reset/preserve, response `flag/starSafeNum`, regression `1.56 + 0.40 = 1.96`, Pet handler/domain, targeted vet và gameserver build PASS. Broad internal giữ nguyên baseline 4 Pet + 2 chat-stat.
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
