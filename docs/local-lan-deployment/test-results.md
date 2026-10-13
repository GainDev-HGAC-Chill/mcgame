# Kết quả kiểm thử — 2026-10-11

## 2026-10-12 — dashboard tiếng Việt / quà khởi tạo / cấp đích

- **PASS mã nguồn/test**: cấu hình typed `new_character_defaults`, kiểm tra danh mục/khóa/phẩm chất/sức chứa, giao dịch cùng nhân vật + item/pet + dấu snapshot; không phát khi tải/đăng nhập. Metadata compose70/15/33/3 đối số khớp. Không cần migration, không ghi cấu hình hay phát quà thử.
- **PASS kiểm tra phạm vi**: cấp đích dùng cache hiện hành có khóa, giữ điểm thưởng ngoài cấp, từ chối trang bị không hợp lệ; equip commit kiểm tra cấp cùng khóa, HP/MP không ghi lại snapshot nhân vật cũ. Parent đọc code và rerun các regression20 lần. Re-review agent cuối bị giới hạn, không báo review độc lập hoàn tất.
- **PASS frontend kiểm tra kiểu/parser**: gắn form quà vào Cấu hình trò chơi, form cấp đích vào Tiến trình; Việt hóa menu/form/xác nhận/error fallback và ngày tháng. Parser5 tests PASS. Lượt Việt hóa bị giới hạn đã được parent tiếp tục, không làm lại từ đầu.
- **BLOCKED**: thay đổi cấp ngoại tuyến/chưa có cache trả409; race detector thiếu CGO. **FAIL full suite** vẫn7 lỗi pet/auth/chat đã biết. Vet/build Windows PASS.
- **PASS production/deploy/browser**: game startup19:20:17, dashboard bản nhãn cuối recreate19:28:13; cả hai running/restart_count0. Browser tab mới hiển thị form/quy tắc/nhãn tiếng Việt; tra cứu item1132/trang bị758/pet91 trả200. Bốn cấu hình sai trả400 và cấu hình lưu trữ không đổi. DB/Redis không restart/reset. Tab cũ có thể cần mở lại hoặc Ctrl+F5 sau deploy. Tạo nhân vật thật, chỉnh cấp thật, Flash reconnect vẫn **NOT TESTED**, chỉ chủ game thao tác.
- Chỉ nội bộ/loopback: bảng cấu hình runtime hiện có RLS tắt + anon/authenticated DML; không được công bố Data API/dashboard ra môi trường không tin cậy. Không thay đổi cổng/auth ngầm.
- [Chi tiết](../research/2026-10-12_10_DASHBOARD_VIETNAMESE_STARTER_IMPLEMENTATION.md).

## 2026-10-12 — backend quà tạo nhân vật mới, kiểm thử không ghi player

- **PASS**: `go test ./internal/application/character ./internal/infrastructure/persistence/postgres ./internal/infrastructure/config/store ./internal/domain/character ./internal/application/item`; `go vet` các package này, pet và gameserver; `go build -o bin/gameserver.exe ./cmd/gameserver`.
- **PASS bằng test doubles**: cấu hình thiếu key giữ90/999999/999999; lỗi đọc/JSON/field/null/format/bounds bị chặn; catalog/binding/màu/growth; capacity các túi/pet; chia/gộp stack; snapshot không alias; rollback khi lỗi character/item/pet/marker/commit; retry và không replay sau commit/login/select/load. Không báo các mock là thử giao dịch PostgreSQL thực.
- **PASS kiểm tra compose chỉ đọc**: các function tái dùng security invoker/search_path rỗng; RLS trên character_items/character_pets/character_stat_features. Không cần schema/migration mới; dùng một pgx transaction và marker creation_package.
- **PASS pet contract focused / FAIL pet package baseline**: `go test ./internal/application/pet -run '^TestContract'` PASS. Full package pet vẫn có4 lỗi đã biết: LearnSkillFromBook MaxHP24/146, EquipPetItem finalStrength24/5, UnequipPetItem finalDefence65/0, DeletePetSkill finalHp20/120. Không sửa baseline hoặc báo toàn bộ PASS.
- **NOT TESTED**: tạo nhân vật thật, grant thật, commit/rollback DB với bản ghi player, Flash nhận quà/reconnect, triển khai container và frontend/API do parent phụ trách. Không chạy lại full suite; các baseline pet/auth/chat FAIL trước đó không được đổi thành PASS. Race detector không chạy.

## 2026-10-12 — lỗi panel / kinh tế, bản17:34

