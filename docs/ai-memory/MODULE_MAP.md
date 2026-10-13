# MCGame Module Map

| Chức năng | File chính | Hàm/handler | Phụ thuộc | Ghi chú |
|---|---|---|---|---|
| Connection/RPC | `internal/infrastructure/rtmp/dispatcher.go` | `Register`, dispatch | RTMPE, AMF0 | Thiếu registration gây `method not found` |
| Callback RPC | `internal/infrastructure/rtmp/connection_rpc.go` | `SendCallback` | Connection transport | Giữ đúng payload Flash |
| Bootstrap | `cmd/gameserver/main.go` | service/handler wiring | repositories, cache | Kiểm tra registration ở đây và `RegisterHandlers` |
| Pet model/DTO | `internal/domain/pet/pet.go` | `ToDTO`, `ToDetailDTO`, `AbsorbPet` | stats | Tooltip phụ thuộc DTO thực tế |
| Pet quality | `internal/domain/pet/quality.go` | `ColorByGrowRate` | grow rate | Bậc màu theo grow rate |
| Pet fusion | `internal/application/pet/service.go` | `FusePets`, `SeniorFusePets` | `FusionRepository` | 1 chính + 4 phụ ở fusion thường |
| Pet fusion RPC | `internal/presentation/rtmp/handlers/pet/fusion.go` | `PetJoin`, `SeniorPetJoin` | pet service | Callback `onDelPet`, `onUpdatePet` |
| Pet DB transaction | `internal/infrastructure/persistence/postgres/pet_repository.go` | `ApplyFusion` | pgx transaction | Cache đổi sau commit |
| Pet cache | `internal/infrastructure/cache/cached_pet_repo.go` | `ApplyFusion` | `PlayerData` | Không để xóa cache từng phần |
| Pet cultivation | `internal/application/pet/item_consumption.go` | item-use wrappers | item repo, pet repo | Trừ stack và rollback Pet khi consume lỗi |
| Pet cultivation RPC | `handlers/pet/enhancement.go`, `fusion.go` | `PetStar`, `PetXsd`, `PetPzXsd`, `PetXd` | pet service | Response + inventory callback |
| Inventory core | `internal/application/item/service.go` | move/add/consume/sort | item repo, gamedata | SID trong `domain/item/item.go` |
| Item use | `internal/presentation/rtmp/handlers/item/use.go` | `UseItem` | effect registry | Không giả định mọi item là consumable |
| Transport scroll | `handlers/item/transport.go` | `UseTransport`, `UseTransportGroup` | scene, item, character | Timed scroll lưu expiry trong properties |
| Túi tiện dụng | `internal/application/item/tempbag_install.go` | `InstallTempBag` | item properties/gamedata | 20/50 ô, replace guard, expiry |
| Túi tiện dụng payload | `internal/application/item/tempbag_login.go` | `BuildTBagPayload` | character/item state | Dùng khi login/getTBag |
| HP/MP item | `internal/application/item/service.go` | item effects/recovery | character service | Xem thêm `docs/memory/recovery-targeting.md` |
| Shop service | `internal/application/shop/service.go` | `BuyItem`, `BuySystemItem` | character/item/gamedata | Point shop dùng `ExPoint` |
| Shop RPC | `handlers/shop/purchase.go` | `BuyItemClient`, `BuySystemItemClient` | shop service | Không phát `onAddChaQuest` |
| Shop tiện dụng | `handlers/shop/convenience.go` | `ClickShop`, `BuyBloodBag` | game-data, blood-bag service | Fire-and-forget Flash RPC |
| Túi cứu thương | `internal/application/bloodbag/service.go` | `Purchase`, `Load` | character + stat-feature repos | Gold + reserve commit chung ở PostgreSQL |
| Túi cứu thương DB | `internal/infrastructure/persistence/postgres/stat_feature_repository.go` | `CommitBloodBagPurchase` | wallet, feature state | Transaction, không migration |
| Quest service | `internal/application/quest/service.go` | lifecycle/objective sync | quest/item/pet repos | Catalog reconciliation |
| Quest notification | `handlers/quest/notify.go` | `PushQuestProgress`, `RefreshQuestProgressSilently` | quest service | `onAddChaQuest` có toast nhận quest |
| Quest 166 capture | `handlers/combat/commands.go` | battle update reward refresh | quest notify | Pet template 56, quality q=1 |
| Quest Admin | `dashboard/src/app/api/quests/` | API routes | Supabase auth/RPC | Flash publication vẫn blocked |
| PostgreSQL item | `internal/infrastructure/persistence/postgres/item_repository.go` | CRUD/move | player schema | Properties JSONB giữ metadata hạn |
| Player cache | `internal/infrastructure/cache/player_cache.go` | `persistPlayerData` | Redis/Postgres | Write-behind không mặc định là transaction đa entity |
