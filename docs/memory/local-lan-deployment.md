# Windows Local/LAN Deployment

## Radmin VPN

2026-10-12: thêm `game radmin` và menu R trong `MCGAME.bat`. Auto-detect/explicit `MCGAME_RADMIN_IP` chỉ nhận IPv4 26/8 gán trên adapter tên chứa Radmin đang bật; fail closed khi không có/nhiều địa chỉ hoặc IP không thuộc adapter. LAN vẫn chặn 26/8. Sinh `docker/client/radmin.xml`, dùng Compose GAME_BIND_ADDRESS/GAME_CLIENT_CONFIG_PATH hiện có để bind riêng HTTP8888/RTMPE1935 và đổi resource/logic client; không đổi gameplay/DB/secrets. Firewall hướng dẫn giới hạn adapter/IP máy chủ/IP từng peer, không mở admin/DB/cache. Xem [Radmin](../local-lan-deployment/radmin.md); remote gameplay chỉ PASS khi kiểm thử máy thứ hai.

Monolith nhận `MCGAME_GATEWAY_PUBLIC_HOST` từ GAME_BIND_ADDRESS để DBLineProvider rewrite host loopback/localhost trong URL RTMP/RTMPE trả về onLineList/getLineInfo; giữ protocol/port/path/query và không override địa chỉ external/split server. Không UPDATE public gateway line tables. Thay profile cần recreate game server để áp advertised host, ngoài edge mapping/mount.

2026-10-11: workflow và kết quả thực tế trong [docs/local-lan-deployment](../local-lan-deployment/setup.md). Go source yêu cầu 1.26, không còn 1.23. Công cụ Go `cmd/game` hỗ trợ start/stop/restart/status/logs/check/client/local/lan. Profile XML sinh riêng trong `docker/client/`, cấu hình bind trong `.env`, không sửa client `.as` hoặc gameplay.

Safety: setup từ chối existing `.env`/database; migrate đọc boolean `::text` thay `t/f`, fail closed khi query lỗi hoặc game tables tồn tại mà history trống; chỉ seed fresh DB. Ports admin/DB/cache/metrics chỉ loopback, game bind profile IP. Không global builder prune, không reset/volume delete khi triển khai.

2026-10-16: `game start` gọi `supabase db push --include-all` để áp cả migration local chưa có trong history nhưng mang timestamp cũ hơn migration cuối đã áp. Điều này sửa lỗi CLI “Found local migration files to be inserted before the last migration” mà không reset/reseed database hiện hữu; `--include-seed` vẫn chỉ dùng khi database hoàn toàn mới. Menu `MCGAME.bat` có thêm phím `F` gọi command `client30`, giữ option 2 là client gốc.

2026-10-23: `MCGAME.bat` được rút còn sáu tác vụ. Start rebuild launcher và Compose; Stop chỉ dừng sau khi full `pg_dump` custom-format được `pg_restore --list` verify, lưu timestamp + SHA-256 ở `.local/backups/database` và cập nhật snapshot Git thường `supabase/backups/mcgame-live.dump`; Reset dùng chính Stop rồi Start, không wipe volume. Restore chỉ nhận archive trong hai thư mục đã duyệt, tạo safety backup local, dừng services, recreate database, restore rồi build/start; live restore chưa chạy. Operator chọn rõ full-live snapshot dù có Auth/session/player data, nên mọi version push phải coi là public. Public fork không cho upload LFS object mới nên snapshot 3,5 MB không dùng LFS.

Profile có thêm `online`: public IPv4/DNS được advertise cho client, edge bind `0.0.0.0`, dashboard vẫn loopback; đây không phải chứng nhận public-production vì chưa có TLS/firewall/security review. Dashboard remote bind/origin không còn hardcode `26.31.111.123`, mà được cập nhật theo Local/LAN/Radmin profile.

Blocker ban đầu WSL2/restart Windows đã vượt qua; Local có phiên chơi thực tế và autosave thành công sau sửa cache. Persistence qua logout/relogin/restart và LAN chưa kiểm chứng đầy đủ; firewall chưa mở, LAN chưa READY. Các assertion failures ngoài scope vẫn được ghi riêng trong test-results.

## Bản thử 30 FPS riêng

`cmd/clientfps` tạo main asset/loader copies riêng với source SHA256 pinning, unique fixed-length bytecode patches và từ chối overwrite. `MCGAME-30FPS.bat` / `game client30` mở bản thử, không thay default client hay restart services. Quay lại qua MCGAME.bat option2. Tests/vet/build và independent FFDec inspection PASS; HTTP/live test BLOCKED ở lần kiểm tra gần nhất do Docker engine offline. Một số hoạt ảnh có thể nhanh hơn ~25%, movement jitter/RPC/autosave load NOT TESTED; không chứng nhận timing hoặc ổn định server. Xem [nghiên cứu 30 FPS](../research/2026-10-11_08_FPS30_TRIAL_RESEARCH.md).

## Chuyển dung lượng khỏi C

2026-10-11: theo phê duyệt người vận hành, chuyển Docker VHD (~28 GiB), Go cache/module và research sang `I:\mcgame\.local` bằng verified SHA256 copies + junctions. C trống từ ~0,8 lên ~30,3 GiB. Không gỡ ứng dụng; bộ cài Program Files giữ lại theo quyết định người vận hành. `.local/docker` chứa dữ liệu thật: **không xóa `.local`, không `git clean -fdx`**. Go env trỏ I; Docker khởi động lại, infra checks PASS. HTTP client gốc/trial30 nay PASS200 (thay blocker HTTP phía trên); live gameplay vẫn NOT TESTED. Chi tiết [storage](../local-lan-deployment/storage.md).