- **PASS sửa/test/deploy startup**: color số cho sort1/sort2 nhiệm vụ và mall; bỏ initPRSPanel legacy ghi đè handler thật; PetPVE có s0–s7 không null (panel/login); Ma Tâm cấp box mặc định có template; wallet báo danh tách vàng/kim phiếu; voucher1132 trả100000 bạc/ngân phiếu theo khóa. Key regressions lặp20 lần; affected vet/build PASS. Chỉ game server recreate;17:34:09 tải5530 quests/itemAwards71, RTMP started, running/restart_count0. Không reset/recreate DB/Redis.
- **PARTIAL bí cảnh**: response openflag không còn Boolean; sự kiện vẫn đóng, showPanel null chưa tìm được nguyên nhân đủ chắc chắn.
- **BLOCKED lò luyện**: chặn quy đổi tạm1 tinh thể/record trước consume; trả false đúng Responder để nhả waiting flag, cả lỗi dispatch/rate limit. Đây không phải hoàn tất hệ thống lò luyện, không đổi vật phẩm/điểm thật.
- **NOT IMPLEMENTED / NOT FIXED**: paid Explorer Medal, phần RPC khóa đặc huấn còn thiếu, túi chế tạo null chưa xác định được dòng/template/cache gây lỗi, xóa nhân vật có deadline/restore. Không dùng fake success hoặc xóa cứng.
- Dashboard: user chọn tăng/hạ cấp + quà item/pet; **Draft**, chưa thêm UI/action mới. [Kế hoạch](../plans/2026-10-12_02_PLAYER_ADMIN_STARTER_DEFAULTS.md) cần duyệt chính sách hạ cấp và giao quà đúng một lần.
- **FAIL full suite**: bốn pet-stat, một auth PM EXP, hai chat-stat baseline; không báo PASS toàn bộ. Review/craft subagents bị giới hạn usage; parent kiểm tra source và rerun test nhưng không tuyên bố review độc lập hoàn tất. Race **BLOCKED**, Flash hậu deploy **NOT TESTED**. [Chi tiết](../research/2026-10-12_09_PANEL_PROTOCOL_ECONOMY_TRIAGE_RESEARCH.md).

## 2026-10-12 — VIP, trả nhiệm vụ bị đơ và xóa nhân vật

- **PASS sửa/test VIP**: initPmData/buyPm trả kết quả Responder, không gọi callback toàn cục không tồn tại; giữ push VIP/tiền hợp lệ. Không mua VIP trên player thật.
- **PASS sửa/test freeze nhiệm vụ**: tái hiện bằng service hai chiều quest→item→quest; thưởng vật phẩm không còn lấy lại cùng mutex; tiến độ quest khác được cập nhật sau nhả khóa và push sau slot thưởng. Cấm replay thưởng cho quest đã bị gián đoạn.
- **PASS responder nhiệm vụ**: rate-limit1100ms/dispatch error có kết quả flag=false/info, không silent-drop; lỗi NPC/điều kiện dùng info đúng Flash. OnLevelUp không ghi snapshot cũ hoặc tin level từ notification.
- **PASS audit lại snapshot5530 ID**:3838 có evaluator,1692 thiếu logic/tiền tệ/target bị chặn. Không phải xác nhận mọi nhiệm vụ chơi được hoặc export DB mới.
- **BLOCKED xóa nhân vật**: chưa có lưu deadline/khôi phục; không nối startDeleteChar sang hard-delete và không xóa player thật. [Chi tiết](../research/2026-10-12_07_CHARACTER_DELETION_RESEARCH.md).
- **FAIL full suite**: bốn pet-stat, một auth PM EXP, hai chat-stat đã ghi nhận trước đó; các package quest/item/activity/RTMP PASS. Race vẫn BLOCKED do CGO/compiler.
- **PASS sách mở rộng (test đơn vị)**:324 kho +1 tab/max5;325 túi đồ +1 tab/max7;326 túi pet +1 ô/max180; kiểm tra giới hạn trước dùng, đồng bộ alias và onUPP/stack/SID. Không dùng sách trên player thật; không hoàn lại sách cũ tự động. Lưu item và character không phải giao dịch nguyên tử; lỗi sau consume vẫn cần chẩn đoán riêng.
- **PASS build image / BLOCKED startup**: stack cũ (DB/Redis/edge/khác) đã dừng lúc12:42 trước redeploy; game mới lỗi DNS db/restart. Đã dừng riêng container game mới để tránh restart-loop; không reset/recreate/bật lại DB/Redis hoặc volume. Chờ xác nhận bật stack hiện có; gameplay Flash **NOT TESTED**. [Nghiên cứu](../research/2026-10-12_08_GAMEPLAY_FAILURES_FOLLOWUP_RESEARCH.md).

## 2026-10-12 — rà soát toàn bộ5530 nhiệm vụ

