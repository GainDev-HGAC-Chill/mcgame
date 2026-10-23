# MCGame Fix History

## FIX-2026-10-23-07 — Launcher 6 mục, backup/restore DB và profile Online VPS

- `MCGAME.bat` nay tự build `bin/game.exe` rồi chỉ hiển thị sáu tác vụ: Start(build Compose), Stop(auto backup), Reset=Stop+Start, Flash 30 FPS, Restore DB và chọn Local/LAN/Radmin/Online VPS. `game restart` dùng cùng safe reset; Windows reset không còn wipe volume/database.
- Backup tạo custom-format full `pg_dump`, verify bằng `pg_restore --list`, lưu timestamp + SHA-256 ở `.local/backups/database`, rồi atomically cập nhật `supabase/backups/mcgame-live.dump`. Stop fail-closed và giữ server chạy nếu backup/publish lỗi.
- Theo lựa chọn rõ ràng của operator, snapshot canonical là full live DB gồm cả Auth/session/player data. File không bị `.gitignore`; local timestamp, `.env`, machine profile và partial file vẫn bị ignore. Snapshot hiện tại 3.519.158 byte, checksum PASS và archive list có 1.814 TOC entry. GitHub từ chối LFS object mới trên public fork, nên commit chưa push được amend để lưu snapshot bằng Git thường.
- Restore chỉ nhận `.dump` từ local backup hoặc tracked snapshot, nhận biết Git LFS pointer chưa pull, kiểm tra SHA-256 nếu có, yêu cầu xác nhận, tạo safety backup không ghi đè snapshot nguồn, dừng services, recreate DB, restore rồi chạy build/start. Restore thật chưa chạy để tránh mutate DB hiện tại.
- Profile Online nhận public IPv4/DNS, advertise host riêng và bind game edge `0.0.0.0`; dashboard giữ loopback. Dashboard LAN/Radmin bind/origin theo profile, bỏ hardcode `26.31.111.123`. Online chỉ là connectivity profile, chưa có TLS/firewall/security hardening.
- Test: `cmd/game` tests/vet/build PASS; affected gameplay tests/vet và Windows/Linux gameserver builds PASS; Compose config PASS với current `.env` và `.env.example`; Git attribute/ignore/checksum/archive checks và diff-check PASS. Không stop/reset/restore/restart, commit hoặc push.
- Trạng thái: IMPLEMENTED/FOCUSED_VERIFIED — cần push lại commit đã amend, rồi live-test restore trên database disposable hoặc khi đã chấp nhận downtime.

## FIX-2026-10-23-06 — Refresh chỉ số Tu Luyện/Mệnh Hồn và audit fresh install

- Root cause runtime: học/nâng Tu Luyện chỉ refresh danh sách skill; `moveSoul` chỉ refresh slot Mệnh Hồn. Cả hai không recompute và push final property sheet nên client chỉ thấy chỉ số mới sau khi đăng nhập lại.
- Sửa Tu Luyện: sau mutation thành công, phát `onUPP` từ character sheet có aggregate bonus và `onRefreshPetProp` cho toàn bộ Pet đã enrich. Sửa Mệnh Hồn: move type 2–4 reload đúng Pet qua Pet service sau persist rồi phát `onRefreshPetProp`; move type 1 character-only không reload Pet. Gameserver wiring truyền Pet service vào Soul handler.
- Test runtime: regression refresh mọi Pet sau Tu Luyện, refresh Pet sau Soul move và không refresh cho character-only move PASS. Focused combat/Soul/skill/Pet tests, targeted vet, Windows `bin/gameserver.exe`, Linux `bin/gameserver-linux` và diff-check PASS.
- Audit fresh install: 103 migration không trùng version; một version legacy `20261016` phải giữ để tương thích history. Shadow replay toàn chuỗi PASS; 101 seed không rỗng, khoảng 15.8 MB. Static audit 220 production Postgres function call phát hiện 6 function không có migration: năm Achievement và `player.consume_catch_ball_round`.
- Schema chưa portable: Achievement draft chưa apply/capture; hai box-admin function chỉ ở draft/live DB; live DB còn character-deletion objects và drift NPC/Quest/Find Back chưa được capture. Seed fresh có Soul `Vĩnh Sinh`/`Tăng Miễn Tử`, nhưng DB hiện hữu vẫn giữ text cũ. Dashboard compose còn bind cứng `26.31.111.123:3000`.
- Trạng thái: CODE_FIXED/FOCUSED_VERIFIED, FRESH_INSTALL_BLOCKED. Local/LAN có thể chạy trên máy hiện tại nhưng clone mới chưa được bảo đảm đầy đủ; public internet chưa sẵn sàng. Chưa apply/capture SQL, restart/deploy, Flash smoke-test, mutate dữ liệu, commit hoặc push.

## FIX-2026-10-23-05 — Tu Luyện, Shop quality, Mệnh Hồn và Văn Chỉ

- Unit A Tu Luyện: thêm provider stat chỉ hoạt động khi toàn bộ catalog khớp exact fingerprint 120 row/12 family. Mỗi family lấy level học cao nhất, các tier độc lập cộng dồn; curve thay thế và hệ số nghề/Pet nằm table-driven, chỉ cộng HP/Vật Phòng/Ma Phòng/Tốc Độ. Catalog, persisted level, class hoặc Pet family drift đều trả zero và log thay vì fallback.
- Tích hợp: character dùng additional bonus provider hiện có; Pet service đổi từ một provider sang danh sách additive nhưng giữ `SetFeatureBonusProvider` tương thích, sau đó đăng ký Training bằng `AddFeatureBonusProvider`. Bonus được nhập trước `RecalculateStats`, nên DTO và battle snapshot dùng cùng final sheet.
- Test Unit A: regression phủ highest-level thay thế row cũ, stacking giữa tier, hệ số character/Pet, catalog và persisted-level drift, hai Pet provider cộng dồn. Focused skill/Pet/item/domain-Pet tests, targeted vet, Windows/Linux gameserver builds và diff-check PASS.
- Unit B Shop quality: thêm item grant API phân biệt explicit color 0 với color bị bỏ trống. Marker persisted ngăn DTO/login fallback sang màu template; Trắng phát `colorCode=0,color=0,preNameType=0`. Toàn bộ NPC/System/VIP/Credit purchase dùng quality integer `0..25` và canonical mapping thay `math.Ceil`; invalid quality bị chặn trước debit. Regression exact slot 3439/template 1123 khóa hành vi Găng Thu Thập Trắng và giữ tên không prefix.
- Dashboard Unit B: NPC/System Shop thay ô số thô bằng lựa chọn 0..25 có nhãn màu, giải thích rõ 0 là Trắng/no-template-fallback; Zod và SQL draft NPC cùng giới hạn. Focused Go tests/vet/build, dashboard typecheck + 6 test PASS; Next compile/typecheck/static generation 28 trang PASS rồi fail standalone symlink `EPERM` Windows baseline.
- Unit C Mệnh Hồn: serialize toàn bộ EXP/dissolve/movement theo character owner và kiểm tra ownership trước mutation Pet. Fraction EXP, exact-next level-up/reset EXP, base+invested dissolve, quick color threshold/lock và temp-panel single/bulk đều khớp Flash/catalog; currency debit/credit có compensation khi feature-state save lỗi.
- Pet stat Unit C: Soul service là additive Pet provider; integral base props và decimal feature props giữ namespace riêng, phát canonical `soulProp<id>` và chỉ nhập `EffectiveStats` khi có mapping domain. Source JSON + seed row `751..760` đã normalize `Vĩnh Sinh`/`Tăng Miễn Tử`; SQL idempotent + inverse chỉ là draft, chưa apply/sinh migration.
- Test Unit C: regression phủ fraction, remaining-cost/next sid, level 10, base EXP dissolve, locked/color threshold, temp callbacks, concurrent EXP serialization, foreign Pet rejection, decimal sheet/EffectiveStats và 10 localization row. Focused Soul/handler/Pet/domain-Pet/item tests, targeted vet, Windows/Linux builds và diff-check PASS.
- Unit D Văn Chỉ: reserved box game-currency ID `242` được intercept trước character currency descriptor và chuyển qua `PPVEService.GrantKPReward`. Service khóa theo character, load/reset/increment overflow-safe rồi persist full PPVE state; kết quả mang `KPData`/`PPVEData` để item handler phát exact `onUpdatePPVEPanel(kpData,ppveData,nil,false)`, không đổi `pvePoint` ID 238 và không phát `onAddMoney`. Dashboard Box Items có option/whitelist `Văn Chỉ (KP Pet PVE)`.
- Test Unit D: regression phủ routing ngoài character currency, PPVE save failure không consume box, giữ nguyên full-state fields, hai grant đồng thời không lost update và callback panel không kèm callback tiền. Tổng focused Soul/Soul-handler/Pet/domain-Pet/item/item-handler/skill/shop tests và targeted vet PASS; Windows/Linux gameserver builds PASS; dashboard typecheck, 11 Node tests và Prettier PASS. Next compile/typecheck/static generation đủ 28 trang rồi chỉ fail standalone trace-copy do symlink `EPERM` Windows baseline; diff-check PASS ngoài cảnh báo LF→CRLF.
- Trạng thái: Units A–D IMPLEMENTED/FOCUSED_VERIFIED. SQL selectable gift-bag và localization Soul vẫn chỉ là draft, chưa apply/sinh migration. Chưa restart/deploy, Flash smoke-test, sửa ActionScript, mutate catalog/player data, commit hoặc push.

## FIX-2026-10-23-04 — Khôi phục Docker build sau commit character damage

- Root cause: commit HEAD `bfdcc50 update fix dmg` đổi generic skill adapter từ heuristic `expSkill/dexSkill` sang `resolveCharacterSkillMultiplier(description)` nhưng không thêm hàm được gọi, đồng thời thêm `strings` hai lần và để `regexp`/`strconv` chưa dùng. Vì vậy `Start server` build dashboard/edge xong nhưng dừng ở compile `mcgame-server`.
- Sửa: hoàn thiện parser phần trăm damage hẹp cho cụm gây/tạo `% sát thương` hoặc `% tấn công`, hỗ trợ dấu thập phân chấm/phẩy. Câu chỉ tăng buff không bị hiểu là damage; mô tả có nhiều hệ số damage khác nhau fail-closed về multiplier mặc định `1.0` thay vì tùy tiện chọn một nhánh điều kiện. Helper duplicate xuất hiện đồng thời trong file được hợp nhất vào một implementation.
- Test: application/domain combat và combat handler PASS; targeted vet PASS; Windows `bin/gameserver.exe`, Linux `CGO_ENABLED=0 bin/gameserver-linux` và `docker compose -f docker/docker-compose.yml build mcgame-server` PASS.
- Trạng thái: FIXED/VERIFIED — image đã build nhưng assistant không tự chạy lại Start/restart, không mutate DB/player/catalog, commit hoặc push.

## FIX-2026-10-23-03 — Đồng nhất Dashboard giữa localhost và VPN IP

