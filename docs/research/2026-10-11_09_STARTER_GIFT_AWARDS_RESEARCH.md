# Starter gift chain: approved local reward configuration

## Scope and operator decisions

The operator cancelled the pending code rollback before any reversion and supplied the full rewards for six starter bags (levels1,10,20,30,40,50). Preserve all previous server fixes and player data. Follow-up approval selects **purple Trác Việt** for the four level1 jewelry items and **Hổ Bì Dương template91** with growRate1.9 at level30. The operator approved restarting the stopped existing stack and updating only mcgame-server. No player item use/reward replay is authorized.

Source error: real Gain RPC `useItem(1,-1,2)` at22:07:10 used template2918 with no award rows. Before configuration, the database had zero item-award rows and instance2 remained stack_count1.

## Catalog mapping and data

Existing bag IDs2918–2923 have required levels1/10/20/30/40/50. The complete operator-approved mapping is embedded as JSON in `supabase/data/starter_gift_awards.sql`; this is a data-only transactional installer, not a schema migration. It verifies all IDs/names, source type550/useType1/required level, expected54 rows, and existing row conflicts. Applies without deleting/overwriting other award data. Reapplication is idempotent; mismatched existing rows cause transaction rollback rather than overwrite.

| Bag | Level | Guaranteed rows | Total reward quantity |
|---|---:|---:|---:|
|2918|1|9|53|
|2919|10|9|87|
|2920|20|9|22|
|2921|30|11|24|
|2922|40|8|47|
|2923|50|8|64|

Every row has payload.guaranteed=true. Merely assigning rate100 would incorrectly select one weighted reward, not grant the whole list. IDs810000001–810000054 are reserved by this manifest; conflicts fail closed. Bag50 terminates the chain.

Key mapping decisions:

- Jewelry1009/1010/1008/1007: award type19, quality20, preNameType5 = purple Trác Việt. Prefix and color are separate semantics.
- Pet91: award type12, raw quality1.9 (not19). Client colorByGrowRate maps1.9 to purple; creature-template colorCode unchanged.
- Diamond16: award type29, count5, payload.materialColorCode3 = material Cấp4. Client material suffix indices0–4 mean levels1–5, unlike equipment quality bands. Added scoped award color support used by both capacity preflight and item granting. Invalid/nonmaterial explicit overrides reject before grants; existing awards without this field keep their behavior.
- Chìa Khóa May Mắn273 is the permanent exact-name catalog item, not dungeon-key4033 with leading whitespace/180-minute lifetime.
- Túi Quà Trang Bị Trác Việt Tím Cấp30/40 are item2049/2050. This request grants those boxes themselves; it does not invent their12-item internal rewards or implement unsupported behavior of other granted consumables.
- Catalog Hoàng Kim Lễ1064 requires use level50 even when awarded in bag40; preserve this existing requirement. The requested bag50 HP/MP medicines371/370 require level60 to use; do not silently change template restrictions.
- Seven-day items1515/1615 retain their catalog t=10080 semantics; no new expiry subsystem is added.

## Validation and rollout

- PASS: all six full-package opening simulations receive every configured reward/count, consume exactly one box, and verify jewelry/pet/material quality. Isolated character99/mock repositories only.
- PASS: below-level and full-bag openings preserve each box and grant nothing; retrying a consumed bag grants nothing. Existing award bag-capacity regression tests remain green.
- PASS: item application/handler suites, affected vet and host server build.
- PASS: project database installer ran twice; persisted row counts9/9/9/11/8/8, all guaranteedtrue. No schema or player tables changed.
- The temporary pre-change reward-table dump was empty and was removed during repository cleanup; the post-configuration source of truth is `supabase/seeds/data_tbl_item_award.sql`, exported via compose pg_dump. No manually authored migration.
- Exported seed's psql-only restrict/unrestrict markers are commented, matching repository seed convention; reward INSERT data remains the dump output.
- Initial redeploy timed out because relocated `.local` was included in build context. Added `.local/` to `.dockerignore` to exclude database disks/backups/caches from images; redeploy retried without those artifacts.
- Runtime rollout/real gameplay status is recorded in `docs/local-lan-deployment/test-results.md`; tests and infra health alone do not certify real Flash item opening.
- PASS: retry built/deployed only mcgame-server; startup22:51:18 loads itemAwards54, RTMP listener started, infra check PASS. Real Flash use remains NOT TESTED; Gain's bag was not automatically opened.

Known broader limitation: the existing award engine preflights capacity but is not a newly designed database transaction across all equipment/item/pet writes; an unrelated repository failure mid-grant can still require diagnosis, not automatic reward replay.
