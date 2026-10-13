# MCGame AI Memory Index

Đọc file này trước khi sửa lỗi. Bộ nhớ giúp khoanh vùng nhanh nhưng source và test hiện tại luôn là bằng chứng cuối cùng.

## Kiến trúc chính

- Flash client decompiled, chỉ đọc: `docs/client/`.
- RTMPE/AMF0 dispatcher: `internal/infrastructure/rtmp/dispatcher.go`.
- RPC handlers: `internal/presentation/rtmp/handlers/<module>/`.
- Application services: `internal/application/<module>/`.
- Domain entities/repository contracts: `internal/domain/`.
- PostgreSQL repositories: `internal/infrastructure/persistence/postgres/`.
- Redis/player write-behind cache: `internal/infrastructure/cache/`.
- Bootstrap/wiring: `cmd/gameserver/main.go`.
- Dashboard: `dashboard/`; compose/Supabase stack: `docker/`, `supabase/`.

## Bộ nhớ

- [Kiến trúc đã xác minh](ARCHITECTURE.md)
- [Lịch sử sửa lỗi](FIX_HISTORY.md)
- [Bản đồ module](MODULE_MAP.md)
- [Trạng thái công việc mới nhất](WORK_STATE.md)

## Hệ thống đã phân tích

- Shop và điểm thưởng `ExPoint`.
- Pet fusion thường/cao cấp, cache và transaction PostgreSQL.
- Pet cultivation item consumption và callback inventory.
- Transport scroll/tiêu dao phù có thời hạn.
- Túi tiện dụng và payload `tBag`.
- Quest lifecycle, quest notification và Quest Admin.
- Quest 166 bắt Nấm Lục.

## Quy tắc sửa source

- Đọc `WORK_STATE.md`, rồi tra đúng chủ đề trong `MODULE_MAP.md` và `FIX_HISTORY.md`.
- Chỉ mở phần `ARCHITECTURE.md` liên quan; không đọc mọi Markdown theo thói quen.
- Không sửa `.as`; dùng client decompiled làm wire contract.
- Không tin tài liệu nếu source đã thay đổi hoặc có dấu hiệu tài liệu cũ.
- Không reset/revert thay đổi chưa commit; workspace thường chứa nhiều công việc song song.
- Không live-test bằng Pet/vật phẩm thật nếu có nguy cơ tiêu hao.
- Không restart container khi chưa được phép.
- Build Go chỉ vào `bin/`.

## Việc chưa hoàn thành

- Sáu lỗi cũ vẫn cần live-test trong Flash; xem `FIX_HISTORY.md`.
- Bốn lỗi ngày 2026-10-13 đã code-fix và automated-test; vẫn cần Flash smoke-test sau deploy.
- Shop chưa có transaction DB chung cho cả ví và inventory.
- Túi tiện dụng chưa xác minh RPC tháo túi trực tiếp từ client.
- Quest Admin migration đã áp dụng local nhưng checkpoint trước ghi migration history chưa đăng ký.
- Flash quest catalog publication/rebuild vẫn BLOCKED.
- Bản build mới chưa được deploy/restart vào `mcgame-server`.
- Full item/pet application suites còn baseline failures ngoài bốn lỗi; xem `WORK_STATE.md`.