- Hiện tượng: hai origin `http://localhost:3000` và `http://26.31.111.123:3000` hiển thị/cho sửa các nhóm Quest và Shop/Túi quà không đồng nhất. Source không có chủ đích phân vai theo hostname; compose thiết kế một dashboard bind cả hai địa chỉ, nhưng Quest dùng guard riêng, allow-list chỉ nêu VPN IP và shell dashboard có thể được giữ theo cache riêng của từng origin/build.
- Sửa quyền: Quest Admin gọi chung `requireSameOrigin` với Shop hệ thống/Túi quà tự chọn, nhưng vẫn ánh xạ lỗi sang contract `QuestAuthError`. Regression khóa cùng quyền write-origin cho `localhost`, `127.0.0.1` và `26.31.111.123`; cross-site/missing/invalid origin vẫn bị chặn.
- Sửa deployment/UI: `DASHBOARD_ALLOWED_ORIGINS` khai báo đủ ba origin được phục vụ. Response `/` và `/dashboard/**` dùng `Cache-Control: private, no-store`, tránh hai hostname giữ hai shell navigation từ build khác nhau.
- Test: 26 Quest/editor/validation/box/origin test PASS, 1 live test SKIP; dashboard typecheck PASS; `docker compose config --quiet` PASS. Next compile/lint/typecheck/static-generation đủ 28 trang rồi chỉ fail standalone trace-copy do Windows symlink `EPERM` baseline. Dashboard container đang dừng nên chưa rebuild/restart hoặc HTTP smoke-test; không mutate DB/catalog/player data, commit hoặc push.
- Trạng thái: CODE_FIXED/FOCUSED_VERIFIED — cần rebuild/start dashboard được cho phép, sau đó hard refresh hai origin và xác nhận cả hai cùng sửa được Quest, Shop hệ thống và Túi quà tự chọn.

## FIX-2026-10-23-02 — Túi Nguyên liệu/Bảo thạch và Tự đặt NL cấp 6 follow-up

- Root cause panel: `EquipFuncBag.getAllShopList()` giữ index shop nhúng trong Flash. Guard cũ bỏ row server lỗi khỏi callback nhưng không xóa row đã có; `showSaleItems()` gọi `_core.getTemplateData(...).type` trước khi kiểm tra `st`, nên cả tab Nguyên liệu và Bảo thạch vẫn có thể lỗi #1009.
- Sửa catalog override: mọi `TBL_SHOP_SLOT` thiếu template được gửi dưới cùng ID nhưng `sid=0,st=5`. `CallBackGlobal.parseAndSetShopSlot()` vì vậy xóa row khỏi index `sid` system shop cũ rồi chỉ đưa tombstone vào index 0; chỉ đặt `st=5` là không đủ. Source cache không bị mutate.
- Root cause Tự đặt: inventory login dùng raw `Item.ToDTO()` và alias cấp 6 trước đây chỉ nhận persisted `ItemTypeMaterial`. Row legacy lưu `ItemTypeConsumable` dù template `kind=6/14` vẫn phát `color=5`, bị điều kiện Flash `color < 5` loại.
- Sửa DTO: shared `BuildClientItemDTO` nhận diện nguyên liệu bằng template `kind=6/14`, giữ canonical `colorCode=5` nhưng phát `color=4` cho cấp 6 bất kể persisted item type. Login dùng shared DTO; các callback item handler đã dùng cùng builder. Cấp 6 vẫn là max và không được ghép lên cấp 7.
- Test: focused application-item, item handler, shop handler và auth handler tests + targeted vet PASS; diff-check sạch ngoài LF→CRLF. Windows/Linux whole-server build vẫn BLOCKED bởi compile ngoài phạm vi ở `internal/application/combat/skill_adapter.go`. Không restart/deploy, sửa ActionScript, mutate DB/catalog/player data, commit hoặc push.
- Trạng thái: CODE_FIXED/FOCUSED_VERIFIED — cần restart được cho phép và kết nối Flash mới để `getShopConfig` áp tombstone, rồi smoke-test hai tab và Tự đặt.

## FIX-2026-10-23-01 — Đồng bộ Kết Tinh Thần Tu, Chìa Ma Tâm và Shop Kết Tinh

- Kết Tinh Thần Tu: generic `UseMultiItem` luôn trả actual `used_item`, kể cả award-backed item. Handler vì vậy refresh đúng DTO khi còn stack và dùng SID thật khi stack hết; award currency 241 tiếp tục phát `onAddMoney(...,"spirituality",delta,total)` ngay sau persist character. Regression phủ dùng 2/3 rồi dùng hết item 2336 và tổng 400.000 mỗi viên.
- Chìa Ma Tâm: exact fingerprint chỉ nhận `type=550,useType=1,name` bắt đầu `Ma Tâm (` và `proplNum` phải trỏ tới `TBL_CREATUREH_HEART` type 1..7. Grant được gộp theo lần multi-use, chỉ persist sau khi inventory consumption thành công, dùng heart template ID làm bag entry ID, phân loại `ren/shou/zhi/ji/mo/lon/te`, rồi push toàn bộ `updateMonsterHeartPanel`. Catalog regression xác nhận đủ 395 key và không có reference hỏng.
- Shop Kết Tinh: bỏ mock `exchangeItem`. Chỉ xử lý authority hiện có `shopId=1,type=19,cType1=32,cType2=0,limit=0`; ánh xạ score 32 sang persisted `CurrencyWisdomCrystal`/client key typo `wisdonCrystal`, debit và persist trước grant, compensation nếu grant thất bại. Equipment được tạo theo template/quality/bind thật; popup `onAddItem` có đủ tên/màu/type/kind và mọi slot đổi được phát `onAddCharactorSlot`.
- Test: focused item/monsterheart/shop/RTMP/gamedata suites PASS và diff-check không có whitespace error ngoài cảnh báo LF→CRLF. Windows/Linux gameserver builds đã PASS trước correction consume-before-grant cuối; rerun cuối của `cmd/gameserver` và targeted vet bị BLOCKED bởi thay đổi đồng thời ngoài phạm vi ở `internal/application/combat/skill_adapter.go` (import trùng/không dùng và thiếu `resolveCharacterSkillMultiplier`). Không restart/deploy, không dùng vật phẩm hoặc người chơi thật, không mutate DB/catalog/player data, không sửa ActionScript, commit hoặc push.
- Trạng thái: CODE_FIXED/FOCUSED_VERIFIED — final whole-server verify chờ compile baseline combat được hoàn tất; sau đó cần restart được cho phép và Flash smoke-test để xác nhận ba panel thực tế.

## FIX-2026-10-22-03 — Năm nhóm gameplay/progression

- Quest Thần Khí: thêm exact fingerprint cho `4707..4712`, bắt buộc đúng class và trả tại NPC 272; sáu reward main magic weapon `1348..1353` tiếp tục đi qua reward pipeline type 19. Catalog/dialogue/class drift fail-closed.
- Tu Luyện: đăng ký RPC `addTrainingSkill`, chỉ học level đầu `useEnv=3,kind=5`, nâng exact-next trong cùng `codeName` qua `skillLearnByClient`; giữ gate level/class-rank, EXP, bạc đã chọn, mutation lock và rollback. Catalog regression khóa 120 row = 12 family × 10 level, dùng chung sáu class. Không tự tạo công thức stat khi buff catalog bằng 0.
- Mệnh Hồn Pet: soul service dùng Pet repository để reconcile/persist tám slot cơ bản cho Pet cấp 80+, giữ legacy state không giảm; mua slot 9–16 kiểm tra owner+level trước khi tiêu Soul EXP và giữ giá client `20000..55000`.
- Huy Chương Mạo Hiểm: state đổi thành `level|progress`; panel trả đúng ví `explorerPnt` và progress. Nâng thường chuyển tối đa điểm ví cần thiết; RPC `levelUpEMedalByGold` hỗ trợ 50 Gold/50 progress và 500 Gold/500 progress, carry nhiều cấp, lock và compensation. Không thêm bạo kích thiếu authority.
- TXKC: mùa cố định `2026-10-22 00:00`–`2026-12-31 00:00` end-exclusive theo `Asia/Ho_Chi_Minh`; state version theo season trong `special_training_course`, reset ngày 00:00/tuần thứ Hai, premium 680 Gold, mua 500 EXP/300 Gold, max 70, task claim 120/280 EXP và reward free/premium 70 cấp từ Flash. Shop thiếu slot catalog bị chặn; `spIndex=-1` nên không phát bonus đặc biệt thiếu authority. Task engine có API idempotent theo state nhưng các business-event chưa có exact hook không được tự cộng giả.
- Test: focused quest/skill/soul/Pet/Explorer Medal/TXKC/handler/cmd suites, targeted vet, Windows và Linux `CGO_ENABLED=0` builds PASS; reward catalog 140 level-entry đều trỏ item tồn tại; diff-check không có whitespace error ngoài cảnh báo LF→CRLF. `go test ./internal/...` còn baseline ngoài phạm vi: battle test goroutine log sau test, activity SendCombine hết event window và hai chat stat expectation. Không restart/deploy, mutate DB/player data, sửa ActionScript, commit hoặc push.
- Trạng thái: CODE_FIXED/FOCUSED_VERIFIED — cần restart để runtime nạp RPC mới; Flash smoke-test và event-hook TXKC authoritative còn chờ.

## FIX-2026-10-22-02 — Cải tạo và túi Chế tạo

- Đổi chất lượng: pending preview giữ toàn bộ properties; confirm dùng một item update để lưu đúng chỉ số cơ bản/phụ, thuộc tính khóa, quality/prefix, element, star và bind đã hiển thị. Soul 0 dùng `activePropType/activePropNum` từ equipment template, còn soul hợp lệ hiện hữu được giữ.
- Đổi linh hồn một phần: roll thấp hơn vẫn trả payload so sánh type 2 nhưng không mutate trang bị/callback stat; lựa chọn “Không áp dụng” vì vậy không còn áp chỉ số thấp. Roll bằng hoặc cao hơn tiếp tục auto-apply theo flow hiện tại.
- Túi Nguyên liệu/Bảo thạch: `getShopConfig` bỏ và log row type 12/19/29 không có template trước khi Flash nhận catalog, ngăn `EquipFuncBag.showSaleItems()` dereference null. Catalog checked-in 2.542 slot đã đối chiếu không có reference lỗi; guard xử lý drift runtime.
- Nguyên liệu cấp 6: DTO dùng alias hẹp `color=4,colorCode=5`, khớp điều kiện `autoInputMake color < 5`; màu cấp dưới không đổi nên tỷ lệ hiếm cấp 5→6 vẫn đúng. Server vẫn resolve level 6. Đây là bậc tối đa nên mix one/all fail trước tiêu hao thay vì clamp 6→6; không tạo cấp 7.
- Test: full item handler/shop/application-item/domain-item suites, targeted vet, Windows và Linux `CGO_ENABLED=0` gameserver builds PASS; diff-check không có whitespace error ngoài cảnh báo LF→CRLF. Chưa restart/deploy, Flash smoke-test, mutate catalog/player data, commit hoặc push.

## FIX-2026-10-22-01 — Dashboard và runtime túi quà tự chọn

- Runtime: `data_tbl_item_award.payload.selection={order,getNum}` là authority cho túi tự chọn khi và chỉ khi toàn bộ row hợp lệ, liên tục và cùng số lượt chọn. Route này chạy trước weighted picker, chỉ nhận Pet/trang bị/vật phẩm, không consume lúc mở panel và chặn multi-use. Grant dùng chung capacity/quality/prefix/material pipeline; pending snapshot khóa theo từng nhân vật, revalidate instance/template/stack/index, chặn duplicate/concurrent selection và không phát callback thành công nếu consume cuối thất bại.
- Dashboard/API: Box Items có mode thường/tự chọn, sắp thứ tự, số lượt chọn, type restriction và cảnh báo reload. Save là full snapshot atomic với strict 128 KiB same-origin POST, row ownership, MD5 revision, stale conflict và hỗ trợ delete-all ở mode thường; endpoint DELETE từng row đã bỏ. Metadata malformed/mixed/fractional fail closed.
- Database: draft `supabase/data/selectable_gift_bag_admin.sql` dùng global allocation lock, giữ ID row thuộc đúng package, full replacement trong transaction và chỉ cấp execute cho `service_role`. Docker Linux engine không chạy và Supabase CLI không có trên PATH, nên function chưa apply/rollback/concurrency/privilege smoke và chưa được `db diff` sinh migration. Không seed/chuyển hardcode item 3376 hoặc mutate catalog/player data.
- Test: focused Go gamedata/application item/item handler PASS, gồm order/malformed/mixed/unsupported, no-consume, wrong/stale/duplicate/concurrent selection, capacity failure, consume-failure callback và grant→consume callback order; targeted vet và Windows gameserver build PASS. Dashboard typecheck, 5 focused config/same-origin tests và Next compile/type/static generation 28 trang PASS; production build vẫn chỉ fail standalone symlink `EPERM` Windows baseline. `-race` không chạy được vì Windows CGO tắt; all-dashboard raw Node runner còn baseline alias `@/lib` ở `npc-editor.test.mjs`.
- Trạng thái: CODE_FIXED/SQL_DRAFTED — chưa end-to-end vì schema compose chưa apply, chưa restart/deploy hoặc Flash smoke-test, không commit/push.

