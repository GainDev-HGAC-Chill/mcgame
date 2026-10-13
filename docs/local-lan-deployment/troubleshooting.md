# Khắc phục sự cố

## Thử client 30 FPS / quay lại client gốc

Mở Docker Desktop và stack bằng MCGAME.bat trước, đóng projector khác, chạy `MCGAME-30FPS.bat`. Đây là bản thử riêng: có thể có hoạt ảnh nhanh hơn khoảng 25%, không khẳng định mọi timing giữ nguyên. Nếu đi không tới đích, party follow sai, battle bất thường hoặc server có nhiều RPC hơn, đóng bản thử và dùng MCGAME.bat option 2 để quay lại bản gốc; không reset nhân vật.

Nếu báo thiếu trial asset, build/run `bin/clientfps.exe` theo setup. Tool cố ý từ chối overwrite/checksum không khớp. HTTP không truy cập được khi Docker/server đang tắt không phải bằng chứng file patch lỗi. Live FPS/gameplay/load vẫn phải kiểm tra thực tế.

## `[Connection] method not found` khi mở bảng hoạt động

Log 19:38:27 chỉ ra `getJXHDAward`; một lần mở panel có thể gọi nhiều activity keys. Đã đăng ký RPC đúng tên, trả `[key, []]` theo nhánh client xử lý “không có preview”, không cấp quà giả hay đánh dấu event hoàn chỉnh. Tính năng sự kiện thực sự vẫn có thể thiếu dữ liệu; xem `docs/research/2026-10-11_06_JXHD_PREVIEW_RPC_RESEARCH.md`.

Nếu báo lỗi khi mở tính năng khác, ghi tên nút và thời điểm; cần xác định từng RPC thay vì blanket success cho method chưa triển khai. Không claim mọi tính năng PASS chỉ vì thông báo missing method đã hết.

## `[Connection] invalid arguments` khi qua cổng vào thành

Log 2026-10-11 19:27:19 xác định `sceneChange(757)` bị từ chối vì `from=0`. Client gửi đúng instance ID; cổng thiếu metadata đích phía server. Đã bổ sung fallback an toàn cho portal không có link: tên khớp duy nhất map thường/non-copy → điểm an toàn trong bounds. Gate 757 trỏ tên Xuất Vân Thôn → map 1 (1000,1000). Cổng có link explicit không thay đổi; tên mơ hồ/không tìm thấy hoặc điểm không hợp lệ vẫn bị chặn.

Sau deploy, mở lại Flash và đi qua chính cổng đã lỗi. Không sửa vị trí Gain trực tiếp, không đổi seed/DB để ép chuyển cảnh. Live traversal chỉ được đánh dấu PASS sau xác nhận người vận hành.

## Ngắt kết nối rồi Flash lỗi `Battle.battleOnEnd` lúc logout

Kiểm tra log server trước khi kết luận lỗi client gây disconnect. Incident 2026-10-11 19:07:15 có Go fatal `concurrent map iteration and map write` ở PlayerCache autosave; Docker restart ngay sau đó. Flash #1009 tại logout là lỗi cleanup sau disconnect, không phải bằng chứng nguyên nhân disconnect.

Đã sửa ownership cache/snapshot và concurrency acknowledgement, không sửa SWF. Đóng projector rồi mở lại sau deploy để thử battle và autosave; kiểm tra restart count/log có crash mới hay không. Không reset nhân vật/cấp lại thưởng để chữa crash. Do write-behind, crash có thể làm mất thay đổi chưa flush; cần xác minh tiến độ sau login, không mặc định mọi dữ liệu phiên trước đã lưu.

## Flash `onUpdateAchieve` #2095/#1063 khi nhận thưởng nhiệm vụ

`FinishQuest` từng gọi `onUpdateAchieve` không tham số, nhưng Flash yêu cầu `(type,id,data)`. Callback kế tiếp `onUpdateSafeBattleMid` cũng thiếu collection tham số. Đã bỏ hai placeholder vì không có delta tương ứng; không tạo thành tựu giả hoặc gửi collection rỗng. Đồng thời sửa `onDelCharactorSlot` khi nộp hết item: gửi instance ID và SID thay vì chỉ SID.

Đóng/mở Flash sau deploy và thử nhận thưởng một nhiệm vụ tiếp theo hợp lệ. Không reset/nộp lại nhiệm vụ đã hoàn thành để kiểm thử; lỗi callback xảy ra sau xử lý reward nên không được mặc định rằng lần nhận trước thất bại hoặc cần cấp lại quà. Test regression dùng dữ liệu in-memory, không đụng Gain. Evidence: `docs/research/2026-10-11_03_QUEST_REWARD_CALLBACK_AUDIT_RESEARCH.md`.

## Flash `updateWSPPanel` #2095/#1009 sau chọn nhân vật

Server từng push dữ liệu Chiến Hồn ngay sau `onChooseCharactor`. Callback này tạo panel nhưng không đợi control/template sẵn sàng. Đã bỏ login push, giữ luồng client mở panel → chờ initialized → `getSpWarData`, cùng callback sau nâng cấp. Không dùng delay tùy ý, không sửa SWF hay dữ liệu nhân vật. Xem `docs/research/2026-10-11_02_CHARACTER_SELECTION_CALLBACK_AUDIT_RESEARCH.md` để biết phạm vi audit và giới hạn xác minh.

