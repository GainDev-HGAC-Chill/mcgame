# PK and Character Rename Achievement Authority

**Date**: 2026-10-27  
**Scope**: Requirements 138–141 and 150. Static contract and persistence verification only; no database migration or player mutation was performed.

## PK requirements 138–141

- Catalog requirements are authority-target rows (`num=0`) describing PK point thresholds 10, 25, 100, and 200.
- The Flash scene character view treats `honor` as the red-name severity: `CharactorView.setName()` calls `setRedName(_gameObject.honor)` whenever `honor > 0` (`docs/client/view/view/compGameStage/CharactorView.as:84-93,660-663`).
- Login and scene payloads already expose persisted `Character.Honor` as `honor`; PostgreSQL scans it from the character wallet.
- Therefore `Character.Honor` is the existing state authority. The four requirements are positive-only state proofs with exact targets 10/25/100/200; no new counter or backfill is needed.

## Rename requirement 150

- Catalog requirement 150 / achievement 3040 is “Sử dụng Thẻ Đổi Tên 1 lần”.
- `TBL_ITEM_TEMPLATE` identifies template 2126 as `Thẻ Đổi Tên`.
- Flash/server RPC remains `changeNameByCard(newName)` through the existing item handler.
- The previous server flow consumed template 2126 and updated the character name through two independent cache mutations. A failure could consume the card without renaming, and no durable achievement marker existed.
- The authoritative boundary is now one PostgreSQL transaction: lock one bag card 2126, update `player.characters.name`, decrement/delete that exact item instance, and append `character-renamed:v1:<operationUUID>`. Any name conflict, missing card, or fact failure rolls the whole transaction back.
- Payload binds the operation UUID, consumed card instance ID, and persisted new name. The consumer maps it only to requirement 150 with target one. The handler processes pending facts and sends the normal achievement delta only after commit; login remains the recovery path.

## Resulting authority census

- State: 476 requirements.
- Event: 80 requirements.
- Unsupported: 141 requirements.

The generic append-fact migration `20261026120000_pet_acquisition_achievement_facts.sql` remains unapplied, so this code must not be deployed before that database prerequisite is approved and installed.
