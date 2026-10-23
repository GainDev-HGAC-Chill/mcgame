# Triển khai Windows Local/LAN

Chơi từ máy khác qua VPN: xem [hướng dẫn Radmin](radmin.md). Profile `radmin` riêng hỗ trợ IP `26.x.x.x`; profile `lan` vẫn chỉ nhận private IPv4.

## Trạng thái hiện tại — 2026-10-11

Local đã có phiên chơi thực tế sau khi restart Windows và sửa lỗi server. Sau chuyển Docker data sang ổ I, stack/infra checks và HTTP client gốc/30 FPS **PASS**; live gameplay 30 FPS vẫn **NOT TESTED**. Xem [kết quả thực tế](test-results.md) và [vị trí dữ liệu mới](storage.md). Không xóa `.local/docker` hoặc chạy `git clean -fdx`.

Đã cài từ nguồn chính thức: Go 1.27.0, Docker Desktop 4.94.0 (Docker 29.8.2, Compose v5.5.1), WSL 2.7.13, Supabase CLI 2.120.0 (`bin/supabase.exe`, kiểm tra SHA256). Source thực tế yêu cầu Go >=1.26.0. Không cần pnpm trên host vì dashboard build trong Docker. Flash Projector có sẵn, chữ ký Adobe hợp lệ; không cần cài Flash browser plugin.

## Sau khi người vận hành tự restart Windows

1. Mở Docker Desktop, bỏ qua đăng nhập Docker nếu không cần, chờ Linux engine hoạt động.
2. Mở PowerShell mới tại workspace và chạy:

```powershell
.\bin\game.exe local start
.\bin\game.exe client
```

`docker/.env` và profile Local đã được tạo. **Không chạy setup lần nữa, không dùng reset.** Start kiểm tra engine, áp migration và seed chỉ khi database mới, sau đó build/start services. First start có thể lâu vì tải images/build dashboard. Nếu engine chưa phản hồi, lệnh dừng sau 15 giây ở bước kiểm tra engine.

Client URL: `http://127.0.0.1:8888/s/sv/GameLoaders.swf?isExpand=true`. Nếu projector không tự mở URL: File → Open → dán URL này.

## Vận hành

Có thể double-click `MCGAME.bat` tại thư mục gốc để dùng menu sáu mục: Start(build), Stop(auto backup DB), Reset=Stop+Start, mở Flash 30 FPS, Restore DB và chọn Local/LAN/Radmin/Online VPS. Backup timestamp nằm ở `.local/backups/database`; snapshot full-live để Git theo dõi là `supabase/backups/mcgame-live.dump`. Restore luôn tạo safety backup trước, nhưng chưa được live-test vì thao tác này thay toàn bộ database. Online VPS chỉ mở HTTP/RTMPE game và không tự cấu hình TLS/firewall/security hardening.

| Lệnh | Chức năng |
|---|---|
| `.\bin\game.exe start` | Migration an toàn, build/start, kiểm tra hạ tầng |
| `.\bin\game.exe stop` | Stop containers, giữ toàn bộ dữ liệu, timeout 60s |
| `.\bin\game.exe restart` | Restart containers, không seed/migration lại |
| `.\bin\game.exe status` | Container status, profile URL, IP/subnet LAN |
| `.\bin\game.exe logs` | 100 dòng gần nhất: game/edge/DB/Redis/dashboard |
| `.\bin\game.exe check` | Docker engine, Compose status, Postgres readiness, Redis PING, nginx config |
| `.\bin\game.exe client` | Mở Flash Projector bằng profile đang chọn |

Check chỉ kiểm tra hạ tầng, không chứng minh login/gameplay hoặc persistence. Restart không áp profile mới; đổi profile phải dùng Start để recreate mapping/mount.

Build lại công cụ khi sửa source:

```powershell
go build -o bin/game.exe ./cmd/game
go build -o bin/gameserver.exe ./cmd/gameserver
```

