# Current Work State

## Mục tiêu phiên

- Hoàn tất bộ nhớ AI và bốn lỗi: găng thu thập, túi cứu thương HP/MP, timed transport disconnect, Pet fusion quality/tooltip/random.
- Tiếp tục nhóm lỗi mới: callback/fusion Pet, nâng skill Pet, auto-skill, ranking, Quest Admin, xóa nhân vật và túi chế tạo.

## Đã hoàn thành

- Tạo và xác minh `docs/ai-memory/`; bổ sung workflow vào `CLAUDE.md`/`AGENTS.md` symlink.
- `clickShop(275)` tra NPC và mở shop 32; không mua/phát item trực tiếp.
- `buyBloodBag` validate `{bt,bn}`, cap/giá Gold, transaction DB chung Gold + reserve, callback và login `ep`.
- Timed transport giữ item 7/30 ngày, không gia hạn/tái kích hoạt, ưu tiên pass trước consumable.
- Pet fusion validate 1+4 cùng template/tier, tỷ lệ 100/80/50/30, đúng một bậc; DTO/callback tooltip đã đồng bộ.
- Đã sửa lại theo bằng chứng Flash runtime: `onRefreshPetProp` dùng một wrapper `{id,s}`; fusion đổi `GrowRate` cơ bản ngẫu nhiên trong bậc kế tiếp và không cộng fusion vào `GrowRateAdd`.
- Đã thêm RPC `petUpSkill`, tiêu hao manual 1365/1366/1367 và callback inventory/Pet.
- Đã giữ `pi`/`soulInfo` ở root wrapper Pet DTO và bổ sung `bt8`, tránh lỗi `Core.updatePetSetting` sau khi mở Pet Manager.

## File đang sửa/chưa commit

- HEAD `2b550d0` đã chứa nhóm Pet/blood-bag/transport/shop và bộ nhớ trước đó; không commit/push thêm.
- Working tree còn nhóm character-delete/client artifact do luồng khác để lại và cập nhật bộ nhớ phiên này. Không reset hoặc ghi đè nhóm này.

## Kiểm tra đã chạy

- Focused bloodbag/shop/statfeature/item/transport/equip/pet/fusion/domain/cache/postgres tests PASS.
- Full shop và item/pet handler packages PASS.
- Targeted `go vet` PASS.
- `go build -o bin/gameserver ./cmd/gameserver` PASS.
- `git diff --check` hiện chỉ fail vì trailing whitespace trong `docs/client/.../CharSelectCanvas.as` do thay đổi ngoài luồng; không sửa decompiled ActionScript.
- Full `internal/application/pet` còn 4 baseline failures về stat/equipment/skill; full `internal/application/item` còn baseline failures về award quality. Không sửa ngoài phạm vi.
- Focused fusion/random/DTO/`petUpSkill`/pet handler tests PASS sau khi khôi phục phiên.
- Auth handler + PostgreSQL tests PASS cho source character-delete hiện có.
- Full `internal/application/pet` chạy lại: vẫn đúng 4 baseline failures về stat/equipment/skill; các package domain/pet, pet handler, auth và postgres PASS.
- Targeted `go vet` PASS; `go build -o bin/gameserver.exe ./cmd/gameserver` PASS.

## Runtime/database

- `bin/gameserver` đã build bản mới; container `mcgame-server` chưa restart và vẫn cần deploy được cho phép.
- Không có migration mới: blood bag dùng bảng wallet/stat-feature hiện có.
- Không live-test và không thay đổi dữ liệu người chơi.

## Rủi ro/chưa xác minh

- Cần Flash smoke-test cả bốn luồng; riêng transport log cũ chứng minh client disconnect nhưng không có stack trace Flash nên exception cụ thể vẫn UNKNOWN.
- Chưa kiểm chứng tự động tiêu hao reserve blood bag sau battle; RPC `useBloodBag` cũ ngoài phạm vi mua và cần contract riêng trước khi đổi.
- Shop purchase chung vẫn chưa atomic xuyên DB giữa ví và inventory; không liên quan transaction blood-bag mới.
- Quest 166/Pet Nấm Lục và Quest Admin giữ nguyên trạng thái cũ, không dùng để test.
- Ranking đã xác nhận `rankGet` là stub và nhiều category thiếu semantics; chưa sửa bằng dữ liệu suy đoán.
- Túi chế tạo crash tại template lookup của shop catalog runtime; catalog checked-in không có row hỏng nên cần ID row/template từ runtime trước khi sửa.
- Quest Admin “chưa hoàn chỉnh” chưa nêu field/workflow cụ thể; source hiện có editor và có thể deployment đang cũ.

## Bước tiếp theo

1. Cần Flash smoke-test callback/fusion, `petUpSkill`, auto-skill và character-delete sau khi được phép deploy/restart riêng server.
2. Cần runtime ID cho row/template làm crash túi chế tạo và yêu cầu cụ thể cho ranking/Quest Admin trước khi triển khai tiếp.
3. Ranking cần repository/query và định nghĩa chính xác cho category tuần/tháng trước khi thay stub bằng dữ liệu thật.

## 2026-10-13 - Immediate Character Delete / 30 FPS Shared Link
- Added immediate delete RPC using ownership-guarded SQL, secondary MD5 validation and cache invalidation.
- Kept GameLoaders30.swf public URL, using fresh internal main asset 32 to bypass old main cache.
- Migration and cleanup are staged; live rollout must be independently verified.