- **PASS audit catalog**: tất cả5530 ID được phân loại, không chỉ chuỗi bạn bè. Có3846 nhiệm vụ có dòng yêu cầu chuẩn và1684 không có. Báo cáo từngID: `.local/research/quest-completion-policy-20261012.json`.
- **Bộ kiểm tra đã có (không phải gameplay PASS)**:3682 item/quái/pet có target;66 hội thoại/tri thức NPC;14 cấp nhân vật;14 cấp pet;23 số pet;39 bạn bè =3838.
- **BLOCKED/NOT IMPLEMENTED**:1493 chưa có điều kiện xác minh;195 rt/rn tiền tệ chưa rõ cơ chế;4 target không tồn tại =1692. Không giả thành công; UI state103 và thông báo thiếu cấu hình. Không nói toàn bộ nhiệm vụ đã chơi được.
- **PASS tests/vet/build**: nhận nhiệm vụ không tin mục tiêu/tiến độ client; treo thưởng dùng catalog; active objectives đối chiếu chuẩn; bảo toàn tiến độ kill khớp; special bỏ mục tiêu giả; đọc UI không ghi đè trạng thái; kiểm tra NPC thật/map/khoảng cách300; hết hạn/JSON lỗi/dependency/guild/target; kill credit đúng người, chống trùng và retry sau lỗi lưu một phần. Bộ test trọng điểm lặp20 lần PASS.
- **FAIL full suite (ngoài phạm vi)**: pet stats, auth PM EXP14900 vs15400, chat stats vẫn còn lỗi. Quest/combat/NPC/cache/PostgreSQL PASS. Race detector **BLOCKED** do CGO/compiler.
- **PASS deploy/startup**: chỉ game server;12:34:08 tải5530 quests/itemAwards71, RTMP started, running/restart_count0. PostgreSQL/Redis không recreate. Không viết SQL player, reset tiến trình, thu hồi hay replay thưởng; không sửa SWF/AS.
- **NOT TESTED**: gameplay Flash sau deploy, tất cả route NPC/custom-system và các nhiệm vụ bị chặn. [Nghiên cứu và giới hạn](../research/2026-10-12_06_ALL_QUEST_REQUIREMENTS_AUDIT_RESEARCH.md).

## 2026-10-12 — điều kiện nhiệm vụ kết giao bạn bè

- **FAIL xác nhận trước sửa**: GainTest không có bạn,2695/2696 đã hoàn thành với objectives=[];2697 active cũng báo state104.
- **PASS tests/vet/build/deploy**: chuỗi2695–2733 kiểm tra đủ số bạn thật và tất cả thân mật>=50 theo mô tả. Cùng bộ kiểm tra chặn state104 và nhận thưởng sai. Type10 rỗng chưa hỗ trợ bị chặn; nhiệm vụ nói chuyện type1 giữ nguyên. Tests bao gồm bạn rỗng/thiếu/trùng/self/blacklist/owner khác,49/50 thân mật, xóa bạn, lỗi repository, JSON runtime flag và bảo toàn lịch sử.
- Chỉ rebuild/recreate game server; startup08:52:08, RTMP started, running/restart_count0. Không sửa DB/player/reward seeds; không reset hay thu hồi thưởng đã nhận.
- **NOT TESTED**: Flash sau đăng nhập lại; nhận thưởng thật với đủ bạn. Script gốc điều kiện xã hội chưa phục hồi, quy tắc hiện dựa trên mô tả catalog. Không chứng nhận toàn bộ nhiệm vụ đã triển khai.

## Đăng ký web đã triển khai — 2026-10-12

- **PASS (independent tests/build)**: application auth/public registration/config/Postgres account repository/cmd gameserver/character/domain auth tests; focused newChar/onIcl/getLineInfo tests; go vet ./... và Windows server build. ID1/password1→bcrypt(MD5hex)→Flash-style login PASS trong test doubles, không tạo account thực. Frontend frozen pnpm install/build và Go/edge Docker images PASS; independent read-only source review không thấy lỗi nghiêm trọng.
- **PASS (deploy/infra)**: nginx -t trước deploy; chỉ recreate game server/edge với --no-deps. Startup08:34:35 xác nhận registration listener0.0.0.0:8081 và RTMP started. game check PASS; không reset/migration/seed/prune hoặc restart DB/Redis/GM/dashboard. Game server chỉ host-publish127.0.0.1:2112;8081 internal-only; admin/DB/cache/Studio giữloopback.
- **PASS (host/browser and safe HTTP negatives)**: register.html/JS/CSS200, browser DOM đầy đủ3ô/nút; API{}400, spoof Origin/forwarded host400, body>4KiB400, GET405, trailing slash404, encoded path400, registration limiter429 ở burst yêu cầu sai. Admin sessions/action và /rest/v1/accounts qua edge404. Client30 URL vẫn200. Không gửi successful registration/không ghi account/player để thử.
- **APPROVED POLICY**: ID1/password1 hợp lệ; không min-length/strength/ASCII restriction, preserve case/whitespace. Chỉ giữ nonempty và trầnFlash50/20UTF16; duplicateusername409. Quà chỉ cấp khi tạo nhân vật mới.
- **NOT TESTED**: đăng ký account thật→Flash login→tạo nhân vật→logout/relogin/restart/persistence; Unicode trong actualSWF và máy thứ hai Radmin. Browser tab đã đóng trước thử tương tác; DOM đọc thành công không được coi là interactive-submit PASS. Host CGO disabled nên chưa chạy race detector. Known pet/chat/auth PM EXP baseline failures không sửa.
- **SECURITY**: chỉ dùng trusted Local/Radmin; HTTP chưa có TLS, không reuse password quan trọng/không coi Internet-ready. Existing CLI advisory về4config tables RLS disabled được ghi nghiên cứu, không auto-remediate hoặc mở Data API. [Plan](../plans/2026-10-12_01_WEB_ACCOUNT_REGISTRATION.md) · [Research](../research/2026-10-12_05_WEB_ACCOUNT_REGISTRATION_RESEARCH.md).

## Tự động quà nhân vật mới — 2026-10-12