CLI tự tìm Supabase trong `bin/`. Docker phải có trên PATH của PowerShell mới.

## Database và test account

Đăng ký game hiện có qua nút **Đăng ký** của Flash hoặc `/register.html` ở IP profile đang bật. [Luồng đăng ký Radmin](radmin.md#đăng-ký-account) hỗ trợ ID1/password1; không cần mở Supabase Auth/DB/admin cho người chơi. Trang/API đã deploy và kiểm tra host-side; real Flash login/creation/persistence vẫn cần kiểm thử bằng account do người vận hành tự chọn.

Chỉ dùng Supabase/Postgres của Compose; không dùng `supabase start`, database cloud, reset hoặc xóa volumes. Seed account thuộc `supabase/seeds/accounts.sql`; chưa xác minh account trên DB vì engine bị chặn. Không coi việc có seed file là account test đã hoạt động. Không ghi password/token vào tài liệu.

Sau khi stack chạy, kiểm tra migration history, số lượng static templates, tạo account/character test riêng nếu cần theo auth workflow project. Kiểm thử logout/relogin và restart bằng cùng character, so sánh vị trí/quest/inventory trước–sau. Không sửa dữ liệu player đang có.

## LAN — chỉ thực hiện sau khi Local login/gameplay PASS

Ethernet hiện tại: `192.168.2.9/24`, mạng Private. Không dùng Radmin VPN `26.*`.

```powershell
$env:MCGAME_LAN_IP = '192.168.2.9'
.\bin\game.exe lan start
.\bin\game.exe client
```

IP trên là giá trị khảo sát hiện tại; kiểm tra lại bằng `Get-NetIPConfiguration` khi DHCP thay đổi. Cấu hình chỉ nhận private IPv4 thực sự gán trên máy. Khi chọn LAN, cả máy chủ và máy khách dùng URL LAN, không dùng localhost.

Chưa mở firewall vì Local chưa hoạt động. Sau Local PASS, tại PowerShell Administrator, chỉ tạo hai rule LAN:

```powershell
New-NetFirewallRule -Name MCGAME-LAN-Assets -DisplayName 'MCGAME LAN Assets' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 8888 -LocalAddress 192.168.2.9 -RemoteAddress 192.168.2.0/24 -InterfaceAlias Ethernet -Profile Private
New-NetFirewallRule -Name MCGAME-LAN-RTMPE -DisplayName 'MCGAME LAN RTMPE' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 1935 -LocalAddress 192.168.2.9 -RemoteAddress 192.168.2.0/24 -InterfaceAlias Ethernet -Profile Private
```

Kiểm tra rule cùng tên trước khi chạy lại; không mở cổng DB/Redis/admin, không tắt firewall, không port-forward router. Khi IP/subnet/adapter đổi phải cập nhật profile và hai rule. Có thể gỡ riêng hai rule do mình tạo bằng `Remove-NetFirewallRule -Name MCGAME-LAN-Assets,MCGAME-LAN-RTMPE`.

Máy thứ hai dùng Flash Projector với `http://192.168.2.9:8888/s/sv/GameLoaders.swf?isExpand=true`. Hai người chơi phải dùng hai account khác nhau. Chỉ ghi LAN READY sau khi hạ tầng và Local đạt yêu cầu; chỉ ghi LAN PASS sau kiểm thử từ máy thứ hai.
## Opt-in 30 FPS trial

The original client remains the default. An experimental copy is available through root `MCGAME-30FPS.bat` (`bin/game.exe client30`). Start Docker Desktop and the existing stack first; close other Flash windows before login. Return to the original through `MCGAME.bat` option 2.

Regeneration, if outputs are absent: `go build -o bin/clientfps.exe ./cmd/clientfps`, then `bin/clientfps.exe` from repository root. This command refuses existing outputs/source checksum mismatch and never replaces original SWFs. Do not reset player data for FPS testing.
