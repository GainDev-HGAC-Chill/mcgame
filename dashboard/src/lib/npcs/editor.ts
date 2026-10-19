import type {
    NpcDetail,
    NpcRecord,
    NpcSavePatch,
    ShopSlotInsert,
    ShopSlotRecord,
} from "@/types/npcs"

export type DraftShopSlot = ShopSlotRecord & {
    _key: string
    _new?: boolean
    _hide?: boolean
}

const editableSlotFields: (keyof ShopSlotInsert)[] = [
    "amount",
    "gold",
    "gt",
    "item_id",
    "money",
    "p_num1",
    "p_num2",
    "p_type1",
    "p_type2",
    "point",
    "price_all",
    "quality",
    "r",
    "sale",
    "st",
    "type",
]

export function draftShopSlots(rows: ShopSlotRecord[]): DraftShopSlot[] {
    return rows.map(row => ({ ...row, _key: `slot-${row.id}` }))
}

export function buildNpcPatch(
    baseline: NpcDetail,
    npc: NpcRecord,
    slots: DraftShopSlot[]
): NpcSavePatch {
    const patch: NpcSavePatch = {}
    const npcPatch: NonNullable<NpcSavePatch["npc"]> = {}
    for (const field of [
        "name",
        "on_service_text",
        "shop_id",
        "pos_map_id",
        "pos_x",
        "pos_y",
    ] as const) {
        if (npc[field] !== baseline.npc[field])
            npcPatch[field] = npc[field] as never
    }
    if (Object.keys(npcPatch).length) patch.npc = npcPatch

    const baselineSlots = new Map(
        (baseline.shop?.slots ?? []).map(row => [row.id, row])
    )
    const insert: ShopSlotInsert[] = []
    const update: NonNullable<NonNullable<NpcSavePatch["slots"]>["update"]> = []
    const hideIds: number[] = []
    for (const row of slots) {
        if (row._new) {
            if (!row._hide) {
                const {
                    id: _id,
                    sid: _sid,
                    _key: _key,
                    _new: _new,
                    _hide: _hide,
                    displayName: _displayName,
                    iconCode: _iconCode,
                    ...value
                } = row
                insert.push(value)
            }
            continue
        }
        const original = baselineSlots.get(row.id)
        if (!original) continue
        if (row._hide && original.st !== 5) {
            hideIds.push(row.id)
            continue
        }
        const changes: Partial<ShopSlotInsert> = {}
        for (const field of editableSlotFields) {
            if (row[field] !== original[field])
                changes[field] = row[field] as never
        }
        if (Object.keys(changes).length) update.push({ id: row.id, changes })
    }
    const slotsPatch: NonNullable<NpcSavePatch["slots"]> = {}
    if (insert.length) slotsPatch.insert = insert
    if (update.length) slotsPatch.update = update
    if (hideIds.length) slotsPatch.hideIds = hideIds
    if (Object.keys(slotsPatch).length) patch.slots = slotsPatch
    return patch
}

export function newShopSlot(shopId: number, position: number): DraftShopSlot {
    return {
        id: -Date.now(),
        sid: shopId,
        amount: -1,
        gold: 0,
        gt: 0,
        item_id: 1,
        money: 1,
        p_num1: 0,
        p_num2: 0,
        p_type1: 0,
        p_type2: 0,
        point: 0,
        position,
        price_all: 0,
        quality: 0,
        r: 0,
        sale: 0,
        st: 0,
        type: 29,
        displayName: null,
        iconCode: 0,
        _key: crypto.randomUUID(),
        _new: true,
    }
}