- **PASS (implementation/tests)**: mỗi Service.Create bắt đầu cấp90/gold999999/goldBind999999, áp sau class stats và trước repository.Create. RaiseLevelTo cộng445 điểm thuộc tính/max và+89 base stats; HP/MP đầy. Tests six classes/two genders, failed create, existing/load/select không regrant và NewChar→onIcl/DTO level90/cumulative exp PASS.
- **PASS (build/deploy/infra)**: character application/domain tests, focused auth newChar/list tests, go vet ./..., Windows build, Docker build/recreate chỉ game server; RTMP startup08:11:00 và game check PASS. nginx reload để resolve upstream sau recreate. Không assistant reset/migration/seed/prune/restart DB/Redis hoặc sửa trực tiếp tài khoản/nhân vật có sẵn.
- **KNOWN FAILURE**: full auth suite TestChooseCharacter_GrantsDailyPMLoginExpOncePerDay14900 vs15400, đã tồn tại; không sửa logic PM EXP trong task.
- **CONFIRMED (source)**: nút Đăng ký Flash có thật nhưng chỉ mở URL register.html; trang/API chưa wired. Không triển khai đăng ký hoặc tạo account trong task này.
- **NOT TESTED**: tạo nhân vật thực trên Flash, login/relogin/restart kiểm tra level/currency persisted; không coi tests/health là live gameplay PASS. [Chi tiết](../research/2026-10-12_04_NEW_CHARACTER_STARTER_PACKAGE_RESEARCH.md).

## Radmin VPN — 2026-10-12

- **PASS (code)**: `go test ./cmd/game ./internal/infrastructure/config/... ./internal/infrastructure/rtmp/...`, focused vet và Windows game/gameserver builds. Tests kiểm tra adapter Radmin up +26/8, explicit/auto chọn IP, từ chối IP không assigned/ambiguous, render resource/logic và rewrite URL kênh localhost/loopback nhưng không external URLs. LAN vẫn từ chối26/8.
- **PASS (profile/deploy)**: IP thực `26.31.111.123`, adapter `Radmin VPN`. Chạy game radmin, Compose recreate riêng edge/game server (`--no-deps`), nginx reload sau game recreate. Không migration/reset/seed/prune, không restart PostgreSQL/Redis hoặc sửa DB/player trực tiếp. Startup07:52:34 báo RTMP started và3 channels; env advertised host đã xác nhận26.31.111.123.
- **PASS (host-side only)**: client URL HTTP200, served config resource/logic dùng Radmin IP; TCP1935 thành công qua interface Radmin; game check PASS. Compose game ports chỉ bind26.31.111.123; DB/cache/admin/Studio/metrics giữ127.0.0.1.
- **BLOCKED (firewall)**: session không có Administrator; chưa có rule MCGAME-Radmin. Cần IP Radmin từng peer và rule inbound giới hạn adapter/local IP/peer IP/TCP8888+1935 theo [hướng dẫn](radmin.md). Không mở rộng firewall hoặc tự nâng quyền.
- **NOT TESTED**: máy thứ hai tải client/login/chọn kênh/chơi cùng và persistence. Host-side HTTP/TCP và infra PASS không chứng minh Radmin remote gameplay PASS.

## Retest 2026-10-12 — luồng tự mặc và callback shop

- **FAIL (người dùng)**: mặc đồ còn trống icon; tháo đồ hiển thị. Log xác nhận luồng thực là useItem, chưa được bản equipOn trước đó sửa.
- **PASS (tests)**: service reload snapshot sau auto-equip; useItem handler đồng bộ SID/instance cuối với repository detached. Sửa QuestNotifyHelper dùng envelope có d cho onAddChaQuest sau mua shop. Item/quest/exchange suites, vet/build PASS; shop compile (chưa có tests).
- **NOT IMPLEMENTED**: đổi phiếu và lắp túi tiện dụng. Đã đăng ký exchangeAwardToGoldBind/setTempBag chỉ để báo unavailable rõ, không giả thành công hay trừ tiền/dùng túi. Cần giao dịch tài khoản→nhân vật an toàn và trạng thái/expiry túi thật.
- **PASS (deploy/startup)**: riêng game server, startup07:25:39 nạp itemAwards67 và RTMP started, running/restart_count0. Flash auto-equip/purchase callback vẫn **NOT TESTED**; không coi startup là gameplay PASS. Không reset DB hoặc replay các lần mua đã thành công.

## Cập nhật 2026-10-12 — mặc đồ / GM / túi shop

