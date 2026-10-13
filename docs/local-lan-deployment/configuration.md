# Cấu hình

## Profile không sửa gameplay/client source

`game local` sinh `docker/client/local.xml`; `game lan` sinh `docker/client/lan.xml`. Các file machine-specific được git-ignore. Template `frontend/s/sv/profile/config.xml` giữ nguyên.

`docker/.env` (git-ignore, không công bố secrets) chứa:

- `GAME_BIND_ADDRESS`: `127.0.0.1` Local hoặc private IPv4 Ethernet được chọn cho LAN.
- `GAME_CLIENT_CONFIG_PATH`: `./client/local.xml` hoặc `./client/lan.xml`, đường dẫn tính từ thư mục Compose.
- `POSTGRES_PASSWORD`, các JWT/API keys và secret dashboard: tự sinh trong setup; không ghi ra báo cáo.

Compose mount profile riêng tại `/s/sv/profile/config.xml`. Resource URL và RTMPE host cùng profile; đường dẫn logic giữ nguyên `/master/test/`. Không thay đổi RTMPE handshake/xác thực, AMF0 hoặc `.as`.

| Cổng host | Bind | Vai trò |
|---|---|---|
| 8888/TCP | Profile IP | HTTP SWF/assets, crossdomain.xml |
| 1935/TCP | Profile IP | nginx TCP passthrough → Go RTMPE |
| 54322/TCP | 127.0.0.1 | Project Postgres direct |
| 54323/TCP | 127.0.0.1 | Studio/MCP |
| 6379/TCP | 127.0.0.1 | Redis |
| 3000/TCP | 127.0.0.1 | Admin dashboard |
| 8000/TCP | 127.0.0.1 | Supabase API |
| 2112/TCP | 127.0.0.1 | Metrics |

Optional pooler ports cũng giới hạn localhost. Chỉ hai cổng game được dùng qua LAN khi profile LAN được áp; không bind game lên mọi adapter/VPN. Profile LAN không publish game trên loopback: máy chủ cũng mở URL LAN.

## Persistence và concurrency

- Postgres: bind mount `docker/volumes/db/data`, named volume `db-config`.
- Redis: append-only persistence, named volume `redis-data`.
- Stop/Restart không xóa mount hoặc volume; game có grace period 60s để shutdown.
- Mặc định RTMP max connections 1000, DB pools game/auth 50/20. Đây chỉ là cấu hình, **không phải số player đã kiểm chứng**. Chưa load-test hoặc kiểm thử hai account online.
- Chỉ seed database mới. Nếu history trống nhưng game tables đã có, công cụ dừng để tránh ghi đè; không tự reset.

Compose config đã validate; toàn bộ routing/mount/health/persistence thực tế vẫn cần kiểm tra sau restart Windows.
## Experimental client FPS profile

`MCGAME-30FPS.bat` selects a separate `GameLoaders30.swf` and main asset; it does not edit XML/IP bindings or server tick settings. Target is 30 FPS, not a guaranteed measured frame rate. Default `MCGAME.bat` still selects `GameLoaders.swf`. Some frame-driven animations may be about 25% faster; this trial is not timing-certified. Details: `docs/research/2026-10-11_08_FPS30_TRIAL_RESEARCH.md`.