## FIX-2026-10-21-07 — Học kỹ năng Pet Siêu Cấp

- Contract: đối chiếu bài chính thức `4000.html`, Flash và catalog. Giữ RPC `petBook(petId, bookItemInstanceId)`/responder `f,rf,n`; 11 sách exact `3216..3226`. Ghi đè lưu ý legacy “lấy cấp thấp nhất” theo yêu cầu operator: phải có đủ Sơ/Trung/Cao cùng family ở cấp 3 và kết quả luôn Super cấp 1.
- Sửa gameplay: exact matrix 33 prerequisite level-3 → 11 Super level-1; không cần ô trống. Slot prerequisite có index nhỏ nhất được thay bằng Super, hai slot còn lại clear `0`, unrelated/locked slot giữ nguyên. Pet đã có bất kỳ level Super nào không thể học lại Sơ/Trung/Cao cùng family. Super book kiểm tra `qLevel>=3` và item `reqClass` bằng creature `classId`, không dùng `classIds` hay family segment `creKind` làm class gate.
- Sửa atomicity: PostgreSQL khóa/revalidate Pet ownership + exact 15-slot snapshot và book ownership/stack, sau đó update Pet và decrement/delete sách trong cùng transaction. Cache chỉ cập nhật Pet và book sau commit. Handler phát inventory callback → ba `onUpdatePet` tăng dần → một property refresh.
- Test: catalog invariant khóa đủ 11 item, 33 prerequisite và 33 Super chain; regression no-empty-slot, deterministic slot, missing/wrong-level/duplicate/already-Super, lower-tier lockout, quality/classId, atomic failure và cache-after-commit PASS. Focused application Pet/PostgreSQL/cache/Pet handler, full affected domain/application Pet+combat và Pet/combat handler, targeted vet, Windows/Linux gameserver builds và repository-wide diff-check đều PASS.
- Trạng thái: IMPLEMENTED/VERIFIED — chưa restart/deploy/live-test, không dùng Pet/sách thật, không mutate schema/seed/player data, không commit hoặc push.

## FIX-2026-10-21-06 — Sách Pet sai class và Cận Vệ Thánh Nữ không hoạt động

- Root cause sách: resolver tên hoạt động đúng, nhưng eligibility hiểu sai `creKind=family|minimumQLevel` thành class gate. Live log/image mới ghi nhận bốn lần `petBook` trên Pet 278; DB read-only cho thấy Thỏ Baby này là `classIds=10,qLevel=3` nhưng đã sở hữu Trí Tuệ Thiên Nhiên family `1`, nên segment đầu không thể khóa class.
- Fix: petBook bỏ family/class gate và chỉ kiểm tra minimum qLevel ở segment hai. Mocked end-to-end learn trên Pet `classIds=10,qLevel=3` xác nhận Báo Chi Tấn Tiệp 1382→2741, Cổ Long Chi Hồn 1377→2820, Cổ Long Trung Cấp 1389→3352, Ngạnh Xác Hóa 1378→2696, Ngạnh Xác Trung Cấp 1390→3355, Vẻ Ngoài Yếu Ớt 1379→2841 và Thụ Bì Ngoại Sáo 1374→2825.
- Audit toàn catalog: 199 item `type=506,useType=2`; 154 exact-name row resolve duy nhất và pass quality eligibility; zero ambiguity/missing direct ID; 45 container/random/test/special/localized rows giữ fail-closed. Chi tiết tại `docs/research/2026-10-21_02_PET_SKILL_BOOK_CATALOG_AUDIT.md`.
- Cận Vệ Thánh Nữ: ownership 3848/3849/3850 nay tạo lại shield trước mỗi lượt, pool 50/75/100% MaxMP, ba hit, buff 2015/2014/2013 và phát `buffAdd` để Flash hiện icon. Chọn semantics mỗi lượt theo seed, không dùng forum “ba lượt đầu” đang mâu thuẫn.
- Test: full affected domain Pet/combat, application Pet/combat và Pet/combat handler PASS; targeted vet, Windows/Linux gameserver builds và diff-check PASS. Không dùng sách/Pet/item thật hoặc mutate DB; runtime/DB chỉ đọc. Correction cuối chưa được rebuild/restart vào container đang chạy.
- Trạng thái: CODE_FIXED — chờ deploy và Flash smoke-test nếu người dùng cho phép.

## FIX-2026-10-21-05 — Mở rộng skill Pet theo audit KingGame

- Research: đối chiếu bài KingGame 2012 với Flash, `TBL_SKILL`, `TBL_BUFF` và runtime. Flash xác nhận chỉ gửi một target `tid`; server phải tự xử lý AoE, multi-hit, proc, status và passive. Forum chỉ bổ sung bằng chứng, không tự động ghi đè seed hay quyết định đã duyệt.
- Active skill: thêm exact plan cho 28 template/10 family đang rơi về legacy: Cao Tần Long Hống Trung Cấp, Đột Thích Xung Phong, Kích Trạng Loạn Tâm, Lạc Thạch Vẫn Tinh, Mê Tâm, Phản Ứng Trì Độn, Táng Thần Quần Thể, Trào Phúng Quần Thể, Uy Nhiếp Ý Chí và Thạch Hóa Công Kích level 1. Mọi target/coefficient/chance/duration/buff/rest dùng exact seed đã corroborate.
- Runtime: planned Stone dùng `StatusStone` + buff 299, chặn action/skill, chịu generic debuff resistance và giải/icon lifecycle chuẩn. Da Kháng Tính 3808/3812/3813 cleanse một harmful removable trước tick với 50/75/100%. Cân Bằng Chi Thược 3763/3771/3772 tăng MagicAttack theo finalEnergy × 0.05/0.10/0.15%, không compound khi refresh. HoT Tự Nhiên Thuẫn Đơn Thể sửa 65/65/65 thành 25/45/65%, giữ defense/target seed.
- Fail-closed: không đoán Cận Vệ Thánh Nữ, Cung Chúc Tân Hỷ, Phản Xạ, Sủng Vật Hiệp Hộ, Thích Vũ, Tiếng Sét Phẫn Nộ hoặc level không có seed. NĐTT cấp 1 giữ seed 120% theo plan người dùng duyệt.
- Test: exact-plan/stone/cleanse/Energy-idempotence/HoT regressions PASS; full affected combat/Pet/handler tests, targeted vet, Windows/Linux build PASS; diff-check sạch ngoài cảnh báo LF→CRLF. Chưa restart/deploy/live-test, mutate dữ liệu, commit hoặc push.
- Trạng thái: CODE_FIXED — chờ deploy và Flash smoke-test riêng nếu người dùng cho phép.

## FIX-2026-10-21-04 — Nhịp Đập Trái Tim luôn gây 1 damage

- Root cause: exact plan 5363/5362/5361 khai báo coefficient chung 1.0, nhưng special chain executor còn bỏ qua plan và gọi trực tiếp `applyDamage(target, 1, true)`, nên mỗi pulse luôn chỉ trừ đúng một HP.
- Sửa: exact coefficient theo mô tả là 1.20/1.50/1.80 MagicAttack; mỗi pulse dùng `applyPlannedDamage`, vốn đã xử lý defense/critical/shield/playback/reflection. Xóa phần reflection cũ để không phản đòn hai lần.
- Follow-up semantics ban đầu đã bị sửa theo xác nhận người dùng: mỗi proc phát lại cùng NĐTT; nếu target còn sống thì tiếp tục target đó, nếu target chết thì chọn enemy sống khác. Chuỗi chỉ dừng khi roll Combo thất bại, hết enemy hoặc chạm cap cấp 1/2/3 là 5/10/15 cast. Mỗi pulse tăng thêm 0.20 MagicAttack coefficient và damage thành công được giữ tăng nghiêm ngặt; separator ngăn Flash gộp các pulse retarget thành AoE. `Thuốc Kích Thích` đổi từ owner character sang `TargetSelf`, nên HP cost và buff đều áp trên Pet caster.
- Correction sau runtime: log 11:52–11:53 chỉ có một cast NĐTT mỗi lượt. Plan cũ roll `30%` cố định nên không dùng thuộc tính Combo và buff Thuốc Kích Thích không thể ảnh hưởng. NĐTT nay lấy effective Pet Combo theo điểm phần trăm (`Combo/100`), gồm buff battle-local; MP chỉ trả một lần. Cap trung gian 11 cast đã được contract cuối thay bằng 5/10/15 theo cấp.
- Test: regression exact plan, runtime damage >1, Combo 0/100, stimulant +30%, repeated same-skill/same-target chain, distinct Flash playback và MP một lần PASS; full domain/application combat + combat handler PASS; targeted vet, Windows build và Linux `CGO_ENABLED=0` build PASS.
- Live evidence 12:09–12:12 trước correction cuối: Pet 278 có `finalCombo=79`; mọi NĐTT thử nghiệm one-shot target rồi phát `bid=10120`. Logic dừng-on-kill lúc đó trái contract cuối cùng và nay đã được thay bằng roll rồi retarget enemy sống.
- Trạng thái: CODE_FIXED — regression retarget, tăng damage, cap, playback và one-MP-cost PASS; bản correction cuối chưa restart/deploy/live-test.

## FIX-2026-10-21-03 — Ảo hóa, học skill và xác nhận dung hợp Pet

- Ảo hóa: runtime `petEat(278,364,1906,5,2,...)` đã xóa Pet phụ và cộng Xảo nhưng handler trả `{success,pet}`, trong khi Flash chỉ đọc `succ/costPet/costItem/propName/value/msg`; item ID 1906 còn bị parse thành bool nên đá không được validate/consume. Sửa exact responder/callback, parse item instance, rate theo code Flash, tiêu Pet phụ + đá ở cả success/failure và ledger thay riêng bonus ảo hóa mà không xóa bonus khác. Pet dưới tím hoặc Pet chính chưa khóa nay bị chặn trước tiêu hao với thông báo panel. Final review thay compensation không đầy đủ bằng transaction khóa/cập nhật Pet chính, xóa Pet phụ và trừ/xóa đá atomically; cache chỉ đồng bộ sau commit.
- Học kỹ năng: sách 1622 `Thuốc Kích Thích` trả `f=2` vì family 3590/3672/3673 dùng `useEnv=2`; resolver cũ chỉ nhận 0/1. Resolver nay nhận 0..2, vẫn loại special `-1`, giữ ambiguity/class/quality/conflict/open-slot gates.
- Dung hợp: runtime 05:04 xác nhận success đã đúng dữ liệu: Pet chính 364 giữ ID và từ 1.50 lên 1.95 (tím), bốn Pet phụ 365/366/332/154 bị xóa. Không đổi implementation; thêm regression exact lam→tím. Failure regression tiếp tục xác nhận xóa bốn phụ và giữ nguyên Pet chính.
- Test: full domain/application Pet, Pet handler, cache và PostgreSQL persistence PASS; regression atomic-error giữ nguyên cả hai Pet + stack đá; targeted vet PASS; `go build -o bin/gameserver.exe ./cmd/gameserver` PASS; diff-check sạch ngoài cảnh báo LF→CRLF.
- Trạng thái: CODE_FIXED — không phục hồi Pet/vật phẩm đã bị bản runtime cũ tiêu, không mutate dữ liệu thêm và chưa restart/deploy hoặc Flash smoke-test.

