# Legacy Feature Completeness Audit

**Date**: 2026-10-18  
**Status**: Complete; implementation plan awaiting approval  
**Scope**: Ma Tâm, Thẻ Bài Ma Thuật, tự động hoàn thành phụ bản, Trang Viên Pháp Thuật, Hồn Khí, thiết lập/Đấu Trường Pet, Phần Thưởng Hồi Phục, Tu Luyện Ảo Ma, Không Gian Điêu Khắc, and page 2 of Chân Hồn Pet.

No ActionScript, database, player data, runtime service, or gameplay code was changed during this audit.

## 1. Evidence order

1. Decompiled Flash client under `docs/client/` is authoritative for RPC names, argument order, callbacks, and payload shape.
2. Current Go source, tests, migrations, and checked game-data are authoritative for server coverage.
3. Official articles on `vuaphapthuat.go.vn` are supporting evidence for player-facing mechanics, not a replacement for missing formulas or wire contracts.
4. Missing probabilities, costs, opponent rosters, and reward tables remain unknown and must not be invented.

## 2. Terminology decisions

- **Ma Tâm** in this request is treated as the `MonsterHeartPanel` system, not the identically named Pet skill or rebirth quest target. The client label is `Language.MONSTER_HEART[0]`.
- **Thẻ Bài Ma Thuật** is treated as `CardGamePanel`/Thẻ Bài Pha Lê because the official article describes five-card combinations and Bốc Quẻ. It is not `SmallGameHideSeekPanel` despite that panel containing the phrase “Thẻ Bài Kỳ Diệu”.
- **Đấu Pet kiểu** has no exact client or official literal. The strongest match is Pet fight lineup/AI configuration used by Trang Viên, Tu Luyện Ảo Ma, and Đấu Trường Pet. This audit therefore covers `PetFightConf`, not the separate MCZD event.
- **Hồi phục** means `FindBackPanel`/Phần Thưởng Hồi Phục, not HP/MP potion recovery.
- **Trang 2 Chân Hồn Pet** means the second outer tab shown by `PRSPanel`: **Thăng Cấp**, backed by `addSoulTreeLvl`. If the report instead means the second divine-form subpage, that is `activePRSShowSpe`; both paths were audited.

## 3. Current coverage matrix

| Feature | Flash contract | Current server state | Verdict |
|---|---|---|---|
| Ma Tâm | `initMonsterHeartData`; set/remove/open holes; upgrade/resolve heart; upgrade box; `updateMonsterHeartPanel` family callbacks | Init and set/remove persist a state blob. Open-hole, upgrade, resolve, box progression, inventory authority, and real stat contribution are absent. | **Partial/state-only** |
| Thẻ Bài Ma Thuật | `initCardPlayPanel`, `startGame`, `changeCard`, `getCardAward`, `addPlayTime` | Only a legacy `initCardPlayPanel` starter payload exists. No session, deck, limit, reward, or claim state. | **Stub** |
| Tự động hoàn thành phụ bản | `initTaskSweepPanelByClient`, `getTaskBattleTime`, `taskSweepSure/Cancel/End/LeftTimeCheck`, `taskFinishByGold` | No matching server RPC exists. Star-instance memory explicitly lists this path as deferred. | **Missing** |
| Trang Viên Pháp Thuật | `getFarmByCid`, `getFriendFarm`, `addMineral`, `harvestMine`, `steelMine`, `delMinearl`, `addFarmNum`, `farmLvUp` plus farm-bag calls | Main gameplay and persistence exist. Farm-bag withdrawal grants inventory before removing the estate slot; stealing similarly crosses repositories without one transaction. | **Implemented, unsafe transfer boundary** |
| Hồn Khí | `initDecoratePanel`, activate/equip/show/hide/level decoration, upgrade one/ten hole levels, suit properties | Rune bag/socket paths and one-hole upgrade exist. Decoration ownership/show lifecycle, ten-times upgrade, and real suit activation are missing or synthetic. | **Partial** |
| Thiết lập/Đấu Trường Pet | `getPetConf`, `getPetArenaConf`, `petFightConf(data,1,isXiulianshi)`, `petArenaTicket(team,data)` | Exact RPCs are registered and persisted through the Pet Arena repository. Current save path always uses config key `farm`; validation of owned pets, positions, and AI command skill ownership requires strengthening. | **Implemented, validation gap** |
| Phần Thưởng Hồi Phục | `getFindBackData`; six claim RPCs; `onFind` responder | Historical entitlement, atomic wallet/EXP/Pet claims, and claim journal exist. `findBackXuanShangByBanner` remains a no-op because banner consumption is pending. Some logic still uses inline SQL rather than the project repository/function pattern. | **Mostly implemented** |
| Tu Luyện Ảo Ma | `getHMTXLSData`, `enterAdvRoom`, `enterPubRoom`, `endHMTXLRoom`, `getHMTAward`; reuses `petFightConf` | `getHMTXLSData` and `enterPubRoom` return compatibility data only. There is no room occupancy, timer, settlement, or claim state. | **Stub** |
| Không Gian Điêu Khắc | `initPPVEPanel`, rank/config/challenge/attempt/KP exchange, carve-node upgrade callback | Panel state, lineup, counters, KP exchange, and rank read model exist. `challengeNextFloor` advances immediately and returns a fabricated win without combat. No verified floor-opponent roster is present. | **Unsafe partial implementation** |
| Chân Hồn Pet page 2 | `initPRSPanel`, `addSoulTreeLvl`; alternative subpage uses `activePRSShowSpe` | Both are registered and persisted. Main-bag consume and PRS save are not one transaction; paid operations can also split deduction and state save. | **Implemented, atomicity/runtime verification gap** |

