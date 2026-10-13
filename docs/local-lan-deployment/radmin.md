# Chơi qua Radmin VPN trên Windows

Hai máy phải cài Radmin VPN và tham gia cùng một mạng Radmin. Chỉ máy chủ cần Docker/Go/source; máy người chơi chỉ cần Radmin và Flash Projector. Không cần port-forward router, không tắt Windows Firewall.

## Máy chủ

Mở Docker Desktop với Linux engine, bật Radmin VPN, chạy tại thư mục source:

```powershell
go build -o bin/game.exe ./cmd/game
.\bin\game.exe radmin start
```

`radmin` tự tìm IPv4 `26.x.x.x` trên adapter Radmin đang bật. Nếu có nhiều IP, chọn rõ bằng `$env:MCGAME_RADMIN_IP = '26.31.111.123'` rồi chạy lại. IP này là địa chỉ khảo sát của máy chủ hiện tại, không dùng cố định cho máy khác.

Profile tạo `docker/client/radmin.xml`, đổi cả resource HTTP và logic RTMPE sang IP Radmin, chỉ bind hai cổng game tới IP đó. DB/Redis/Studio/dashboard/metrics vẫn ở localhost. `start` áp profile bằng Compose; `restart` không áp mapping mới. Không chạy setup/reset và không xóa volume để đổi mạng.

Compose truyền `MCGAME_GATEWAY_PUBLIC_HOST` vào monolith: URL kênh có host localhost/loopback được thay bằng IP profile khi trả danh sách kênh. Scheme, port và đường dẫn giữ nguyên; URL kênh đã trỏ máy khác không bị ghi đè. Không thay dữ liệu runtime config trong database.

Cũng có thể chọn **R** trong `MCGAME.bat`, sau đó chọn **Start server**.

## Firewall — PowerShell Administrator

Chỉ cho phép IP Radmin của người chơi tin cậy; thay IP ví dụ bên dưới bằng IP thực trên máy người chơi. Rule giới hạn adapter Radmin, IP máy chủ và TCP 8888/1935. `Profile Any` cần thiết khi Windows phân loại adapter VPN là Public; không mở mọi adapter hay mọi cổng.

```powershell
$serverIP = '26.31.111.123'
$peerIPs = @('26.10.20.30')
$adapter = 'Radmin VPN'
$rule = 'MCGAME-Radmin-Game'
if (Get-NetFirewallRule -Name $rule -ErrorAction SilentlyContinue) {
    throw 'Rule đã tồn tại; kiểm tra trước khi cập nhật.'
}
New-NetFirewallRule -Name $rule -DisplayName 'MCGame Radmin HTTP + RTMPE' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 8888,1935 -LocalAddress $serverIP -RemoteAddress $peerIPs -InterfaceAlias $adapter -Profile Any
```

Không mở DB `54322`, Redis `6379`, Studio `54323` hoặc dashboard `3000` cho người chơi. Nếu đổi IP máy chủ/người chơi, phải cập nhật profile/rule tương ứng. Gỡ riêng rule trên bằng `Remove-NetFirewallRule -Name MCGAME-Radmin-Game`.

## Máy người chơi

### Đăng ký account

Nhấn **Đăng ký** ở màn hình đăng nhập Flash, hoặc mở bằng trình duyệt:

```text
http://26.31.111.123:8888/register.html
```

Điền ID, mật khẩu và xác nhận mật khẩu, rồi quay lại Flash để đăng nhập. ID `1` / mật khẩu `1` được phép nếu ID chưa tồn tại; không có quy tắc mật khẩu mạnh/độ dài tối thiểu. Không tự cắt khoảng trắng/đổi hoa thường. Trần tương thích là ID50/mật khẩu20 đơn vị UTF-16; emoji có thể tính2. ID trùng báo lỗi, không ghi đè account hay đổi mật khẩu.

Sau đăng nhập, tạo nhân vật mới để nhận cấp90,999999 vàng và999999 kim phiếu. Đăng ký account không tự tạo nhân vật hoặc cộng tiền; nhân vật cũ không được cấp lại.

Chỉ dùng trên mạng Local/Radmin tin cậy; HTTP không bảo vệ mật khẩu ở tầng web, không dùng lại mật khẩu quan trọng. Đăng ký có giới hạn5lượt/phút/IP +burst3 để tránh spam;429 thì chờ ít nhất12giây. Không mở DB/GM/Data API và không cần host port mới. Kiểm tra hiện tại page/API host-side PASS, account/Flash đăng nhập thật và máy thứ hai vẫn cần người vận hành xác nhận.

Kiểm tra kết nối trong PowerShell:

```powershell
Test-NetConnection 26.31.111.123 -Port 8888
Test-NetConnection 26.31.111.123 -Port 1935
```

Mở Flash Player projector, **File → Open**, nhập:

```text
http://26.31.111.123:8888/s/sv/GameLoaders.swf?isExpand=true
```

Máy chủ cũng chơi bằng URL này khi profile Radmin đang bật. Mỗi người phải dùng account riêng đã được tạo; không chia sẻ account GM hoặc mật khẩu quản trị. Trang quản trị chỉ mở trên máy chủ: `http://127.0.0.1:3000`.

## Kiểm chứng

`game check` chỉ xác nhận hạ tầng. Chỉ ghi Radmin gameplay PASS sau khi máy thứ hai tải client, login, chọn nhân vật, vào cảnh/chơi cùng, logout/relogin và xác nhận lưu tiến độ. Thành công HTTP trên máy chủ không chứng minh máy thứ hai kết nối được. Xem [kết quả](test-results.md).

## Quay lại Local

```powershell
.\bin\game.exe local start
```

Lệnh giữ dữ liệu, đưa game về localhost. Rule Radmin có thể gỡ riêng nếu không dùng nữa.