## FIX-2026-10-21-02 — Tẩy điểm Pet báo Invalid input

- Hiện tượng: Flash gửi đúng `petXd(300,1901)` với Quy Nguyên Đơn template 1526 nhưng server trả `[Connection] Invalid input` và không tiêu item.
- Root cause: compose DB lưu Pet 300 với `attr_points=0`, `distributed_attr_points=440`, `max_attr_points=430` và tổng năm `Apt*Ex=440`, đủ điều kiện legacy reset an toàn. Ba function đọc trả `SETOF player.character_pets` theo thứ tự cột vật lý `attr_points, distributed_attr_points, max_attr_points, life`, nhưng Go scan thành attr/max/distributed/life nên runtime thấy distributed=430 và reject vì không khớp 440.
- Sửa: dùng chung `scanPetRow` cho read by ID, following Pet và list Pet, map đúng attr/distributed/max/life. Thêm regression mapping DB 0/440/430/9800 và regression service đúng shape Pet 300; reset hoàn 440 điểm, xóa năm `Apt*Ex`, giữ max 430 và chỉ tiêu một item khi save thành công.
- Test: full PostgreSQL persistence, domain/application Pet và Pet RTMP handler PASS; targeted vet PASS; `go build -o bin/gameserver.exe ./cmd/gameserver` PASS; diff-check không có whitespace error ngoài cảnh báo LF→CRLF.
- Trạng thái: CODE_FIXED — chỉ query DB read-only, không dùng/tiêu item 1901, không thay đổi Pet/player data và chưa restart/deploy hoặc Flash smoke-test.

## FIX-2026-10-21-01 — Dọn backup và hợp nhất migration STH pending

- Hiện tượng: repository theo dõi 48 bản sao `.bak`; ignore rule `.bak` không bắt các tên thực tế. Chuỗi STH bị chia thành bốn migration pending, trong đó version ngắn `20261016_sth_events.sql` chạy trước migration tạo `character_sth_progress` nhưng đã `ALTER` bảng đó, làm fresh replay sai thứ tự.
- Sửa cleanup: xóa 48 backup tracked, toàn bộ cây backup local đã có canonical seed/migration/audit before-image thay thế và artifact relocation smoke; thêm ignore cho các hậu tố backup/temp phổ biến. Không xóa research artifact, dependency/cache hoặc source hiện hành.
- Sửa migration: hợp nhất implementation STH vào `20261016090000_sth_runtime.sql`, giữ schema, backfill table type, default reward/event config và upgrade từ trạng thái partial. Sáu NPC-creature STH được đưa vào seed canonical; migration chỉ chèn động khi catalog đã tồn tại để không va ID trong fresh migration-before-seed. Sau khi compose DB hoạt động lại, history xác nhận `20261016`, `20261016133000`, `20261020070000` đã được apply; ba filename được giữ dưới dạng compatibility no-op để không phá upgrade history, tuyệt đối không repair/revert history.
- Quest/NPC Admin: SQL draft cho lần apply kế tiếp trỏ rollback source tới chính `before_image` trong audit table thay vì file backup local đã xóa; migration đã apply được giữ bất biến.
- Test: 103 local version không trùng; 102 version chuẩn 14-digit và một legacy `20261016` phải giữ để khớp history. Consolidated STH có transaction/dollar quote cân bằng, đủ 6 bảng; seed NPC có 1.954 row, không trùng ID và đủ 6 row STH. Compose `migration list` khớp local/remote toàn bộ; `db push --include-all --dry-run` trả up-to-date, zero migration/seed pending. Focused STH handler PASS, dashboard typecheck và gameserver build PASS; diff-check không có whitespace error. Full activity còn baseline `TestBuySendCombine_SuccessUpdatesBuyCountAndGold` (`got 0, want 1`) lặp lại khi chạy riêng, không liên quan file cleanup/STH.
- Trạng thái: CODE_CLEANED/HISTORY_COMPATIBLE — không apply migration, repair history, restart/deploy gameplay service hoặc thay đổi DB/player data.

## FIX-2026-10-19-01 — Dashboard quản lý NPC, nhiệm vụ liên kết và shop

- Hiện tượng: dashboard chỉ có Quest Admin, không có nơi quản lý hội thoại ambient, quan hệ NPC↔quest hoặc shop chuẩn; DB shop thay đổi cũng không được `getShopConfig` phát cho Flash.
- Sửa dashboard/DB: thêm list/detail NPC ba tab, reverse join quest read-only, editor `name/on_service_text/shop_id` và slot type 19/29; mutation có same-origin, validation, revision, idempotency, shared-shop confirmation và audit transaction. Slot xóa bằng `st=5`; không tạo container, không sửa vị trí slot cũ hoặc special `TBL_CREDIT` shop.
- Sửa runtime: scene phát `onServiceText`; `getShopConfig` serialize toàn bộ slot cache theo field camelCase và thứ tự ID để client cập nhật index `sid/st` khi kết nối.
- Database: backup catalog local trước schema; apply RPC/audit RLS trên compose; rollback verification PASS; migration `20261018232116_npc_admin_dashboard.sql` được `db diff` sinh từ DB tạm và fresh replay PASS. Không thay đổi player row.
- Test: focused Go scene/shop/gamedata PASS, targeted vet và gameserver build PASS; dashboard typecheck + 5 validation/editor tests PASS; production compile/type/static generation PASS nhưng standalone copy vẫn fail Windows symlink `EPERM` đã biết.
- Trạng thái: CODE_FIXED/SCHEMA_APPLIED_LOCAL — chưa restart/deploy gameserver/dashboard hoặc Flash smoke-test.
- Hardening tiếp theo: thêm lookup shop/item/equipment; reject boolean query không hợp lệ, empty/no-op và fractional slot ID; giữ mọi legacy `st` ngoài 0/5 read-only. Code/typecheck/test PASS; function hardening chờ re-apply vì Docker engine bị tắt sau lần apply schema đầu.
- UI shop tiếp theo: thay form dày bằng lưới ô hàng có ảnh, tên đúng catalog trong game, badge vị trí/trạng thái và giá bán; editor chi tiết chỉ mở cho ô được chọn. Dashboard hydrate metadata chỉ đọc từ catalog và không đưa `displayName/iconCode` vào mutation. Typecheck + 6 focused tests PASS; production compile/static generation PASS trước lỗi symlink Windows baseline.
- Follow-up vị trí NPC: bảng tìm kiếm thêm cột tọa độ, map hiển thị ID + tên catalog; tab NPC cho chọn map và sửa X/Y. Save whitelist/validate exact `pos_map_id/pos_x/pos_y`, yêu cầu map tồn tại, tọa độ integer signed 32-bit và giữ revision/request UUID/audit/no-op; legacy placement không chạm được giữ nguyên. Dashboard typecheck + 7 focused tests + production compile/type/static generation PASS; SQL draft/rollback test đã cập nhật nhưng chưa apply vì Docker Linux engine tắt, standalone copy vẫn fail symlink `EPERM` Windows baseline.

## FIX-2026-10-19-02 — Nền tảng Thành Tựu end-to-end và claim an toàn

- Hiện tượng: snapshot đánh key requirement bằng achievement ID; schema không lưu được 33 achievement nhiều điều kiện; không có evaluator production; reward columns của toàn bộ catalog trống; claim đánh dấu đã nhận trước khi cấp, nuốt lỗi và có race nhận lặp. Farm còn phát sai requirement 294 từ bộ đếm RAM.
- Sửa persistence/code: thêm SQL draft lưu từng requirement, event key idempotent, suy ra completion khi đủ mọi điều kiện và claim reservation/finalize/release. Login reconcile cấp độ/ngày đăng nhập; quest hoàn thành và battle thắng dùng business ID chống cộng lặp; snapshot/callback dùng đúng requirement ID và contract ba tham số.
- Sửa reward: registry ID canonical cho 29/30 reward trong hướng dẫn chính thức, hỗ trợ item tiêu hao, title và Ấn Chương; validate catalog lúc startup. `Giáo Sư Biết Tuốt` fail closed vì không có exact title ID. Grant lỗi release reservation và không claim; finalize lỗi giữ reservation để chặn cấp lặp.
- Sửa farm: xóa callback requirement 294 sai ngữ nghĩa và bộ đếm volatile; chưa tự suy đoán evaluator “bị trộm cây”.
- Test: focused achievement/auth/quest/combat/NPC/utils/cmd PASS; targeted vet PASS; `go build -o bin/gameserver.exe ./cmd/gameserver` PASS. Broad `go test ./internal/...` chỉ còn Quest conversation registry `235 want 227` và hai chat-stat baseline ngoài phạm vi.
- Trạng thái: CODE_FIXED/SQL_DRAFTED — Docker Linux engine tắt nên `supabase/data/achievement_system.sql` chưa apply, chưa sinh migration/fresh replay hoặc chạy DB concurrency test; chưa restart/deploy/smoke-test và không thay đổi player data.

## FIX-2026-10-18-20 — Audit và sửa pipeline thuộc tính Pet

- Hiện tượng: Bạch Hổ ID 168 hiển thị tư chất lực khoảng 70k nhưng công vật lý khoảng 29k; một số base prop và passive Pet không có hiệu lực.
- Nguyên nhân dữ liệu: Pet lưu `grow_rate=15.000` từ award-quality encoded 15 từng bị truyền như raw 15.0; catalog đúng là 1.5. Local DB có 14 row legacy cùng dấu hiệu 15.000/19.000. Pet 168 thực tế có `aptStrengthEx=0`; `finalStrength=129` là 50 template + 79 cấp, không phải 100 điểm đã phân phối.
- Nguyên nhân code: recalculate đọc aptitude từ template thay vì cột persisted; bỏ direct creature props; 12 passive Kích Thích Tiềm Năng 3744–3755 có công thức trong mô tả nhưng `prop1..6=0`; normalization `/10` tuyệt đối lại làm raw quality 1.5 thành 0.15.
- Sửa: dùng aptitude persisted; seed hit/dodge/speed/critical/combo/counter/defy/reborn/resiDefy từ template; áp exact quality-scaled formulas cho 3744–3755; quality `>=10` mới chia 10, raw decimal giữ nguyên. Không đổi shared coefficients và không sửa dữ liệu người chơi.
- Test: full domain/application Pet, item, combat và Pet/item/combat handlers PASS; targeted vet, gameserver build và scoped diff-check PASS. Broad internal chỉ còn Quest registry `235 want 227` và hai chat-stat baseline ngoài phạm vi.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test; 14 row legacy chưa được remediation.

## FIX-2026-10-18-19 — Nâng Man Lực Pet và skill nhân vật lỗi RPC

- Hiện tượng: `petUpSkill(168,4)` trả `Not found`; `skillLearnByClient("2216", null)` và `("2226", null)` trả `invalid arguments`.
- Nguyên nhân Pet: resolver sách 2242 chọn ngẫu nhiên giữa các skill cùng tên/cấp, đã ghi special `60903` (`useEnv=-1`, không có level kế) thay vì chuỗi Pet `4172→4171→4170`. Da Kháng Tính có collision tương tự `3808` với chuỗi Pet bắt đầu ở `3812`.
- Sửa Pet: name resolver chỉ nhận skill dùng trong môi trường 0/1, fail-closed khi level thấp nhất vẫn mơ hồ; exact compatibility source sửa các slot cũ `60903→4172` và `3808→3812` trong lần nâng hợp lệ tiếp theo, rồi vẫn dùng cost/manual/rollback hiện có.
- Nguyên nhân và sửa skill nhân vật: Flash gửi skill ID dạng numeric string và position `null`; handler nay nhận string số bên cạnh AMF number, vẫn từ chối chuỗi không hợp lệ.
- Test: full Pet, character-skill và combat handler PASS; targeted vet, gameserver build và scoped diff-check PASS. Không restart/deploy hoặc trực tiếp sửa player/Pet data.

