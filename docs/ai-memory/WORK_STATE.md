# Current Work State

## Mục tiêu phiên

- Hoàn tất bộ nhớ AI và bốn lỗi: găng thu thập, túi cứu thương HP/MP, timed transport disconnect, Pet fusion quality/tooltip/random.

## Đã hoàn thành

- Tạo và xác minh `docs/ai-memory/`; bổ sung workflow vào `CLAUDE.md`/`AGENTS.md` symlink.
- `clickShop(275)` tra NPC và mở shop 32; không mua/phát item trực tiếp.
- `buyBloodBag` validate `{bt,bn}`, cap/giá Gold, transaction DB chung Gold + reserve, callback và login `ep`.
- Timed transport giữ item 7/30 ngày, không gia hạn/tái kích hoạt, ưu tiên pass trước consumable.
- Pet fusion validate 1+4 cùng template/tier, tỷ lệ 100/80/50/30, đúng một bậc; DTO/callback tooltip đã đồng bộ.

## File đang sửa/chưa commit

- Toàn bộ thay đổi vẫn uncommitted/unpushed; workspace có nhiều thay đổi cũ ngoài phạm vi, không reset/revert/stash.
- File mới chính: `internal/application/bloodbag/`, `handlers/shop/convenience.go`, focused tests và plan `docs/plans/2026-10-13_01_GAMEPLAY_RPC_TRANSPORT_PET_FIXES.md`.

## Kiểm tra đã chạy

- Focused bloodbag/shop/statfeature/item/transport/equip/pet/fusion/domain/cache/postgres tests PASS.
- Full shop và item/pet handler packages PASS.
- Targeted `go vet` PASS.
- `go build -o bin/gameserver ./cmd/gameserver` PASS.
- `git diff --check` sạch; chỉ cảnh báo LF/CRLF.
- Full `internal/application/pet` còn 4 baseline failures về stat/equipment/skill; full `internal/application/item` còn baseline failures về award quality. Không sửa ngoài phạm vi.

## Runtime/database

- `bin/gameserver` đã build bản mới; container `mcgame-server` chưa restart và vẫn cần deploy được cho phép.
- Không có migration mới: blood bag dùng bảng wallet/stat-feature hiện có.
- Không live-test và không thay đổi dữ liệu người chơi.

## Rủi ro/chưa xác minh

- Cần Flash smoke-test cả bốn luồng; riêng transport log cũ chứng minh client disconnect nhưng không có stack trace Flash nên exception cụ thể vẫn UNKNOWN.
- Chưa kiểm chứng tự động tiêu hao reserve blood bag sau battle; RPC `useBloodBag` cũ ngoài phạm vi mua và cần contract riêng trước khi đổi.
- Shop purchase chung vẫn chưa atomic xuyên DB giữa ví và inventory; không liên quan transaction blood-bag mới.
- Quest 166/Pet Nấm Lục và Quest Admin giữ nguyên trạng thái cũ, không dùng để test.

## Bước tiếp theo

1. Khi được phép, rebuild/restart riêng `mcgame-server`.
2. Smoke-test an toàn: click mua găng→shop 32; mua blood bag partial/full/relog; timed scroll 7/30/premium; fusion bằng test Pet và kiểm tra tooltip ngay.
3. Nếu wire test pass, mới cân nhắc quiet hai RPC mới; hiện vẫn để dispatcher log.