- **PASS (người vận hành xác nhận)**: sắp xếp/kéo đồ hết lỗi; các panel trước đó ổn. Không coi xác nhận này là đã triển khai đầy đủ sự kiện chưa cấu hình.
- **PASS (tests)**: item application/handler suites, vet/build; mặc/thay đồ, hai slot nhẫn và tháo đồ gửi đúng DTO/ID/SID trong tests. Pet fallback giữ tăng trưởng1.5;13 fixtures túi shop nhận đúng nội dung/grade/pet.
- **PASS (data)**:13 dòng thưởng shop theo mô tả catalog, installer chạy2 lần; total67, starter54 giữ nguyên. Backup+export seed; không sửa player/pet hoặc tự mở túi.
- **FOUND**: GM http://127.0.0.1:3000/dashboard/players/1 → Wallet/Resources/Items. Không tạo trang mới, không cấp tài nguyên để thử. Hiện thiếu xác thực ứng dụng, chỉ dùng localhost.
- **NOT TESTED**: Flash mặc/tháo đồ ngay không relog, mua/mở túi pet mới, GM grant/persistence. Shop325 ứng viên được audit riêng; không tuyên bố mọi túi đã có thưởng khi thiếu nguồn/tỉ lệ thật.
- **PASS (deploy/infra)**: redeploy riêng game server, startup00:23:12 nạp itemAwards67, RTMP started, running/restart_count0, game check PASS và GM route HTTP200. Không reset/redeploy PostgreSQL/Redis; không coi health là gameplay PASS.

## Sắp xếp túi / panel sự kiện

- **PASS**: tests item application/handler + activity, vet và host build. Callback sort phục hồi toàn bộ túi: xóa vị trí cũ trước, thêm DTO cuối, repaint onItemSort([]); không tự sửa inventory Gain. Bỏ thưởng mẫu gây dereference undefined; thêm serchForGift/getConsume/getActAwardInfo đúng tên.
- **PASS**: Compose build/recreate riêng mcgame-server báo Started. Không reset/redeploy PostgreSQL/Redis; không ghi schema/player data.
- **BLOCKED**: kiểm tra runtime sau harness restart không kết nối được Docker Desktop Linux engine named pipe; không coi output Started trước đó là health hiện tại. Không chạy lại deploy.
- **NOT TESTED**: Flash sort/drag và panel nạp/shop/gift sau sửa. Cần đăng nhập lại trước kiểm tra túi. Sự kiện server mới vẫn **chưa cấu hình** lịch/thưởng; RPC báo rõ thay vì dựng lịch hay nhận quà giả. [Chi tiết](../research/2026-10-11_10_BAG_SORT_EVENT_PANELS_RESEARCH.md).

## Cập nhật sau người vận hành restart Windows

### Sau chuyển dữ liệu Docker/Go sang I

- **PASS**: Docker data VHD copy SHA256 khớp, junction giữ đường dẫn cũ; không reset/prune/delete volumes. C trống ~30,3 GiB sau chuyển, từ ~0,8 GiB.
- **PASS**: stack 10 container chạy lại, infra check PostgreSQL/Redis/nginx; HTTP HEAD client gốc/loader30/main30 đều200. Kết quả này thay blocker HTTP của trial dưới đây.
- **NOT TESTED**: Flash gameplay/FPS/progression sau chuyển. Không dùng health để khẳng định gameplay ổn định.
- Không gỡ ứng dụng; bộ cài nhỏ Program Files giữ lại theo quyết định người vận hành. Dữ liệu thật nằm trong `.local/docker`, không xóa dù gitignored. Chi tiết: [storage](storage.md).

### Lỗi dùng Túi Quà Tân Thủ cấp1

- **FAIL/BLOCKED**: RPC `useItem(1,-1,2)` lúc22:07:10 trả `Internal error: cannot use item`; template2918/type550 chưa có reward config. Xảy ra trước chuyển ổ đĩa.
- **PASS** kiểm tra chỉ đọc: award table tổng0 rows; instance2 của Gain vẫn stack_count1, không bị trừ. Chưa sửa inventory/quest rewards/database.
- Cần nguồn phần thưởng gốc hoặc định nghĩa quà local được người vận hành duyệt để mở túi thật; không giả thành công.

### Cấu hình thưởng tân thủ đã được duyệt

- Người vận hành hủy rollback trước khi có file nào bị revert; duyệt danh sách thưởng cấp1–50, trang sức tím Trác Việt và pet91/tăng trưởng1.9.
- **PASS**: thêm54 dòng guaranteed, số dòng9/9/9/11/8/8; tổng số lượng53/87/22/24/47/64. Installer chạy2 lần không tạo trùng. Seed xuất bằng pg_dump; backup bảng trước đó giữ ở I.
- **PASS**: tests mở đủ6 gói, thiếu cấp, túi đầy và dùng lại túi đã tiêu thụ; quality trang sức20/prefix5, pet1.9 và Kim Cương cấp4/colorCode3. Item application/handler suites, vet và host build PASS; independent static review không thấy lỗi nghiêm trọng.
- **PASS** kiểm tra chỉ đọc: túi instance2 của Gain vẫn1, timestamps không đổi; không phát quà hay sửa player data.
- **PASS**: rebuild/recreate riêng mcgame-server; startup22:51:18 xác nhận `itemAwards=54`, RTMP started, running/restart_count0 và `game check` PASS. PostgreSQL/Redis không redeploy/reset. Lần đầu timeout do `.local` vào build context; đã exclude và retry thành công. **NOT TESTED**: real Flash mở túi/callbacks/progression sau cấu hình; không coi test mô phỏng hay restart_count0 vừa recreate là gameplay/ổn định dài hạn PASS.
- Quà là các item/box đúng danh sách; nội dung bên trong các túi trang bị2049/2050 và toàn bộ tính năng dùng từng consumable không thuộc lần cấu hình này. Chi tiết [research](../research/2026-10-11_09_STARTER_GIFT_AWARDS_RESEARCH.md).

