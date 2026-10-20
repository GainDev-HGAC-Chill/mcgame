# Equipment live refresh, shop pet growth and local GM

## Operator report and scope

Operator confirmed bag sort/drag no longer errors and the other previously reported panel fields are okay. New report: equip removes a bag item and increases stats, but equipped icon appears only after relog. Requested existing/new GM page for test resources, shop pet growth1.5 and exhaustive missing-box review. No player item use, purchase, equipment change, resource grant or pet repair was performed by the assistant. Existing pet growth remains untouched.

## Equipment contract and fix

Flash researcher traced the existing actual SWF CallBack export `.local/research/mcgame-callback-as-research/scripts/com/qeedoo/game/system/CallBack.as`; inventory UI details are additionally from partial docs/client.

- onMoveItem fills destination then cleans source. Existing EquipOn sent incoming first, then displaced with source1+requestedSlotID. Auto selection uses slotID=-1, yielding sourceSID0; correcting only the SID would still clean the just-installed equipped icon because ordering is wrong.
- Empty-slot onMoveItem is supported; static evidence does not prove its failure root cause. Instance-cache refresh is separate from inventory-slot redraw, so the repair explicitly reconstructs both.
- `equip_slot_refresh.go`: capture oldSID before mutation; resolved equipped SID from persisted post-equip item is also the displaced original equipment SID. Delete all old positions before any add; each final record sends onChangeEquiptInst(dto), then onAddCharactorSlot(dto). No new IDs, no fabricated fields, owner-only. Unequip uses the actual final bag destination, including drag target.
- Existing stats/HP-MP/element/active-list/scene-appearance callbacks retained. No blanket onSwapItem because displacement destination need not equal incoming original bag slot. Template position and two-ring auto-selection behavior remain unchanged.
- Callback failures propagate, not silently pretend the UI was updated; inventory mutation already happened in the service, so failure does not imply safe reward/action replay.

Tests exercise empty accessory, replacement, both ring destinations, ring replacement, double-click unequip and drag unequip. Callback assertions require all old deletes first, same persistent IDs, actual final SID and instance update before slot addition. Existing item test transport now captures supported callbacks.

## Pet growth and deterministic shop contents

Before change the pet-description fallback called ContractPets and ignored stated quality1.5, falling back to creature grow_base0.8. It now parses description growth and calls ContractPetsWithQuality with scaled15→growth1.5. Parser preserves decimal punctuation and accepts accents/case/ascii/newlines/comma and bare1.5 such as Chiến Dương descriptions. Explicit higher description quality is not downgraded. Award-based pet quality remains raw1.5, not15. Missing/unclear growth is not arbitrarily guessed. No existing pet rows were updated.

Read-only live shop audit found325 candidates with no award rows; this is not325 proven broken reward boxes, because some use fallback or are storage/quest/non-box false positives. Full scope/classification is [shop audit](2026-10-12_02_SHOP_BAG_AUDIT_RESEARCH.md); snapshot `.local/research/shop-bag-audit-20261012.json`. Catalog listing does not certify current client-visible sale status.

Parent additionally checked `handlers/shop/config.go`: getShopConfig/getLimitShopConfig/getRemainShopConfig currently send empty configs, not additional live bag catalogs. System purchase resolver can accept item-template IDs outside shop_slot; those unlisted templates are outside the325-row listing audit, so do not call it every potentially purchasable item or all future shop rotations.

Completed independent exhaustive snapshot classification:25 supported pet candidates,12 partial pet candidates,70 deterministic missing candidates,32 blocked specialized cases,112 ambiguous cases and74 false positives;48 have all listings hidden. After13 deterministic configurations,57 of the original deterministic category still need target/semantics resolution. Counts are static audit classes, not real opening PASS or exact current client availability. Report includes every325 IDs and preserved listing states. Parent corrected exploratory Vải target12 to live common material25 before implementation.

Installed13 catalog-description-defined guaranteed awards via `supabase/data/shop_deterministic_awards.sql`:

- Material bags1455/1456/1457/1458/1459/1469/1470/1472/1473/1474:12 each, client ColorCode3=material grade4. IDs24/21/23/26/28/7/18/22/25/27. Generic Vải maps to common Vải Thô25, not premium Gấm Vóc26.
- Nội Đơn Bách Luyện bag1616:10 item274.
- Camy3254:pet1088 growth1.5; Dê Con II4840:pet728 growth1.5. These no-literal-pet descriptions did not match existing fallback.

Installer verifies current descriptions, shop listing, reward ID/name/type and conflicting rows; no overwrite/delete of existing configs. Applied twice idempotently: total67 = original starter54 + shop13. The temporary pre-change dump was removed during repository cleanup after the complete award table was exported to `supabase/seeds/data_tbl_item_award.sql`; psql restrict markers remain commented per repository convention. No schema migration or player data modification. Unknown random rates/bonus pools, class-dependent packages and opaque boxes remain unconfigured—do not manufacture success or arbitrary rewards.

## GM page found, not created

Existing local page http://127.0.0.1:3000/dashboard/players/1 (Gain) has Wallet add/set, Resources capacities, Items/Add Item and richer items catalog at /dashboard/items. No grant was made. Existing-pet edit is available, direct pet creation is not. Online wallet changes use cached dirty state and normal persistence; item sends attempt flush and live callback. Reopen/relog may be necessary for database-only/callback-failed modes.

Security finding: current dashboard middleware/layout/session do not enforce app login; admin HTTP handlers do not validate proxy secret. Older docs describing /api/admin/auth are not current. Keep localhost bindings; no ports exposed or firewall changes. Creating a new unauthenticated GM page would not improve this.

## Validation status

- PASS: affected item application/handler suites, vet, host build.
- PASS: isolated equipment callback regression, description growth parsing and13 real SQL-manifest opening fixtures; material grade/count and pet IDs/growth checked, existing starter tests retained.
- PASS: database installer twice, total67 and original54 preserved. No player tables changed by these statements.
- PASS: redeployed only mcgame-server. Startup00:23:12 loads itemAwards67 and RTMP started; running/restart_count0, game check PASS, GM route HTTP200. PostgreSQL/Redis not recreated/reset. Immediate health is not sustained stability.
- NOT TESTED: live Flash equip/replace/unequip without relog; pet bag purchase/opening; GM resource grant/persistence. Tests/health do not certify actual gameplay or all shop boxes.