## FIX-2026-10-18-18 — Hổ Khiếu Sơn Lâm bị loại trước combat nên không AOE

- Hiện tượng: runtime nhận lệnh Pet dùng 7026/7028 nhưng log `Rejected skill command: pet does not know skill`; skill không vào effect engine nên trông như mất AOE. DTO cùng phiên vẫn có skill trong `skill1..skill15`.
- Nguyên nhân: `OwnedSkillLevel` đọc legacy `property.skills`, trong khi learn/upgrade/DTO và persistence hiện dùng các slot canonical `skill1..skill15`.
- Sửa: authority combat đọc `LearnedSkillSlots()` và lấy level thật từ `TBL_SKILL`; vẫn kiểm tra exact ID nên skill chưa sở hữu tiếp tục bị từ chối.
- Test: focused ownership regression, full Pet/combat application và domain combat, combat handler, targeted vet, gameserver build và diff-check PASS. Chưa restart/deploy hoặc mutate Pet/player data.

## FIX-2026-10-17-17 — Bí kíp Mật Bảo, item Cây Ước Nguyện và buff bánh

- Hiện tượng: bí kíp chế tạo Mật Bảo trả `life skill recipe not found`; item claim từ Kho Ước Nguyện đã vào DB nhưng chỉ hiện sau `bagSort`; bánh type 502 bị tiêu hao nhưng không có buff.
- Nguyên nhân: 66 sách Mật Bảo type 516 bị route nhầm vào `TBL_SKILL`; lotto claim chỉ refresh holding bag; food handler chỉ đọc `i1/i2` trong khi bánh stat dùng `TBL_BUFF` và có `i1=i2=0`.
- Sửa: map exact sách `5042..5107` tới Mystery recipe `1..66`, persist `learnedRec`, duplicate no-consume và rollback khi consume fail; trả item thực từ lotto claim rồi push DTO `onAddCharactorSlot`; thêm 33 mapping food→buff timed, stat refresh và chặn zero-effect chưa hỗ trợ trước tiêu hao.
- Test: full Mystery Treasure/lotto/lotto-handler/item-handler PASS; focused food PASS; targeted vet và build `bin/gameserver.exe` PASS. Full item chỉ còn sáu baseline award-quality/starter-gift đã biết.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test; không dùng item/người chơi thật và không thay đổi DB/player data.

## FIX-2026-10-19-03 — Tick damage skill Pet hiển thị sai vị trí/số HP

- Hiện tượng: periodic damage của `Độc Thích Đơn Thể`, `Độc Thích Quần Thể` và `Tàn Nhẫn Tư Liệt` đã trừ HP server nhưng playback dùng `ActionTypeAuto`, bỏ trống `TargetPosition` nên mặc định `tid=0`, đồng thời đặt `hHp` trong `tSObj`. Flash vì vậy lấy HP nền của slot 0 để tính popup hoặc hiện số trên sai mục tiêu; lethal tick không có Die action.
- Sửa: gom cả hai turn-processing path qua một helper, gửi tick dương thành Hurt tại đúng `sid/tid`, đặt HP mới trong `sSObj`, rồi phát Die hoặc Idle. Tick bị shield/vô địch hấp thụ hoàn toàn không tạo popup giả.
- Test: domain/application combat và combat handler PASS; regression xác nhận tick 300 tại slot 7 gửi `sid=tid=7`, `bid=5`, `sSObj.hHp=700`, không có `tSObj.hHp`; lethal tick phát Hurt → Die. Targeted vet và build `bin/gameserver.exe` PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-19-04 — Bách Biến Trùng Kích chỉ gây 1 damage

- Hiện tượng: active skill của Đại Thánh Chí Tôn, gồm ID gốc `6712` mô tả x1,4 sát thương, chỉ gây 1 damage.
- Nguyên nhân: chuỗi creature/evolution-only `6712..6720` không thuộc roster suy từ sách Pet và chưa được thêm vào ba special family trước đó. Adapter vì thế dùng fallback `expSkill=1/dexSkill=-1`, tương đương base damage 1 thay vì hệ số công vật lý.
- Sửa: exact-ID registry cho cả ba nhánh: `1.4/1.7/2.0`, `1.6/1.9/2.5`, `1.9/2.2/3.0`; adapter ưu tiên template ID trước family name để tránh tên catalog có khoảng trắng và ký hiệu `∮`/`ㄨ` không đồng nhất. Chưa đoán random Choáng/Độc/Suy Nhược của `buffId=-2`.
- Test: regression đủ chín ID và adapter ID 6712 PASS; full application/domain combat và combat handler PASS; targeted vet, gameserver build và scoped diff-check PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-19-05 — Vô Song Loạn Vũ không truy kích sau khi kết liễu

- Hiện tượng: Đại Thánh Chí Tôn sở hữu `Vô Song Loạn Vũ` cấp 3 nhưng sau khi `Bách Biến Trùng Kích` hạ mục tiêu không đánh tiếp ba mục tiêu ngẫu nhiên.
- Nguyên nhân: battle participant không snapshot canonical `skill1..skill15`; combat plan cũng không có on-kill hook nên passive `6721/6722/6723` không thể được phát hiện hoặc kích hoạt.
- Sửa: participant Pet mang skill ownership battle-local; Bách Biến exact plan có generic on-kill follow-up. Passive cấp 1/2/3 chọn tối đa 1/2/3 mục tiêu sống ngẫu nhiên khác nhau, dùng exact `Trùng Kích Truy Kích` `6724..6750` và multiplier x2,5..x4 theo nhánh/cấp Bách Biến, không trừ MP lần hai. Chỉ chính Bách Biến kết liễu mới kích hoạt.
- An toàn: mục tiêu đã chết bị loại; thiếu passive, hit không kết liễu hoặc thiếu exact follow-up template đều fail closed. Bốn debuff ngẫu nhiên của `buffId=-2` chưa được đoán.
- Test: snapshot skill slot, exact chín mapping follow-up, level-3 truy kích đủ ba mục tiêu distinct, non-lethal/no-passive rejection; full application/domain combat và combat handler PASS; targeted vet, gameserver build và diff-check PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.
- Bổ sung theo xác nhận người dùng: mỗi wave Trùng Kích có ít nhất một mục tiêu chết sẽ kích hoạt wave kế tiếp; dừng khi wave không kết liễu hoặc hết mục tiêu. Playback có separator nội bộ để Flash phát từng wave riêng. Regression năm enemy xác nhận hai wave cấp 3 lần lượt xử lý 3 rồi 2 mục tiêu và phát hai cast.
- Correction tiếp theo: bỏ damage effect tổng hợp khỏi passive; đăng ký exact active plan cho đủ `6724..6750` và buộc cả wave đầu lẫn wave dây chuyền đọc plan đó. Regression năm hit xác nhận tất cả action mang exact skill ID/name và sát thương x2,5 vượt rõ damage đánh thường.

## FIX-2026-10-19-06 — Di Hình Hoán Vị không giải debuff hoặc tăng kháng

- Hiện tượng: Pet có `Di Hình Hoán Vị` nhưng đầu lượt không có xác suất giải trạng thái và lượng kháng mô tả không tham gia combat.
- Nguyên nhân: passive `6751/6752/6753` chưa có turn-start hook; status modifier cũng chưa có đường đưa prop 58 `Debuff Kháng` vào xác suất áp hiệu ứng xấu.
- Sửa: dùng helper exact `6754/6755/6756` làm authority cấp 1/2/3: roll 20/30/40%, khi thành công xóa toàn bộ harmful status removable trước tick/control và thêm 80/120/150 `PropResistDebuff` trong đúng một lượt. Harmful planned status dùng công thức base chance + debuff hit - debuff resistance; base-guaranteed cũng có thể bị kháng.
- An toàn: effect không removable được giữ; buff có lợi không bị xóa; roll thất bại loại kháng cũ để không tồn tại quá hạn; nếu có nhiều cấp thì chỉ cấp cao nhất có hiệu lực.
- Test: đủ ba exact config, deterministic success cleanse periodic debuff trước damage, giữ locked/beneficial effect, effective resistance 150 và xác suất 40%→30% khi attacker hit 50; deterministic failure xóa resistance cũ. Full domain/application combat và combat handler, targeted vet, gameserver build và diff-check PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-20-01 — Ba skill Huyền Minh Linh Vũ sai target/damage và không hiện buff

- Hiện tượng: `U Minh Huyền Vũ` chỉ đánh một mục tiêu, chưa cộng MaxHP Pet và thiếu `Thất Hồn`; `Thợ Săn Sát Thủ`/`Tứ Phương Khiếp Đảm` không có logo xác nhận và bị cảm nhận là không tác dụng.
- Nguyên nhân: chuỗi exact `7002..7010` chưa có audited plan nên rơi vào fallback; TSST cộng Accuracy phẳng thay vì phần trăm; planned status không mang buff ID ra wire. Bộ gom AoE còn xóa `tSObj` của status-only segment, nên dù modifier battle-local tồn tại, Flash không nhận `buffAdd`.
- Sửa U Minh: exact chín plan target selected enemy rồi mở rộng hai hàng ngang tới cap 4/6/8; damage `25% Attack + 15..30% caster MaxHP` đi qua cùng defense/crit/element/position/modifier pipeline. `Thất Hồn` buff 14 có base chance đúng từng row 50..70%, sau đó cộng debuff/stun hit và trừ generic/stun resistance.
- Sửa buff/debuff: TSST dùng Speed và Accuracy +10/15/20% trong 5 lượt với IDs `80118..80120`; TPKĐ giữ giảm hai Attack + Speed 5/8/12% trong 1 lượt với IDs `80121..80123`. Status phát `buffAdd`; expiry và dispel phát `buffDel`/`buffColDel`; AoE có target-state round riêng nên icon không bị mất hoặc treo.
- Test: đủ chín exact mapping U Minh, adapter ID 7008, target tám enemy hai hàng, legacy và per-mille composite damage, 60% Thất Hồn trừ 100 generic + 200 stun resistance còn 30%; TSST effective Speed/Hit 110/110 và icon 80118; TPKĐ effective stats 95/95/95, icon 80121 và expiry remove. Full domain/application combat + combat handler, targeted vet, gameserver build và diff-check PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-16-28 — Không thể chủ động bỏ nhiệm vụ type 1 đã nhận

- Hiện tượng: cả Flash QuestManager và QuestPanel tự khóa nút Hủy khi active DTO có `data.type == 1`; nếu gọi RPC trực tiếp, service cũng từ chối toàn bộ type 1 bằng `ErrQuestNotAvailable`.
- Sửa: bỏ type-1 rejection, serialize abandon bằng mutation lock và vẫn yêu cầu đúng character + progress active trước khi xóa. Active type-1 DTO dùng presentation type 2 để mở nút Hủy; canonical catalog type 1 không đổi nên completion policy, prerequisite và reward vẫn giữ nguyên.
- Tradeoff client: nhiệm vụ type 1 đã nhận tạm nằm trong nhóm nhiệm vụ chính; danh sách nhiệm vụ chưa nhận vẫn giữ nhóm gốc. Không sửa ActionScript, không tạo history/phần thưởng và không thay đổi dữ liệu người chơi.
- Test: focused abandon/presentation regressions PASS; quest/NPC handler PASS; targeted vet, gameserver build và scoped diff-check PASS. Full application quest chỉ còn baseline conversation registry `234 want 227` ngoài phạm vi.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-16-27 — Pet AoE trừ HP nhưng không hiện damage hoặc chỉ tác dụng một mục tiêu