### Bản thử client 30 FPS, opt-in

- **PASS**: Go generator tạo hai SWF copies riêng; original SHA256 unchanged. Main decoded chỉ đổi 9 bytes; unique class/method patch matching, output refuses overwrite, default launcher URL giữ nguyên.
- **PASS**: cmd/clientfps + cmd/game tests/vet/build; independent FFDec xác nhận FPS30, DEFAULT28, RPC count thresholds30 và movement/follow comparison thresholds30.
- Launcher: `MCGAME-30FPS.bat` / `bin/game.exe client30`; quay lại bản gốc bằng MCGAME.bat option2. Không đổi server tick/config/player data.
- **BLOCKED**: HTTP fetch hai asset và Docker inspect do Docker Desktop Linux engine đang không chạy; chưa xác nhận edge phục vụ trial files.
- **NOT TESTED**: real FPS, Flash login/movement/party/portal/battle/autosave/server load. Hoạt ảnh không scale có thể nhanh hơn ~25%; chưa chứng nhận timing tương đương24FPS.
- Evidence: `docs/research/2026-10-11_08_FPS30_TRIAL_RESEARCH.md`.

### 60 FPS feasibility audit

- **PASS**: matching served SWF P-code and animation/movement exports read-only; identified frame-driven animation, double-scaled movement and RPC count-bypass hazards.
- **BLOCKED**: a constant-only FPS=60 change does not preserve game timing; no timing-safe copy has been implemented/validated. No FPS patch activated, no original asset/default launcher modified.
- **NOT TESTED**: 60 FPS client gameplay and server capacity under that client. Current infrastructure health does not certify either.
- Evidence: `docs/research/2026-10-11_07_FPS60_TIMING_AUDIT_RESEARCH.md`.

### Missing `getJXHDAward` khi mở hoạt động

- **FAIL (bản trước)**: runtime 19:38:27 báo missing getJXHDAward nhiều lần; đây là RPC preview hoạt động, không phải claim. Trong 20-minute log window kiểm tra không có unique missing method khác.
- **PASS**: exact RPC registration, tests 8 supported key envelopes `[key, []]`, invalid/unauthorized requests và dispatcher integration; full activity/RTMP tests, vet, host/image build, Compose redeploy riêng game server, `game check`.
- **NOT TESTED**: Flash mở lại panel sau deploy. No-preview response được client xử lý nhưng không chứng minh các event/reward systems đã triển khai; reward metadata vẫn chưa wired ở handler này.
- Không sửa player/DB/SWF, không cấp quà giả hoặc blanket success unknown methods. Evidence: `docs/research/2026-10-11_06_JXHD_PREVIEW_RPC_RESEARCH.md`.

### Cổng vào Xuất Vân Thôn: `sceneChange(757)`

- **FAIL (bản trước)**: 19:27:19, portal 757 thiếu `from`; server báo invalid arguments dù client gửi đúng instance ID.
- **PASS**: fallback portal thiếu link dùng tên map đích khớp duy nhất và safe point hợp lệ; 757 → map 1 (1000,1000). Unknown/ambiguous/nonportal/copy/special/bad-coordinate tests bị chặn; explicit-link behavior giữ nguyên. Full scene service/handler tests, affected vet, host/image build PASS.
- **PASS**: Compose build/recreate riêng game server và `game check`. Không sửa seed/DB/player position trực tiếp hoặc SWF.
- **NOT TESTED**: Flash đi qua cổng sau redeploy. Exact historical portal arrival point chưa được recovered; (1000,1000) là safe-point policy.
- Bản cache trước: log runtime có nhiều persistence ticks 19:24:56–19:29:56, `saved_count=1`, restart_count=0 trước lần redeploy này. Đây là checkpoint autosave/battle server-side, không phải chứng minh mọi UI/gameplay PASS.
- Evidence: `docs/research/2026-10-11_05_UNLINKED_PORTAL_RESEARCH.md`.

### Crash autosave và disconnect (2026-10-11 19:07 → deploy 19:21)

- **FAIL (bản trước)**: Go fatal `concurrent map iteration and map write` lúc 19:07:15 ở Redis PlayerCache autosave; Docker restart lúc 19:07:16, restart_count=1. Flash báo #1009 khi logout cleanup sau disconnect.
- **PASS**: sửa entity ownership/deep copies và detached full/dirty snapshots; bảo toàn dirty updates tới trong lúc flush, acknowledge tombstone đã lưu; serialize DB/Redis writes per-player; atomic position/stats patches. Không sửa client SWF hoặc reset dữ liệu.
- **PASS**: final Linux `go test -race` cache/domain pet/valuecopy; full focused cache/pet/valuecopy/combat/quest/NPC/RTMP tests, `go vet ./...`, host build, Docker image build. Independent static recheck không tìm serious repair-induced defect còn lại.
- **PASS**: Compose rebuild/recreate riêng game server lúc 19:21:44; `game check` và trạng thái running/restart_count=0 ngay sau deploy. PostgreSQL/Redis không restart/reset. Không dùng restart_count=0 ngay sau recreate làm bằng chứng ổn định dài hạn.
- **NOT TESTED**: Flash reconnect, battle + pet qua nhiều autosave 60s và logout cleanup sau deploy; tiến độ Gain còn đầy đủ sau crash. Write-behind có thể mất thay đổi chưa flush; không tự replay rewards.
- **BLOCKED**: golangci-lint không có trên host; Windows cũng không có gcc/clang, race tests đã chạy bằng ephemeral Linux container/read-only source mount thay thế.
- Evidence: `docs/research/2026-10-11_04_AUTOSAVE_CACHE_CRASH_RESEARCH.md`.

