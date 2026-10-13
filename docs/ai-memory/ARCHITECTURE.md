# MCGame Architecture Memory

## Flash và RPC

- Flash gọi tên RPC động qua RTMPE/AMF0. Client decompiled tại `docs/client/` là wire contract chỉ đọc.
- `internal/infrastructure/rtmp/dispatcher.go` đăng ký và dispatch method tới handler.
- Handler nhận `[]interface{}`; số AMF0 thường là `float64` và phải parse chủ động.
- Callback về client đi qua `rtmp.Connection.SendCallback` trong `internal/infrastructure/rtmp/connection_rpc.go`.
- Method client chờ phải trả response trên mọi nhánh; thiếu registration thường hiện `[Connection] method not found`.

## Server layers

- Bootstrap: `cmd/gameserver/main.go` tạo repository, cache, service, handler và đăng ký RPC.
- Presentation: `internal/presentation/rtmp/handlers/` ánh xạ request/response đúng tên Flash.
- Application: `internal/application/` chứa nghiệp vụ và orchestration.
- Domain: `internal/domain/` chứa entity và repository interface.
- Persistence: `internal/infrastructure/persistence/postgres/` gọi function/schema PostgreSQL.

## Pet

- Entity/DTO/quality: `internal/domain/pet/pet.go`, `internal/domain/pet/quality.go`.
- Nghiệp vụ: `internal/application/pet/service.go`; item consumption chung: `item_consumption.go`.
- Fusion RPC: `internal/presentation/rtmp/handlers/pet/fusion.go`.
- Fusion production path: `pet.Service` → `pet.FusionRepository` → `cache.CachedPetRepository.ApplyFusion` → `postgres.PetRepository.ApplyFusion`.
- PostgreSQL update Pet chính và xóa Pet phụ trong một transaction; cache chỉ đổi sau commit.
- UI Pet dùng cả response responder và callbacks như `onDelPet`, `onUpdatePet`.

## Inventory và Item Use

- Entity/SID: `internal/domain/item/item.go`.
- Service: `internal/application/item/service.go`.
- Item handlers: `internal/presentation/rtmp/handlers/item/`.
- Stack sync phổ biến: `onDelCharactorSlot` khi hết stack, `onUpChaSlotStackNum` khi còn stack.
- Tiêu dao phù: `handlers/item/transport.go` → `item.Service.UsePreferredTimedOrConsumableItem`.
- Túi tiện dụng: `item/tempbag_install.go`, `item/tempbag_login.go`, handler `handlers/item/tempbag_install.go`.

## Shop

- Nghiệp vụ: `internal/application/shop/service.go`; handler: `handlers/shop/purchase.go`, `vip.go`.
- `TBL_SHOP_SLOT.point` hiện dùng `Character.ExPoint`, client key `exPoint`.
- Per-character mutex chặn hai purchase đồng thời trong cùng process.
- Sau mua, quest được refresh im lặng qua `QuestNotifyHelper.RefreshQuestProgressSilently`; không phát `onAddChaQuest`.
- Transaction DB chung ví + item: UNKNOWN/chưa có.
- Legacy `clickShop(npcID)` phải tra `TBL_NPC.shop_id` rồi push `onOpenShop`; argument không phải shop ID.

## Blood bag

- `buyBloodBag({bt,bn})` được xử lý trong `handlers/shop/convenience.go` và `application/bloodbag/service.go`.
- Bốn reserve lưu dưới feature key `blood_bag`; login `ep` là chuỗi `|characterHp|characterMp|petHp|petMp`.
- PostgreSQL repository commit `character_wallet.gold` và `character_stat_features` trong cùng transaction, sau đó đồng bộ character cache.

## Quest

- Service: `internal/application/quest/service.go`; handler: `handlers/quest/`.
- `onAddChaQuest` vừa cập nhật quest vừa hiện thông báo nhận quest ở Flash, nên không dùng cho refresh sau mua Shop.
- Capture Pet refresh: `handlers/combat/commands.go` gọi `QuestNotifyHelper.PushQuestProgress` khi có Pet reward mới.
- Quest Admin nằm ở `dashboard/src/app/api/quests/`, `dashboard/src/app/dashboard/quests/` và SQL dưới `supabase/data/`/`supabase/migrations/`.

## PostgreSQL, Supabase và cache

- Schema chính: `public` auth, `player` runtime, `data` catalog.
- Runtime repository thường được bọc bởi `internal/infrastructure/cache/`.
- `PlayerData` clone entity khi đọc/ghi; dirty snapshot được flush tại `PlayerCache.persistPlayerData`.
- Item/Pet mới cần ID nên create có thể ghi DB ngay; update thường write-behind khi người chơi online.
- Mọi migration phải qua compose database và Supabase CLI theo `CLAUDE.md`.

## Đồng bộ client

- Response responder cập nhật UI panel cục bộ; callback cập nhật inventory/Pet/quest dùng chung.
- Chỉ ghi DB/cache mà không gửi callback thường tạo hiện tượng “đăng nhập lại mới thấy”.
- Callback payload phải giữ chính xác field client đọc; không đổi tên để “chuẩn hóa”.
- `onRefreshPetProp` nhận hai positional args `(petID, property)`, không nhận wrapper `{id,s}`.