- Hiện tượng: Pet damage plan thay đổi HP thật trên server nhưng không sinh Hurt/Idle actions. AoE playback xóa `tSObj.hHp` khỏi opener để chuyển sang shared Hurt round, nhưng round đó rỗng; quái có thể chết mà không hiện số. Ngoài ra target cap được áp trước khi lọc slot đã chết, nên một slot chết gần có thể làm mất target sống xa hơn.
- Sửa: mọi direct/fixed Pet hit dương sinh chuỗi hit → Hurt → Die/Idle; mỗi target đã nhận damage có `hHp` riêng trước death. Area resolver lọc dead target trước rồi mới cap số lượng.
- Semantics giữ nguyên: miss không gây/hiện damage; pure buff/debuff và lần cast đầu của DoT-only không giả damage; DoT hiển thị khi tick; AoE theo hình học chỉ đánh các ô hợp lệ đang có mục tiêu sống.
- Test: full domain/application combat và combat handler PASS; catalog invariant xác nhận mọi direct-damage plan có hệ số và target dương; targeted vet, build `bin/gameserver.exe` và diff-check PASS.
- Trạng thái: CODE_FIXED — chưa restart/deploy hoặc Flash smoke-test.

## FIX-2026-10-16-26 — Launcher chặn migration out-of-order và thiếu mục Flash 30 FPS

- Hiện tượng: Start dừng tại `supabase db push` vì migration `20261016040000_findback_bounty_verification.sql` có timestamp trước migration cuối đã ghi trong history; thông báo wrapper còn nhầm thành có thể thiếu CLI. Menu chính không có lựa chọn 30 FPS dù command/client riêng đã tồn tại.
- Sửa: launcher truyền `--include-all`, vẫn chỉ `--include-seed` cho database mới và đổi lỗi wrapper thành thông báo trung thực. `MCGAME.bat` thêm phím `F` chạy `game client30`.
- An toàn: không reset/volume delete/reseed database hiện hữu. Dry-run xác nhận đúng hai migration pending `20261016040000` và `20261016090000`, không có seed.
- Test: `go test ./cmd/game`, `go vet ./cmd/game`, build `bin/game.exe`, asset 30 FPS và scoped diff-check PASS.
- Trạng thái: CODE_FIXED/BINARY_BUILT — chưa chạy Start thật, chưa apply migration hoặc restart services.

## FIX-2026-10-16-25 — Sáu class không hoàn thành được Thăng Cấp Trung Cấp 3-1

- Hiện tượng: quest `143/287/639/640/643/644` không có requirement row và không thuộc completion policy đã duyệt, nên luôn giữ state 103/`CompletionBlocked` dù nhân vật đã là Sơ Cấp.
- Sửa: thêm exact fingerprint policy riêng cho sáu quest; completion authority kiểm tra đúng class, `class_rank == 1` và vị trí gần Khảo Quan Trung Cấp NPC 600. Persisted objective cũ không được phép bypass điều kiện.
- Phạm vi: không tự sửa tiến độ, không tăng class rank tại bước 3-1, không suy đoán logic cho bước 3-2/3-3.
- Test: focused policy/runtime regression đủ sáu class PASS; quest/NPC handler PASS; targeted vet và build `bin/gameserver.exe` PASS. Full quest/questaudit vẫn fail baseline ngoài phạm vi vì conversation registry hiện có 234 entry trong working tree nhưng assertion cũ chờ 227.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-16-24 — Quest Admin cho sửa nhiệm vụ đang được nhận

- Hiện tượng: dashboard chỉ cho sửa tên/mô tả/hội thoại và khóa mục tiêu, NPC, thưởng, requirements/prerequisites khi có `character_quests` active; DB RPC cũng trả `ACTIVE_QUEST_IN_USE`.
- Sửa: trang quest không còn khóa structural fields hoặc nút lưu theo active count; cảnh báo vẫn hiển thị rõ thay đổi có thể ảnh hưởng đối chiếu runtime. Migration thay `public.quest_admin_save` để bỏ active-use rejection, giữ revision conflict, validation, transactional before-image/audit, child ownership và cycle detection. Guard của shared loop không đổi.
- An toàn: không tự bỏ/reset/bồi thường tiến độ người chơi; save vẫn yêu cầu reason, request UUID và exact revision.
- Test: dashboard typecheck PASS; editor/validation/permission 20 PASS, 1 live test SKIP; Prettier scoped và diff-check PASS.
- Trạng thái: CODE_FIXED/MIGRATION_STAGED — chưa apply local vì `supabase-db` đang stopped; chưa deploy dashboard.

## FIX-2026-10-16-23 — Boss bản đồ và thưởng Sổ Tay Ma Thú

- Hiện tượng: boss bản đồ click trực tiếp không bắt đầu trận; 13 boss Kênh 1 không xuất hiện; boss map có thể dựng sai quái/level; toàn bộ boss map và Sổ Tay không có loot riêng.
- Nguyên nhân: NPC boss hiện dùng type 10 nên Flash gửi `clickNpc`, nhưng NPC handler đăng ký sau đã ghi đè combat route. Static row của 13 mythic boss dùng sentinel map 200. Battle cũ tra `cid == npcID` thay vì roster `TBL_NPC_CREATURE.nid`, đồng thời mất schedule NID khi enemy giữ creature CID. `data_tbl_boss_loot` trống hoàn toàn và `daily_only` normal còn bị bootstrap thành schedule state.
- Sửa: route `clickNpc`/`hitNpc` vào shared schedule-boss path; validate map/channel/alive fail-closed; dựng đủ roster catalog và gắn canonical `BossCtx`; inject mythic boss theo registry/channel scope; loại `daily_only` khỏi scheduler. Thêm generator/seed thưởng theo ground/flying/mythic và sáu dải level.
- Dữ liệu local: đã apply 862 loot row vào compose DB; phủ 150/150 registry boss, 3–7 row/boss, 0 item/equipment reference lỗi. Không thay đổi player data.
- Test: focused scheduleboss/combat/scene/NPC/generator PASS; targeted vet và build gameserver/generator PASS. Broad internal không có lỗi boss; còn baseline ngoài phạm vi ở item award quality, Quest registry, chat stat và một flaky center test đã PASS khi chạy riêng.
- Trạng thái: CODE_FIXED/LOCAL_DATA_APPLIED — chưa restart/deploy hoặc Flash smoke-test.

### Follow-up channel 0

- Runtime 07:22:18–07:22:20: `clickNpc(707)` trả `[Connection] Not found`; character 3 đang map 2, connection channel 0 và DB có `(nid=707, channel_id=0, is_alive=true)`.
- Nguyên nhân: validation mới dùng `channelID <= 0`, trong khi gateway/runtime hiện coi channel 0 là hợp lệ.
- Sửa: scope `all` chấp nhận channel 0; channel âm vẫn bị chặn; `channel:1` vẫn chỉ nhận 1. Bổ sung regression cho engagement và scene visibility channel 0.
- Test: focused scheduleboss/combat/scene/NPC handlers, targeted vet và gameserver build PASS. Chưa restart/retest Flash.

## FIX-2026-10-15-22 — Sách triệu hồi, Mật Đạo và vật phẩm xã hội

- Hiện tượng: Sách Triệu Hồi bị tiêu hao nhưng không có Pet đi sau; Sách Dịch Chuyển Mật Đạo bị tiêu hao nhưng không đổi map; Hoa Hồng/Chocolate bị tiêu hao nhưng không tăng thân mật; người dùng nhấp đôi Trứng Gà nhưng không thấy tác dụng.
- Nguyên nhân: type-500 chưa biết exact summon/secret-passage IDs nên chỉ trả message generic. Flash chỉ cho render follow Pet khi có long-buff 1709/1979. `useIntimacyItem` cũ chỉ consume stack. Trứng Gà dùng RPC mục tiêu `delPopNum`, không phải generic `useItem`; callback stack của `addPopNum`/`delPopNum` còn có thể trừ quantity hai lần trên cùng object đã mutate.
- Sửa: 2514/2543 cấp buff 1709 trong 1 giờ/30 ngày và chỉ gọi Pet cam đã khóa; 2798 cấp buff 1979 trong 30 ngày và nhận Pet tím/cam đã khóa. Persist follow, gửi `onCancelPet`→`onCreatePet`; không có Pet hợp lệ thì không consume. 128/2137 preflight không group/battle rồi consume và vào map 140106. 1302/1303 tăng 60, 1315/32 tăng 660 thân mật; Hoa Hồng yêu cầu khác giới, chỉ áp dụng với bạn bè, cập nhật hai row chiều nếu tồn tại, refresh stack và phát lại `onInitViewImC`. Nhấp đôi 724/783 không consume và chỉ dẫn dùng nút mục tiêu; wire `addPopNum`/`delPopNum` vẫn là effect thật.
- Test: exact summon buff/duration/quality, bound-quality eligibility, exact intimacy values/gender flags, bidirectional persistence/non-friend rejection, secret-passage/social target registry, item/social/handler/Pet packages, targeted vet và gameserver build PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test; không migration và không sửa dữ liệu người chơi.

## FIX-2026-10-15-21 — Thuốc Tăng Trưởng Pet không cộng EXP

- Hiện tượng: dùng Thuốc Tăng Trưởng lên Pet không thay đổi kinh nghiệm/cấp Pet.
- Nguyên nhân: cả tám template là `type=501`, `useType=2` nhưng server route vào HP/MP potion; RTMP handler chỉ truyền `PetID` cho type507. Lượng EXP không nằm ở `i1/i2`, mà ở exact Flash registry `PET_GROW_ITEM_NUM`.
- Sửa: exact-ID growth effect cho 961/962/964/1585/1586/1587/1836/3339 với 30k/150k/300k/300k/2m/6.4m/12.5m/100m EXP; validate Pet target/ownership/Pet level/max level, persist qua `pet.Service.GainExperience`, hỗ trợ single/multi count và phát `exp`, changed `level`, `onRefreshPetProp`.
- Test: exact registry, quantity, Pet-vs-character level gate, wrong target, requested handler flow/item consumption và progress callbacks PASS; item/Pet packages, targeted vet và gameserver build PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test; không thay đổi dữ liệu Pet/người chơi thật.

## FIX-2026-10-15-20 — Item sự kiện ẩn và túi tiện dụng route nhầm

- Hiện tượng: item từ Shop/sự kiện đôi khi chỉ hiện sau khi sắp xếp túi; kéo vật phẩm vào túi tiện dụng báo `Internal error: cannot equip item`.
- Nguyên nhân: ba activity reward callback gửi raw Go `*Item` thay vì Flash slot DTO. `addtItemFromBag`/`addmxFromBag` bị đăng ký vào generic `MoveItem`, khiến target position 1..22 bị parse thành equipment SID.
- Sửa: chuẩn hóa activity notice + `BuildClientItemDTO`; giữ shop changed-stack snapshot hiện có. Thêm handler riêng decode page 24 ô, map normal index 1-based/MX 101-based, validate capacity/type, xóa icon nguồn và refresh `updateTempBags`; temp sort giữ cùng index contract. Follow-up sửa `tempList` từ inventory DTO sang compact `{t,n,b,c}` để item đã chuyển render ngay thay vì trông như bị mất. Hỗ trợ alias `tempToBag`/`mxtempToBag` và đúng wire RPC `addItemFromTemp(position,targetSID[,count])`/case-sensitive `addMXFromTemp`; full/partial stack dùng SID đích, không swap item không tương thích vào temp bag.
- Test: exact normal/MX compact payload, actual runtime args `[1,2106]`, move-out normal/MX, requested destination/no-swap, item application và item handler PASS; targeted vet và gameserver build PASS. Broad internal trước đó có lỗi registry Quest 234-vs-227 từ thay đổi quest ngoài phạm vi và 2 chat-stat baseline; Pet suite hiện PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-15-19 — Đồng bộ tỷ lệ và hệ số cường hóa Pet