### Callback nhận thưởng nhiệm vụ (2026-10-11)

- Người vận hành báo **FAIL** trên bản trước: `onUpdateAchieve` #2095/#1063, expected 3 got 0 khi nhận thưởng.
- **PASS**: audit toàn bộ callback FinishQuest và EXP/pet helper với CallBack recovered từ SWF phục vụ. Bỏ placeholder zero-arg `onUpdateAchieve` và `onUpdateSafeBattleMid`; sửa item deletion thành `(instance ID,SID)`.
- **PASS**: regression TakeQuest → FinishQuest, nộp hết/nộp một phần, persisted completion, không callback ở claim lặp; full quest handler/application và NPC/utils/RTMP tests, vet, host/image build.
- **PASS**: Compose rebuild/recreate riêng `mcgame-server --no-deps` và `game check`. DB/Redis không restart/reset; không sửa/cấp lại dữ liệu Gain.
- **NOT TESTED**: nhận thưởng nhiệm vụ tiếp theo trên Flash sau deploy. Không đánh dấu gameplay PASS chỉ dựa vào tests/health checks; không reset nhiệm vụ đã hoàn thành để kiểm thử.
- Evidence: `docs/research/2026-10-11_03_QUEST_REWARD_CALLBACK_AUDIT_RESEARCH.md`.

### Sửa vòng đời `updateWSPPanel` sau chọn nhân vật (2026-10-11)

- **PASS**: đối chiếu callback/ViewManager/WarSpritePanel với SWF đang phục vụ; bỏ push panel chưa initialized khi login, giữ `getSpWarData` và push sau upgrade.
- **PASS**: regression chuỗi chọn nhân vật/relogin, scalar currencies, không sửa provider state; panel RPC trả dữ liệu existing/default/error/no-character và không ghi DB; focused RTMP/utils/activity/character/WarSprite tests, vet, host/image build.
- **PASS**: Compose redeploy `mcgame-server --no-deps` và `game check`; PostgreSQL/Redis không restart/reset, không sửa dữ liệu Gain hoặc SWF.
- **FAIL (đã tồn tại)**: full auth suite `TestChooseCharacter_GrantsDailyPMLoginExpOncePerDay`, PM EXP 14900 so với 15400. Không thay reward logic trong bản sửa này.
- **NOT TESTED**: chọn Gain/vào cảnh/đăng nhập lại và mở Chiến Hồn bằng Flash sau deploy. Chưa khẳng định gameplay PASS hoặc mọi lỗi callback đều hết.
- Audit rộng hơn: `docs/research/2026-10-11_02_CHARACTER_SELECTION_CALLBACK_AUDIT_RESEARCH.md`.

### Sửa lỗi đăng nhập lại `onIcl`

Client báo #2095/#2088 ngay khi login lại. Đã xác định Global `getLineInfo` gửi sai scene callback `onIcl` tới `CallBackGlobal extends Proxy`; removed side effect và thêm guard Global. Test regression routing PASS cho `tcn`, `master/test`, Scene/Line; payload scene giữ nguyên. RTMP/utils/activity/character tests, vet và host/image build PASS. Compose build/recreate riêng game server và `game check` PASS; DB/Redis không restart. Đăng nhập Flash lại sau bản sửa này vẫn **NOT TESTED**, cần người vận hành xác nhận.

Blocker WSL2 ban đầu đã được gỡ: Docker engine trả 29.8.2. Compose đã chạy game/edge/dashboard và Supabase; DB/Redis/auth/rest/Studio/API healthy. Log từ phiên Flash của người vận hành xác nhận `admin1` login, tạo character `Gain`, sceneLogin, nhận/hoàn thành quest 1 và nhận quest 6. Đây là bằng chứng tương tác client thật, không chỉ unit tests; persistence sau logout/restart và LAN vẫn chưa kiểm thử.

