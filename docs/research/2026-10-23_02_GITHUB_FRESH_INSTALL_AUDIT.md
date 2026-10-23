# GitHub fresh-install audit

## Kết luận

Source hiện chưa đủ để cam kết người khác clone GitHub rồi dựng đầy đủ trên máy mới. Chuỗi migration có thể replay, nhưng không đại diện đầy đủ cho schema đang được production code và compose DB sử dụng. Local/LAN trên máy đã cấu hình có thể hoạt động; fresh local/LAN bị chặn bởi schema thiếu và dashboard bind cứng. Public internet chưa sẵn sàng.

Follow-up cùng ngày đã thêm full-live snapshot `supabase/backups/mcgame-live.dump` theo lựa chọn rõ ràng của operator. Snapshot này giúp clone mới restore đúng trạng thái compose DB hiện tại, nhưng không làm migration/seed trở nên đầy đủ và cũng không bổ sung năm Achievement function vốn đang thiếu ngay trong live DB. Snapshot chứa Auth/session/player data và phải được coi là public sau khi push.

## Bằng chứng đã kiểm tra

- 103 migration, không trùng version. Version legacy `20261016` không đủ 14 chữ số nhưng đã nằm trong history nên phải giữ tương thích.
- `supabase migration list` khớp local/remote; `db push --include-all --dry-run` báo up to date.
- `supabase db diff` dựng shadow và replay toàn bộ migration thành công.
- 101 seed file, không file rỗng, tổng khoảng 15.8 MB; `supabase/config.toml` dùng `./seeds/*.sql` cho fresh database.
- Seed `data_tbl_pet_soul.sql` có đủ row `751..760` với `Vĩnh Sinh`/`Tăng Miễn Tử`. Compose DB hiện hữu vẫn có `永生`/`Tăng免死`, chứng minh seed mới không tự cập nhật deployment cũ.
- Không có tracked file vượt giới hạn GitHub 100 MB; file tracked lớn nhất được thấy là `gameserver.exe` khoảng 34.97 MB.

## SQL blocker

Static audit tìm 220 lời gọi Postgres function trong production persistence. Sáu function không có trong migration:

1. `player.list_character_achievement_requirements`
2. `player.record_character_achievement_event`
3. `player.reserve_character_achievement_claim`
4. `player.finalize_character_achievement_claim`
5. `player.release_character_achievement_claim`
6. `player.consume_catch_ball_round`

Năm Achievement function và ba bảng hỗ trợ chỉ nằm trong `supabase/data/achievement_system.sql`; chúng cũng chưa có trong compose DB hiện tại. `consume_catch_ball_round` có trong compose DB nhưng không có migration hoặc draft tracked.

Hai function `data.box_item_award_revision` và `data.box_item_admin_save` có trong compose DB nhưng chỉ nằm trong draft `supabase/data/selectable_gift_bag_admin.sql`, chưa có migration. Fresh clone vì vậy thiếu backend save cho dashboard Túi quà.

Shadow-to-live diff còn phát hiện:

- bảng `player.character_deletion_requests` và các function schedule/list/pending/restore/purge chỉ có ở live DB;
- definition drift của `public.npc_admin_save`, `public.npc_admin_snapshot`, `public.quest_admin_validate` và `player.seed_findback_missing_day`;
- NPC position/hardening draft mới hơn migration chain.

File review diff tạm thời: `C:\Users\gaingain\AppData\Local\Temp\opencode\mcgame-fresh-install-diff-expanded.sql`. Đây là artifact audit, không phải migration portable để apply.

## Mức sẵn sàng triển khai

| Mục tiêu | Trạng thái | Lý do chính |
|---|---|---|
| Local trên deployment hiện tại | Có điều kiện | Stack hiện tại có direct-applied SQL, nhưng Achievement vẫn thiếu và Soul localization live còn cũ. |
| Fresh local | Chưa sẵn sàng | Migration thiếu object runtime; dashboard bind thêm IP `26.31.111.123` có thể fail trên máy không sở hữu IP đó. |
| LAN/private VPN | Chưa portable | Tool chỉ có profile local/LAN/Radmin; dashboard bind/origin dùng IP Radmin của máy hiện tại thay vì profile-derived config. |
| Public internet | Không sẵn sàng | Không có public profile, TLS/domain/reverse-proxy/security hardening; project chưa qua security audit. |

## Việc cần làm trước khi công bố fresh-install ready

1. Với sự cho phép của operator, apply các draft SQL cần thiết vào compose DB, smoke-test và capture bằng `supabase db diff -f ...`; không hand-write migration.
2. Khôi phục/capture definition của `consume_catch_ball_round`, Achievement, box admin, character deletion và các live drift đã xác minh.
3. Chuyển dashboard bind/origin sang biến profile; local chỉ bind localhost, LAN/Radmin dùng IP được profile chọn, không hardcode `26.31.111.123`.
4. Dựng một database disposable mới chỉ từ migration + seed, chạy integration smoke và so schema với baseline mong đợi.
5. Nếu hỗ trợ online, thiết kế profile riêng với TLS, domain/reverse proxy, firewall, secret rotation, rate limiting và security review trước khi mở cổng public.