## 4. Verified Flash contracts

### 4.1 Ma Tâm

- Panel: `docs/client/view/view/compDragable/MonsterHeartPanel.as`.
- Slot actions: `docs/client/view/view/comp/MonsterHeartSlot.as`.
- Init result: `{data,bag}` through `onInitMonsterHeartData`.
- Mutation RPC arguments:
  - `monsterHeartSet(bagPos,holePos,box,type)`
  - `monsterHeartReMove(holePos,box)`
  - `openHoleMonsterHeart(box,holePos)`
  - `openHoleMonsterHeartJingang(box,holePos,itemKK)`
  - `upgradeMonsterHeartItem(isAll,giid,tempNumUp,useJinshiHeart,useByGold)`
  - `resolveMonsterHeart(giid,resNum)`
  - `upgradeMonsterHeartBox(boxIndex)`
- Checked tables: `TBL_CREATUREH_HEART`, `TBL_CREATUREH_COMBINE`, `TBL_CREATUREH_CONTAIN`, and `TBL_CREATUREH_POINT`.

### 4.2 Thẻ Bài Ma Thuật

- Panel: `docs/client/view/view/compDragable/CardGamePanel.as`.
- RPC/callback pairs: `initCardPlayPanel`/`onInitCardPanel`, `startGame`/`onStartGame`, `changeCard`/`onChageCard`, `getCardAward`/`onGetCardAward`, and `addPlayTime`/`onAddPlayTime`.
- Session payload includes at least `flag`, `max`, `now`, `s1..s5`, `changeTime`, and optional `useTime`.
- No checked card-specific reward/probability table was found. A server implementation may persist and validate the card session, but must not fabricate awards.

### 4.3 Auto-sweep

- Panel: `docs/client/view/view/compDragable/TaskSweepPanel.as`.
- The client supplies task and selected-count data, but eligibility, cost, duration, completion, and rewards must be re-derived server-side.
- The official guide confirms: the player must already have cleared the instance and hold its task; normal start costs silver; instant completion costs gold; levels 100–119 select one instance/day and 120+ select two.

### 4.4 Trang Viên

- Legacy panel: `FazendaPanel`/`FazendaFarm`.
- Official mechanics distinguish this from the life skill Trồng Trọt: one creature per plot, harvest into the estate warehouse, plot expansion by estate level, and stealing subject to stock thresholds.
- Server coverage is broad; the material defect is consistency across estate and normal inventory stores.

### 4.5 Hồn Khí

- Panel: the first two tabs of `DecoratePanel`; mutations originate in `DecorateItem` and `DecoratePanel`.
- Core state per position includes `{did,position,isShow,hid,showLvl}`.
- Official articles confirm four Hồn Khí positions, one selected effect per item, six qualities, level cap 50, effect tiers at levels 30 and 50, character stat contribution, and Bí Ngân/Ngân Phiếu costs.
- Exact per-level costs and success behavior must come from checked tables/client constants, not the article summary.