Client báo callback `lvUp` không tồn tại (#2095/#1069). Đã sửa hai outbound callback sai, regression tests utils/activity/character, vet và build PASS. Compose image build/recreate riêng game server thành công lúc 17:54; DB/Redis không restart. `game check` PASS sau deploy (DB readiness, Redis PING, nginx config). Live level-up sau fix chưa xác minh. Không tự sửa level/EXP hoặc password để kiểm thử.

Các mục BLOCKED trong bảng dưới là **snapshot triển khai ban đầu trước restart**, không phải trạng thái Docker hiện tại.

## Snapshot ban đầu trước restart

**LOCAL: BLOCKED. LAN: BLOCKED, chưa đạt LAN READY. Không có bằng chứng chơi game thành công.**

| Hạng mục | Kết quả | Bằng chứng / giới hạn |
|---|---|---|
| Dependency installation | PASS | Go 1.27.0, Docker Desktop 4.94.0, Docker 29.8.2, Compose v5.5.1, WSL 2.7.13, Supabase CLI 2.120.0 |
| Environment runtime | BLOCKED | Windows features RestartNeeded=True; WSL optional component required; Docker Linux engine unavailable |
| Build Go tools/server | PASS | `bin/game.exe`, `bin/gameserver.exe`, static Linux `bin/gameserver-linux` build thành công |
| Management unit tests | PASS | `go test ./cmd/game`; setup bảo toàn secrets/data, reject unsafe IP, profile rendering, seed policy |
| Management vet | PASS | `go vet ./cmd/game` |
| Full Go tests | FAIL | 7 failures trong 3 packages, liệt kê bên dưới |
| Compose configuration | PASS | `docker compose ... config --quiet` exit 0; không đồng nghĩa containers healthy |
| Docker stack health | BLOCKED | Desktop chạy nhưng engine không có VM; status HTTP 500 |
| Migration / seeds | BLOCKED | Chưa có PostgreSQL project đang chạy; chưa áp migration/seed |
| Account test / character persistence | BLOCKED | Chưa đăng nhập/chưa tạo nhân vật/chưa restart DB thử dữ liệu |
| Flash Projector executable | PASS | Authenticode Adobe Valid; `game client` mở process/window Adobe Flash Player 32 |
| HTTP SWF loaded | BLOCKED | Không có asset container/server; không xác nhận SWF tải thành công |
| Local login screen | BLOCKED | Chưa xác minh hiển thị màn hình đăng nhập |
| Local account login | BLOCKED | Chưa login thật |
| Server/line selection | BLOCKED | Chưa kiểm thử |
| Character create/select | BLOCKED | Chưa kiểm thử |
| Enter game / movement | BLOCKED | Chưa kiểm thử |
| Gameplay action | BLOCKED | Chưa kiểm thử |
| Logout/relogin persistence | BLOCKED | Chưa kiểm thử |
| LAN IP/subnet discovery | PASS | Ethernet 192.168.2.9/24, Private; Radmin VPN không chọn |
| LAN profile/network readiness | NOT TESTED | Local profile được sinh; LAN command có sẵn nhưng chưa áp sau Local PASS |
| Firewall game rules | NOT TESTED | Chưa tạo rule; không tắt firewall/Defender |
| Second LAN machine connection | NOT TESTED | Không có máy khách thứ hai để kiểm thử |
| Two players simultaneous | NOT TESTED | Chưa kiểm thử hai projector/account hoặc LAN clients |
| Start/Stop/Restart runtime | BLOCKED | Start thực tế dừng exit 1 sau timeout engine 15s; Stop/Restart chưa kiểm thử runtime |
| Dashboard/Redis/nginx runtime | BLOCKED | Chưa có stack running |

## Full-suite failures (không thay gameplay trong phạm vi deployment)

- `internal/application/pet`: `TestLearnSkillFromBookUsesItemInstanceAndAppliesBuff`, `TestEquipPetItemMovesItemToPetSlotAndAppliesBonuses`, `TestUnequipPetItemMovesBackToBagAndClearsEquipmentBonus`, `TestDeletePetSkillConsumesItemAndClearsSlot`.
- `internal/presentation/rtmp/handlers/auth`: `TestChooseCharacter_GrantsDailyPMLoginExpOncePerDay`.
- `internal/presentation/rtmp/handlers/chat`: `TestSay_StatExp_AppliesExperienceAndPersists`, `TestSay_StatLevel_RaisesTargetLevelAndPersists`.

## Đã thay đổi

- `cmd/game/{main,setup,stack}.go`: command dispatch, setup guards, migration safety, stop grace, bỏ global builder prune.
- `cmd/game/operations.go`, `operations_test.go`: quản lý Local/LAN, start/stop/restart/status/logs/check/client, regression tests.
- `docker/docker-compose.yml`: loopback admin/data/cache ports, IP-bound game ports, client profile mount, shutdown grace.
- `.gitignore`: bỏ qua profile XML theo máy.
- `README.md`, `docker/README.md`: bổ sung workflow Windows an toàn và phiên bản thực tế.
- `docs/local-lan-deployment/*.md`, `docs/memory/local-lan-deployment.md`, `docs/PROJECT_KNOWLEDGE_BASE.md`: hướng dẫn và kết quả.
- Generated git-ignored files: `docker/.env`, `docker/client/local.xml`, binaries trong `bin/`.

## Kiểm thử tiếp theo bắt buộc

Người vận hành tự restart Windows → Docker engine → `game local start` → health/migration/templates/account → Flash login → chọn line → tạo/chọn character → vào map → di chuyển → gameplay action → logout/relogin → restart/persistence → chuyển LAN/firewall hẹp → hai account/máy khách đồng thời. Chỉ cập nhật PASS sau thao tác thật; không lấy unit tests/port/health thay thế.