- Hiện tượng: server luôn nâng thành công tới 12 sao dù Flash hiển thị tỷ lệ giảm dần; tổng hệ số đã lưu đúng, ví dụ `1.56 + 0.40 = 1.96`, nhưng bảng Pet vẫn có thể hiện `1.56`.
- Nguyên nhân: application không roll xác suất; responder chỉ cập nhật `growRateAdd` ở root Pet trong khi `PetPanel`/`PetManagerPanel` tính từ nested `property.growRateAdd`.
- Sửa: dùng exact base rates Flash `100/100/90/80/70/60/50/30/30/30/30/20%` cho sao đích 1→12. Failure giữ sao dưới 8; từ 8+ về 0 nếu không có Nội Đơn Chúc Phúc hợp lệ, có protection thì giữ sao và consume item. `petStar`/`petStarClear` đồng thời phát `onRefreshPetProp({id,s})` từ DTO mới nhất.
- Điều chỉnh theo xác nhận operator 2026-10-15: failure từ 8+ không protection đặt cả sao và `GrowRateAdd` về 0 rồi tính lại stats; dưới 8 giữ cả hai, có protection cũng giữ cả hai. `petStarClear` thủ công không đổi trong điều chỉnh này.
- Test: exact rate table, deterministic success/failure, mốc 8 có/không protection và growth reset/preserve, response `flag/starSafeNum`, regression `1.56 + 0.40 = 1.96`, Pet handler/domain, targeted vet và gameserver build PASS. Broad internal giữ nguyên baseline 4 Pet + 2 chat-stat.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-14-18 — Hoàn tất registry nhiệm vụ đối thoại

- Hiện tượng: completion policy chỉ cho phép 67 quest type 1 theo ID, nên 160 quest đối thoại type 2/4 đã audit vẫn luôn state 103 và không thể turn-in.
- Sửa: thay allowlist ID-only bằng registry 227 fingerprint exact `ID/name/type/startNPC/finishNPC`; chỉ cho phép khi không có ordinary requirement, `rt/rn=0`, dialogue đầy đủ và finish NPC dương. Quest 46 được giữ nguyên; catalog drift và nhóm mơ hồ tiếp tục fail closed.
- Audit: `cmd/questaudit` kiểm tra type counts, fingerprint/NPC/dialogue drift, requirement/currency overlap và exact policy partition. Checked catalogs báo 227 (67/67/93), zero diagnostics; partition 3682 catalog, 227 conversation, 90 special, 4 missing-target, 215 currency, 1312 unconfigured.
- Test: focused quest/handler/NPC/audit và targeted vet PASS; audit registry test chỉ dùng catalog checked-in, không phụ thuộc `.local`; broad internal chỉ còn 4 baseline Pet + 2 chat-stat; `questaudit` và `gameserver` build PASS.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test; không migration và không thay đổi player data.

## FIX-2026-10-14-16 — Phiếu lễ và Thẻ Kim Phiếu

- Hiện tượng: `Ngân Phiếu Lễ` 654, `Hoàng Kim Lễ` 1064 và `Thẻ Kim Phiếu` 3848 đi vào generic type-550 và không đổi được tiền theo mô tả.
- Sửa: application path riêng cho đúng ba template; 654 cộng 10.000 `moneyBind`, 1064 cộng 50 `goldBind`, 3848 chọn ngẫu nhiên 88/188/288/588/888/1888 `goldBind`; hỗ trợ dùng đơn/nhiều và callback `onAddMoney`.
- An toàn: validate template/level/target/count/overflow; lỗi persist ví sau consume dừng trước success callback và không tự replay.
- Test: focused application và toàn bộ item handler PASS; targeted vet/build PASS.
- Trạng thái: CODE_FIXED — chưa deploy hoặc Flash smoke-test.

## FIX-2026-10-14-17 — Quest Admin không cần đăng nhập

- Hiện tượng: giao diện chỉnh sửa nhiệm vụ phụ thuộc Supabase Auth/admin session.
- Sửa: Quest API cấp editor identity nội bộ cố định, không gọi Auth; PATCH vẫn bắt buộc exact same-origin. Validation/revision/active-use/audit giữ nguyên.
- Test: quest editor/validation/permission tests và dashboard typecheck PASS; production build compile/typecheck xong nhưng copy standalone bị Windows symlink `EPERM`.
- Trạng thái: CODE_FIXED — chưa deploy/smoke-test dashboard.

## FIX-2026-10-13-01 — Shop điểm thưởng

- Hiện tượng: WebPanel hiện có điểm nhưng System Shop báo `not enough points`.
- Nguyên nhân: Shop kiểm tra/trừ `Character.Honor`, trong khi WebPanel/currency registry dùng `ExPoint` (`currency_type=223`).
- Sửa: `internal/application/shop/service.go`, `handlers/shop/config.go`; dùng `ExPoint`, callback key `exPoint`, khóa purchase theo character.
- Test: shop package PASS; test concurrent chỉ một purchase thành công; build/vet PASS. `-race` chưa chạy vì môi trường tắt CGO.
- Deploy: rebuild/restart gameserver; không migration.
- Trạng thái: PARTIAL — chưa có transaction DB chung cho ví và inventory, chưa live-test Flash.

## FIX-2026-10-13-02 — Dung hợp Pet tiêu hao sai

- Hiện tượng: gửi 1 chính + 4 phụ nhưng server chỉ xử lý một Pet phụ; UI chỉ đúng sau relog.
- Nguyên nhân: handler chỉ đọc hai args; service xóa từng Pet và handler không phát callback đầy đủ.
- Sửa: `application/pet/service.go`, `domain/pet/repository.go`, cached/postgres pet repository, `handlers/pet/fusion.go`.
- Cách sửa: validate đủ/không trùng/đúng owner; transaction update chính + delete phụ; callbacks `onDelPet`/`onUpdatePet`.
- Test: success/failure/replay focused tests, cache/postgres tests, build/vet PASS.
- Deploy: rebuild/restart; không migration.
- Trạng thái: CODE_FIXED — chưa kiểm chứng trực tiếp trong Flash.

## FIX-2026-10-13-03 — Item luyện Pet không cập nhật

- Nguyên nhân: nhiều RPC bỏ qua item instance hoặc chỉ đổi Pet, không trừ stack/không trả field mà responder Flash đọc.
- Sửa: `application/pet/item_consumption.go`, `service.go`, `handlers/pet/enhancement.go`, `fusion.go`, `helpers.go`.
- Cách sửa: validate owner/bag/stack, tiêu hao một lần, rollback Pet khi consume lỗi, trả remaining stack và callback inventory.
- Test: XSD consume/rollback tests, handler package, build/vet PASS.
- Trạng thái: PARTIAL — cần live-test từng tab và vật phẩm bảo hộ tăng sao.

## FIX-2026-10-13-04 — Tiêu Dao Phù thời hạn bị mất

- Nguyên nhân: transport dùng helper luôn giảm stack, không phân biệt `TBL_ITEM_TEMPLATE.t > 0`.
- Sửa: `application/item/timed_reusable.go`, `handlers/item/transport.go`.
- Cách sửa: kích hoạt expiry một lần trong item properties; tái sử dụng không gia hạn/không tiêu hao; item `t <= 0` vẫn tiêu hao.
- Test: timed reuse/no-extension và permanent consume PASS; item handler/build/vet PASS.
- Trạng thái: CODE_FIXED — chưa live-test; disconnect mới được điều tra trong phiên hiện tại.

## FIX-2026-10-13-05 — Túi tiện dụng 20/50 ô

- Nguyên nhân: `SetTempBag` là handler chặn chủ đích; payload không lưu bag instance/capacity/expiry.
- Sửa: `application/item/tempbag_install.go`, `tempbag_login.go`, `handlers/item/tempbag_install.go`, sort filter.
- Cách sửa: lưu bag đã lắp bằng item slot nội bộ, capacity theo `proplNum`, expiry theo `t`, block thay túi nếu làm ẩn contents.
- Test: install/replace/expiry/login payload/content-safety/sort PASS; build/vet PASS.
- Trạng thái: PARTIAL — chưa xác minh RPC tháo túi và chưa live-test Flash.

## FIX-2026-10-13-06 — Shop phát lại thông báo nhận quest

- Nguyên nhân: mọi purchase gọi full `PushQuestProgress`; callback `onAddChaQuest` luôn hiện toast nhận quest trong `QuestPanel.as`.
- Sửa: `handlers/shop/`, `handlers/quest/notify.go`.
- Cách sửa: refresh objective/NPC state im lặng, không phát `onAddChaQuest`.
- Test: silent refresh không phát callback add-quest; handler/build/vet PASS.
- Trạng thái: CODE_FIXED — chưa live-test.

## FIX-2026-10-12-QUEST166 — Bắt Nấm Lục

- Quest 166 yêu cầu pet template 56, kind 3/type 12/q 1/num 1.
- Pet của character 6 đã tồn tại và không được chỉnh sửa/tiêu hao khi điều tra.
- Nguyên nhân hiển thị 0/1: capture lưu Pet nhưng thiếu quest refresh callback.
- Sửa: `handlers/combat/commands.go`; test reward-count và focused combat/quest PASS.
- Trạng thái: CODE_FIXED — binary host đã build, container vẫn chạy bản cũ; cần relog và capture mới sau khi được phép restart.

## FIX-2026-10-12-QUESTADMIN — Quest Admin

- Dashboard/API/schema/audit/revision/active-use guard đã triển khai; auth tests, typecheck, dashboard build và focused Go tests đã PASS ở checkpoint trước.
- Flash catalog activation vẫn `database-only`; không có publication/rebuild an toàn.
- Hai migration local đã áp dụng nhưng checkpoint ghi migration history chưa đăng ký.
- Trạng thái: PARTIAL/BLOCKED cho Flash activation; không replay DDL hoặc sửa dữ liệu người chơi.

## FIX-2026-10-13-07 — Nút mua găng thu thập báo method not found

- Client gọi fire-and-forget `clickShop(275)`; 275 là NPC Tiệm Tạp Hóa Ông Mộ, có `shop_id=32`, không phải shop/slot ID.
- Nguyên nhân: server thiếu registration/handler. Shop 32 đã có găng 1123–1126; luồng mua chuẩn xử lý giá, tiền và inventory.
- Sửa: `handlers/shop/convenience.go`, `handler.go`, wiring `cmd/gameserver/main.go`; tra NPC game-data rồi push `onOpenShop(32)`. Không phát item miễn phí.
- Test: route 275→32, unknown NPC, equipment position 13 callback; shop/item handlers, vet và build PASS.
- Trạng thái: CODE_FIXED — cần Flash smoke-test mở shop, mua và trang bị/thu thập.

## FIX-2026-10-13-08 — Túi cứu thương HP/MP báo method not found

- Client gọi `buyBloodBag({bt,bn})`; loại 1..4, cap 10,000,000, giá `ceil(bn/5000)` Gold, callback `setBloodBag("|...|")`.
- Nguyên nhân: server thiếu toàn bộ RPC/state/login payload; `ep` luôn là zero.
- Sửa: `application/bloodbag/`, shop convenience handler, `FeatureBloodBag`, stat-feature login `ep`, PostgreSQL transaction Gold + feature state.
- Test: bốn state mapping, giá boundary, cap, thiếu Gold, rollback, concurrent replay, atomic repository path và login/stat-feature packages PASS.
- Migration: không; dùng `player.character_wallet` và `player.character_stat_features` hiện có.
- Trạng thái: CODE_FIXED — chưa live-test mua đầy/một phần và tự hồi phục sau battle.

## FIX-2026-10-13-09 — Tiêu Dao Phù 7/30 ngày gây disconnect

- Log xác nhận client đóng kết nối ngay sau `useTransport`; không có panic/server crash. Bản deploy cũ đi qua consume path cho item thời hạn.
- Sửa code hiện tại giữ item `T>0`, lưu expiry một lần, không gia hạn khi reuse, không tái kích hoạt item hết hạn; ưu tiên pass thời hạn trước item dùng một lần cùng nhóm.
- Test: 7 ngày, 30 ngày, single-use, priority, expiry/no-reactivation và item handler PASS.
- Trạng thái: CODE_FIXED — nguyên nhân client exception cụ thể UNKNOWN; cần smoke-test Flash sau deploy để xác nhận không còn disconnect.