### 4.6 Pet setup

- Panels: `PetFightConf`, `PetConfigCanvas`, and `AIConfPanel`.
- `conf1` uses positions 5–9. Each row includes `{pid,pos,cmdList}`; commands distinguish basic attack and skill and include target code.
- Official Pet Arena documentation confirms 2–5 pets, per-Pet AI, a repeated sequence of up to 24 actions, daily attempts, and rank exchange on an upset win.

### 4.7 Find Back

- Panel: `FindBackPanel`.
- `expType=1` is silver/70%; `expType=2` is gold/100%.
- Result shape is `{type,num,exp,refresh?,color?}`.
- Official documentation confirms no activity-point recovery and no VIP EXP recovery.

### 4.8 Tu Luyện Ảo Ma

- Panel: `XiulianshiPanel`.
- `getHMTXLSData` pushes `onHMTXLSData`; room entry/exit and award RPCs are fire-and-forget.
- Minimum state contains server time, level, public/advanced/normal room data, and optional award state.
- The official site confirms the feature name but does not publish room timing, occupancy limits, costs, accumulation rate, or reward formula.

### 4.9 Không Gian Điêu Khắc

- Panel: `PetPVESystem`; lineup: `PetPVEConfigPanel`; carve UI: `PetPVEKPIcon`.
- Official documentation confirms 150 floors, Văn Chỉ rewards, carve slots unlocked by floor progress, and Pet stat bonuses applicable in Pet Arena and PPVE.
- Neither the official article nor checked static data provides a verified floor-to-enemy roster. Returning a win without combat is not acceptable.

### 4.10 Chân Hồn Pet

- Panel: `PRSPanel` with outer tabs `[Cây Chân Hồn, Thăng Cấp, Thần Hóa, Đổi Thần Dụ]`.
- `addSoulTreeLvl(cid)` is the second outer tab progression action.
- Official documentation confirms five qualities with ten levels each and Pet stat contribution, but does not override table costs or client payload shape.

## 5. Official web sources

- Thẻ Bài Ma Thuật: <http://vuaphapthuat.go.vn/danh-muc-bai-viet/the-bai-ma-thuat/444.html>
- Hoàn Thành Phụ Bản: <https://vuaphapthuat.go.vn/danh-muc-bai-viet/hoan-thanh-phu-ban/473.html>
- Trang Viên: <http://vuaphapthuat.go.vn/bai-viet/trang-vien/4349.html>
- Trồng Trọt, a separate system: <http://vuaphapthuat.go.vn/bai-viet/trong-trot/4362.html>
- Cường Hóa Hồn Khí: <http://vuaphapthuat.go.vn/bai-viet/cuong-hoa-hon-khi/6335.html>
- Đấu Trường Pet: <http://vuaphapthuat.go.vn/danh-muc-bai-viet/dau-truong-pet/196.html>
- Phần Thưởng Hồi Phục: <http://vuaphapthuat.go.vn/danh-muc-bai-viet/phan-thuong-hoi-phuc/672.html>
- Tu Luyện Ảo Ma release reference: <http://vuaphapthuat.go.vn/bai-viet/09-06-update-phien-ban-moi-mua-he-soi-dong/9841.html>
- Không Gian Điêu Khắc: <https://vuaphapthuat.go.vn/danh-muc-bai-viet/khong-gian-dieu-khac/1988.html>
- Chân Hồn Thần Thú: <https://vuaphapthuat.go.vn/danh-muc-bai-viet/chan-hon-than-thu/1491.html>

The site pages are old and sometimes served through HTTP or return navigation-heavy HTML, but their article bodies and titles remain indexed. Confidence is high for the mechanics explicitly listed above and low for any absent formula.

## 6. Safety conclusions

- Do not replace an existing real handler with another compatibility stub; dispatcher registration is last-wins.
- Do not trust client task IDs, counts, elapsed time, Pet IDs, skill IDs, target codes, reward choices, or claimed floor wins.
- All inventory/currency plus feature-state mutations need one transaction or explicit compensation with idempotency.
- Card awards, HMTXLS rewards, and PPVE enemies remain blocked until a checked source is recovered. Their UI/read state may be implemented, but success must fail closed rather than grant guessed value.
- Runtime smoke tests require explicit deploy/restart permission and test-only characters/items.