Sau triển khai, đóng/mở Flash, chọn Gain, kiểm tra vào cảnh rồi mở panel Chiến Hồn. Ghi riêng kết quả client thực tế; Go tests/health check không chứng minh gameplay PASS.

## Flash #2095 / #2088 khi đăng nhập: `onIcl` gửi sai kết nối

Global và Scene dùng hai NetConnections/callback clients khác nhau. `getLineInfo` trước đây tự gửi `onIcl` lên Global; `CallBackGlobal extends Proxy` không có `onIcl`, khiến Flash báo Proxy.getProperty. Đã bỏ side effect này và chặn gửi onIcl lên Global tại outbound boundary, không thay đổi payload hoặc dữ liệu nhân vật.

Matching SWF đã được đọc lại bằng decompiler trong thư mục temporary; không sửa SWF hoặc `.as`. Test routing Global/Scene, test RTMP/util/activity/character, vet, build và Compose redeploy riêng game server PASS. `game check` PASS. Sau deploy phải đóng/mở projector và đăng nhập lại character cũ để xác nhận lỗi trên client đã hết.

## Flash #2095 / #1069: callback `lvUp` không tồn tại

Lỗi xác nhận từ client: `Property lvUp not found on com.qeedoo.game.system.CallBack`. `lvUp` là RPC client gọi server, không phải callback server gọi client. Đã bỏ hai `SendCallback("lvUp", ...)` sai trong shared EXP helper và offline EXP refresh; giữ callback `onUPP` / `onPlayerLevelUp` và RPC registration `lvUp`.

Test hồi quy utils/activity/character và build Go PASS. Triển khai lại riêng `mcgame-server` bằng Compose build/up `--no-deps`, không chạy seed/migration/reset và không restart database. Sau khi deploy xong, mở lại projector, đăng nhập character cũ và kiểm thử lên cấp bằng gameplay bình thường. Chưa đánh dấu Flash lỗi đã hết cho đến khi có kiểm thử thực tế. Không dùng Dismiss All để coi lỗi được sửa.

## Blocker thực tế: WSL2 cần restart Windows

Đã bật `Microsoft-Windows-Subsystem-Linux` và `VirtualMachinePlatform` qua UAC hợp lệ với `-NoRestart`; Windows trả `RestartNeeded: True`. WSL đã cài 2.7.13, nhưng `wsl --status` báo `Wsl/WSL_E_WSL_OPTIONAL_COMPONENT_REQUIRED`. Docker backend log báo `Virtual Machine Platform not enabled`, `No virtualization available`.

Hardware virtualization và SLAT đã bật; không cần sửa BIOS ở thời điểm này. Máy có khoảng 32 GiB RAM. Người vận hành tự restart Windows, mở Docker Desktop, chạy `wsl --status` và `docker info`. Không dùng reset DB hoặc xóa Docker volumes để chữa lỗi engine.

Docker CLI/Compose có phiên bản không có nghĩa engine đã chạy. Trước đây `game status` trả Docker API HTTP 500; `game start` có kiểm tra engine giới hạn 15s để không treo vô hạn. Nếu Docker còn ở màn hình Welcome, chọn Skip nếu không muốn sign in; không cần Docker account để chơi Local.

## Lỗi an toàn deployment đã sửa

- `psql` boolean `t/f` bị so sánh với `true`: đổi SQL sang `::text`, kiểm tra lỗi query và policy chỉ seed database mới. Chưa kiểm chứng trên PostgreSQL thật vì engine chưa chạy.
- Setup trước đây có thể wipe database để thay secrets: nay từ chối database hoặc `.env` có sẵn, kể cả `-y`.
- Bỏ prune builder cache toàn Docker khi game up. Giữ nguyên dữ liệu/volumes.
- Redis/API/dashboard/metrics/pooler trước đây publish mọi interface: giới hạn loopback.
- Profile riêng Local/LAN thay cho sửa config client trong source.

## Migration history không khớp

Dừng deployment, backup trước, kiểm tra history và schema bằng project MCP hoặc CLI trỏ DB_URL tạo từ `docker/.env`. Chỉ reconcile migration history sau khi đối chiếu schema. **Không chạy game reset theo khuyến nghị cũ trong tài liệu.** Không tự include-seed trên database có người chơi.

## Flash trắng / không đăng nhập

Kiểm tra sau khi engine hoạt động: HTTP 200 cho SWF, config.xml, crossdomain.xml; resource URL và logic host đúng profile; container edge/game logs; port 1935; account test. Mở bằng projector Adobe có sẵn, không browser hiện đại. HTTP 200 hoặc port listening chưa chứng minh vào game được.

## Test suite hiện có FAIL

`go test ./...` có 7 assertion failures tại pet, auth PM daily-exp và chat stat handlers. Không sửa công thức gameplay chỉ để deployment/test PASS. Xem tên test trong [test-results.md](test-results.md); đây là vấn đề cần kiểm tra riêng, không phải bằng chứng Docker hỏng.

## LAN

Local phải PASS trước. Kiểm tra Private network, IP/subnet Ethernet, firewall chỉ 1935/8888 trong subnet tin cậy, client dùng URL LAN và profile LAN. Nếu DHCP đổi IP, chạy lại cấu hình LAN và cập nhật đúng hai rule của game. Không mở Redis/Postgres/admin hoặc thay router.