## FIX-2026-10-13-10 — Pet fusion quality/tooltip/random

- Client contract: 1 chính + 4 phụ cùng template/cùng tier; success rates 100/80/50/30; responder `{flag,pid,grow}`.
- Nguyên nhân tooltip: root growth đổi nhưng nested `property` không refresh. Runtime #1063 xác nhận `CallBack.onRefreshPetProp` nhận đúng một wrapper `{id,s}`, không phải hai positional args. Fusion trước đó còn cộng phần tăng bậc vào `GrowRateAdd`, khiến client hiển thị `1.50 + 0.11` thay vì phẩm chất cơ bản tím.
- Sửa: validate template/tier, chặn max tier, giữ random success; khi thành công đặt `GrowRate` cơ bản ngẫu nhiên theo từng 0.01 trong bậc kế tiếp và không đổi `GrowRateAdd`; DTO tách base/add; callback dùng wrapper `{id,s}`.
- Hồi quy bổ sung: build cũ từng lưu phần tăng bậc vào `GrowRateAdd` (điển hình `1.50 + 0.11`). Fusion thành công nay đặt `GrowRateAdd=0` để không giữ lại hệ số cường hóa giả trên Pet đã bị dữ liệu cũ làm bẩn.
- Test: failure/success/replay, incompatible materials, max tier, rates, DTO growth contract, pet handler/cache/postgres focused tests PASS.
- Trạng thái: CODE_FIXED — chưa live-test tooltip/fusion animation trong Flash.

## FIX-2026-10-13-11 — Nâng cấp kỹ năng Pet báo method not found

- Nguyên nhân: Flash gọi fire-and-forget `petUpSkill(petID, slot)` nhưng dispatcher chưa đăng ký RPC.
- Sửa: thêm application flow tìm skill cùng `codeName` ở level kế tiếp, tiêu hao manual theo phẩm chất 1365/1366/1367 (level 1→2: 1; level 2→3: 3), lưu Pet và gửi callback inventory/Pet.
- Test: focused upgrade service và toàn bộ pet handler PASS; targeted vet/build PASS.
- Trạng thái: CODE_FIXED — chưa smoke-test Flash và chưa kiểm chứng transaction thật khi persistence item thất bại.

## FIX-2026-10-13-12 — Đặt skill Pet vào auto gây #1010

- Nguyên nhân: `PetManagerPanel.onInitViewPetMngP` bỏ qua `data.pi`/`data.soulInfo` khi merge và chờ hai object này ở wrapper root; DTO server chỉ trả `{data: ...}`, làm `Core.updatePetSetting` đọc `pi` undefined.
- Sửa: `Pet.ToDTO` giữ `pi` và `soulInfo` ở wrapper root, đồng thời bổ sung `bt8` vào payload `pi`.
- Test: DTO contract và toàn bộ pet handler PASS; targeted vet/build PASS.
- Trạng thái: CODE_FIXED — chưa kéo-thả skill trong Flash sau deploy.

## FIX-2026-10-13-13 — Nút xóa nhân vật bị vô hiệu hóa

- Working tree hiện có implementation server schedule/restore/purge và các artifact client/SWF từ một luồng khác; không ghi đè hoặc nhận là thay đổi của nhóm này.
- Test server: auth handler và PostgreSQL package PASS; build PASS.
- Trạng thái: PARTIAL/UNVERIFIED — client source có thay đổi ngoài phạm vi và chưa smoke-test; purge mặc định tắt.

## 2026-10-13 - Immediate Character Delete / 30 FPS Shared Link
- Added immediate delete RPC using ownership-guarded SQL, secondary MD5 validation and cache invalidation.
- Kept GameLoaders30.swf public URL, using fresh internal main asset 32 to bypass old main cache.
- Migration and cleanup are staged; live rollout must be independently verified.

## FIX-2026-10-14-14 — 55 họ skill chủ động Pet dùng sai damage/effect

- Nguyên nhân: adapter dùng `expSkill`/`dexSkill` progression làm combat formula, buff ID ngoài 1..10 bị bỏ qua, skill support vẫn đi damage path và thiếu cơ chế battle-local cho stat/periodic/shield/control/special.
- Sửa: thêm registry family+level phủ 153 template/55 family, domain `SkillPlan`/effect engine và các primitive đã duyệt. Bổ sung chín template nguồn Pet đặc thù: `Hổ Khiếu Sơn Lâm`, `Thợ Săn Sát Thủ`, `Tứ Phương Khiếp Đảm`; thêm Accuracy modifier, random-target RNG battle-local và remove-all-beneficial effect.
- Test: registry khớp chính xác 144 template sách Pet cộng 9 template đặc thù; focused application/domain PASS; `go vet ./...` PASS; build `bin/gameserver` PASS. Broad internal regression chỉ còn baseline ngoài phạm vi ở item/Pet/chat.
- Trạng thái: CODE_FIXED — chưa Flash smoke-test/deploy; không restart server và không đổi dữ liệu production.

## FIX-2026-10-14-15 — Hệ số cường hóa/tooltip Pet và Mảnh Pasar

- Nguyên nhân cường hóa: `UpgradeStarWithItem` chỉ tăng `UpgradeNum`, không đọc `proplNum` của Nội Đơn nên `GrowRateAdd` luôn giữ nguyên.
- Nguyên nhân tooltip: ô Pet gọi `gdc(39, petId)` nhưng server trả dữ liệu rỗng; `TipCre` còn cần năm trường `att*` ở root bên cạnh `skill1..skill15`.
- Nguyên nhân Pasar: template 5351 đi vào generic type-550 item effect rồi trả `cannot use item`.
- Sửa: validate Nội Đơn type 507/useType 4/propType 3 và cộng `proplNum/100`; nối `gdc` Pet với detail DTO có ownership; chặn template 5351 thành silent no-op trước generic dispatch, không tiêu hao.
- Test: toàn bộ 10 Nội Đơn checked-in; DTO/`gdc` tooltip; application + handler Pasar; toàn bộ gamedata/item handler và domain Pet PASS. Đã bổ sung import/setter/wiring Buff Service cho combat handler; `go vet ./...` và build `bin/gameserver` PASS. Diff-check các file phạm vi PASS; `item/service.go` hiện còn trailing whitespace ở hunk encounter-book ngoài luồng.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test; không thay đổi dữ liệu production.

## FIX-2026-10-17-16 — Character/Fairy skill authority và vật phẩm Pha Lê

- Nguyên nhân: học/nâng skill thường còn tin đường client và thiếu exact-next/cost authority; combat chấp nhận level/skill không thuộc actor; Fairy book/config/guard mutation thiếu canonical validation và gameplay không đọc Fairy; currency item 5948/5949 có thể rơi về bạc do mapping 73–75 không thống nhất.
- Sửa: canonical hóa currency `73/74/75` và bốn award 5947/5948/5949/5993; thêm ordinary/life-skill progression có lock, exact-next và debit/rollback; combat xác minh ownership Character/Pet và bỏ client-level scaling; Fairy dùng canonical 22 family, một guard, config an toàn, base/growth/satiety bonus, sáu passive và sáu active battle-local đủ dữ liệu. Các active family thiếu timing/formula vẫn fail-closed và log warning.
- Test: focused currency/item/skill/life-skill/Pet/combat/Fairy suites PASS; targeted vet PASS; `go build -o bin/gameserver.exe ./cmd/gameserver` PASS; `git diff --check` không có lỗi whitespace ngoài cảnh báo line ending. Broad `go test ./internal/... -count=1` chỉ còn baseline ngoài phạm vi: item award-quality/starter-gift, Quest registry `235 want 227`, và hai chat-stat tests.
- Trạng thái: CODE_FIXED — chưa deploy/restart hoặc Flash smoke-test; không dùng item/người chơi thật, Quest 166 hay Pet Nấm Lục; không migration hoặc sửa player data.

## FIX-2026-10-18-18 — RPC baseline, Thiết lập Pet và trang Thăng Cấp Chân Hồn

- Xác nhận full registration order không để activity backlog ghi đè handler thật của Ma Tâm, Trang Viên, Hồn Khí, Pet Arena, Find Back, PPVE và PRS; Card vẫn là stub có chủ đích, sweep vẫn chưa đăng ký trước Unit D.
- Thiết lập/Đấu Trường Pet nay validate Pet sở hữu duy nhất, vị trí 5–9, 2–5 Pet khi báo danh, tối đa 24 lệnh, action/target và skill sở hữu; tách state farm/HMTXLS/arena và dùng arena snapshot để chọn Pet chiến đấu.
- `addSoulTreeLvl` đi theo `TBL_PRS_TREE.next_id`, khóa mutation theo nhân vật, trả full panel state và hoàn Chân Hồn Thạch khi save state thất bại. Compensation tương tự được thêm cho đổi chip và paid collect.
- Test: combined registration, Pet Arena application/handler và PRS application/handler PASS; targeted vet PASS. Không migration, deploy/restart, ActionScript, player-data mutation, commit hoặc push.
- Trạng thái: CODE_FIXED — cần Flash smoke-test bằng nhân vật/Pet test sau khi được phép; Pet Arena combat vẫn chỉ mô phỏng lead Pet.

## FIX-2026-10-18-19 — PPVE không còn tự thắng/tự tăng tầng

- Nguyên nhân: `challengeNextFloor` gọi thẳng bookkeeping rồi push `BATTLE_WIN` dù chưa có combat hoặc floor-opponent roster.
- Sửa: fail-closed tại handler, phát loading + `BATTLE_NOT_END`, không gọi mutation service nên không trừ lượt, tăng tầng, rank hoặc KP.
- Test: activity handler và full `internal/application/pet` PASS; targeted vet PASS.
- Trạng thái: SAFETY_FIXED/BLOCKED — challenge thật chờ nguồn roster đối thủ authoritative; chưa deploy/restart hoặc Flash smoke-test.

## FIX-2026-10-18-20 — Ma Tâm authority và Hồn Khí catalog-backed

- Ma Tâm: gắn tim nay kiểm tra catalog, loại tim/ô, category, lỗ mở và số lượng sở hữu; mutation được serialize theo nhân vật. Stat trực tiếp của tim đã vào effective character stats theo đúng divisor Flash. Các RPC tiêu item/tiền và combine bonus tiếp tục fail-closed, không đoán economy.
- Hồn Khí: thêm ownership/equip/show lifecycle, suit activation, nâng một/mười lần bằng cả Bí Ngân + bạc khóa, catalog stat provider và compensation khi save state lỗi. Rune write dùng cùng khóa mutation để tránh ghi đè JSONB.
- Test: Monster Heart application/statfeature/handler; Dress application/handler; Rune application/handler; gamedata focused đều PASS. Chưa migration, deploy/restart, ActionScript, player-data mutation, commit hoặc push.
- Trạng thái: B PARTIAL_SAFE, F CODE_FIXED — cần broad vet/build và Flash smoke-test được cấp phép.

## FIX-2026-10-18-21 — Card, sweep và HMTXLS fail-closed đầy đủ RPC

- Thẻ Bài trả explicit disabled state đúng nhánh Flash và đăng ký đủ start/change/claim/add-time dưới dạng no-op; không tạo session bị kẹt hoặc cấp reward đoán.
- Sweep đăng ký đủ bảy RPC client gọi nhưng không tạo eligibility, timer, cost, quest progress hay reward giả.
- HMTXLS đăng ký đủ public/advanced entry, exit và claim; mọi mutation bị chặn và refresh `onHMTXLSData` rỗng, không còn báo giả “Làm mới dữ liệu” như đã vào phòng.
- Test: activity backlog/icon-panel và combined dispatcher PASS. Không migration, deploy/restart, player-data mutation, commit hoặc push.
- Trạng thái: COMPATIBILITY_FIXED/GAMEPLAY_BLOCKED — chờ nguồn authority đã ghi trong plan.
