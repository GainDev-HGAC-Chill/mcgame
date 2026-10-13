# Chuyển dữ liệu project khỏi ổ C — 2026-10-11

## Kết quả

Ổ C ban đầu chỉ trống khoảng 0,8 GB; sau chuyển trống khoảng 30,3 GB (GiB theo công cụ). Ổ I còn khoảng 64,4 GB. Người vận hành chọn **chỉ chuyển, không gỡ ứng dụng**, cho phép dừng Docker/game, và chọn giữ các bộ cài nhỏ trong Program Files vì chưa cấp quyền Administrator.

| Dữ liệu | Vị trí vật lý mới |
|---|---|
| Docker data disk (~28 GiB) | `I:\mcgame\.local\docker\disk\docker_data.vhdx` |
| Go build cache (~1,15 GiB) | `I:\mcgame\.local\go\cache` |
| Go workspace/module cache | `I:\mcgame\.local\go\workspace` |
| Các file nghiên cứu `mcgame*` | `I:\mcgame\.local\research` |

Các thư mục cũ dưới `%LOCALAPPDATA%\Docker\wsl\disk`, `%LOCALAPPDATA%\go-build`, `%USERPROFILE%\go` và thư mục nghiên cứu cũ được giữ bằng NTFS junction trỏ sang I. Các file nghiên cứu lẻ được chuyển trực tiếp. Docker settings/WSL registration không đổi: đây là chuyển qua junction, không phải thay đổi Disk image location trong giao diện Docker.

Go GOCACHE/GOPATH/GOMODCACHE được cập nhật; biến GOPATH cấp User cũng trỏ sang I. Terminal/IDE đang mở trước đó có thể vẫn giữ môi trường cũ, nhưng đường dẫn cũ có junction tương thích. Go/Java/FFDec/Docker Desktop executables vẫn ở C theo lựa chọn người vận hành; chưa gỡ Riot/XAMPP/SQL Server/Visual Studio hay ứng dụng khác.

## Bảo vệ dữ liệu

**`.local/docker` là dữ liệu thật, không phải cache có thể xóa.** `.local/` được gitignore để không commit database/tools. Không xóa thư mục project, không chạy `git clean -fdx`, không xóa/prune Docker volumes hoặc factory reset. Giữ ổ I khả dụng khi chạy Docker/Go. Junction không phải bản sao lưu; cần backup database riêng trước thao tác rủi ro.

`.dockerignore` cũng loại `.local/` khỏi build context: không đưa VHD/database backup/cache vào Docker image. Không bỏ quy tắc này khi chỉnh build.

Tool Go `cmd/relocate` (build `go build -o bin/relocate.exe ./cmd/relocate`) chỉ dùng cho thư mục offline. Nó từ chối target đã tồn tại, copy từng file, so SHA256 nguồn đọc/copy đích, lưu manifest ở I, rename nguồn sang backup, tạo junction, xác nhận cùng directory identity rồi xóa từng file nguồn đã copy. Không dùng cho ứng dụng/database đang ghi. Nếu lỗi, giữ dữ liệu còn lại để xử lý; không tự chạy lại trên target có sẵn.

Docker đã tự dừng giữa kiểm tra ban đầu và lệnh Compose stop (engine pipe biến mất); Compose stop không hoàn tất. Sau đó xác nhận Docker không chạy, WSL không có distro đang chạy và VHD mở độc quyền được trước khi copy. Không dùng force-kill hoặc reset. SHA256 toàn bộ data VHD trước/sau copy khớp; bản nguồn chỉ được bỏ sau khi junction xác minh. Sau restart, VHD có thể đổi hash do dữ liệu hoạt động bình thường.

## Kiểm chứng

- **PASS**: Docker VHD (~30.058.479.616 bytes ở thời điểm copy), Go cache/module và research copies có checksum khớp; junctions trỏ đúng I.
- **PASS**: Go tests `./cmd/relocate ./cmd/clientfps ./cmd/game`, vet tool chuyển dữ liệu; smoke test junction/read-through.
- **PASS**: Docker engine 29.8.2, 10 container hiện hữu chạy lại; health checks có cấu hình trở về healthy; `bin/game.exe check` PASS.
- **PASS**: HTTP HEAD client gốc, loader30 và main30 đều 200.
- **NOT TESTED**: Flash login/gameplay, đo FPS, progression qua logout/relogin sau chuyển; service health không chứng minh gameplay.
- `mcgame-server` RestartCount quan sát sau chuyển = 4; chưa lấy baseline trước chuyển nên không kết luận cả bốn lần do chuyển disk. PostgreSQL RestartCount = 0 tại kiểm tra.
