# MCGame — 2026-10-13 — Xóa nhân vật ngay / 30 FPS URL cũ

## Yêu cầu
1. Xóa **vĩnh viễn ngay** sau khi xác thực mật khẩu cấp 2 (không còn chờ 7 ngày).
2. Giữ URL bạn bè đã nhận: `/s/sv/GameLoaders30.swf?isExpand=true`.
3. Chuẩn hóa source, Supabase migration, và lịch sử chỉnh sửa.

## Thay đổi Go
- `StartDeleteChar` vẫn nhận `(characterID, MD5 secondary password)` theo contract Flash.
- Kiểm tra phiên đã đăng nhập, tài khoản sở hữu nhân vật, mật khẩu cấp 2, và nhân vật không online.
- Gọi `player.delete_character_immediately(account_id, character_id)`; PostgreSQL kiểm tra quyền sở hữu và khóa hàng rồi gọi `player.delete_character(id)` trong cùng giao dịch.
- Nếu xóa thành công, làm mất hiệu lực cache, làm mới danh sách nhân vật.
- Không còn `delTime`, không còn API khôi phục hay bộ đếm 7 ngày.
- Cấu hình `MCGAME_CHARACTER_PURGE_ENABLED` được xóa; không cần cron xóa sau 7 ngày.
- Không cho client gọi trực tiếp SQL; chỉ server đã xác thực mới gọi.

## Thay đổi client
- File chính 30 FPS đã xác minh (`sha256 7159ef...`) vẫn được giữ nguyên.
- Sinh file mới có tên `6a41a5ba5c0006ab4e0c1321a2b05032` cùng nội dung với file đã bật nút Xóa.
- Thay nội dung **GameLoaders30.swf** trỏ tới file `...05032` (đúng ba tham chiếu), nhưng giữ **nguyên URL GameLoaders30.swf** để người chơi dùng link đã chia sẻ.
- Đặt `Cache-Control: no-store` riêng loader 30 trong Nginx; reload Nginx khi được phép.
- Các client đã cache loader cũ *trước lần cập nhật này* có thể cần xóa Flash cache hoặc thêm tham số `&v=...` một lần.

## Database
- Migration `20261013160000_character_delete_immediate.sql` **expand-only**, không xóa các bảng/hàm cũ trước khi code mới chạy.
- Có preflight báo lỗi nếu còn bất kỳ bản ghi chờ xóa 7 ngày. Khôi phục/xử lý theo từng nhân vật trước khi migrate.
- Có SQL cleanup tùy chọn tại `tools/character_deletion_legacy_cleanup.sql`, chỉ chạy khi đã triển khai và kiểm chứng server mới, không có bản ghi chờ xóa.
- **Không reset database, không tự xóa nhân vật khi cài bản vá.**

## Kiểm chứng và còn cần thực hiện
- File mới được kiểm tra tĩnh bằng `gofmt`, `py_compile` và patcher SWF kiểm tra cấu trúc; chưa chạy tích hợp Flash/Go/PostgreSQL trong môi trường của người dùng.
- Bắt buộc kiểm tra dry-run, backup, migration, build, `go test`, triển khai vào thời điểm cho phép, sau đó dùng nhân vật test để thử xóa ngay.
- Xóa thật là không thể khôi phục bằng nút Restore. Chỉ có backup PostgreSQL để phục hồi thủ công.
- Migration cleanup chính thức chuẩn bị sẵn ở `post-deploy-migration/20261013160001_character_deletion_remove_7day.sql`; chỉ đưa vào `supabase/migrations` sau khi chuyển server.
