import assert from "node:assert/strict"
import { test } from "node:test"

import { buildNpcPatch, draftShopSlots } from "../src/lib/npcs/editor.ts"

const slot = {
    id: 10,
    amount: -1,
    gold: 0,
    gt: 0,
    item_id: 1,
    money: 40,
    p_num1: 0,
    p_num2: 0,
    p_type1: 0,
    p_type2: 0,
    point: 0,
    position: 1,
    price_all: 0,
    quality: 0,
    r: 0,
    sale: 0,
    sid: 2,
    st: 0,
    type: 29,
    displayName: "Thuốc thử",
    iconCode: 123,
}
const detail = {
    npc: {
        id: 1,
        name: "NPC",
        on_service_text: "Chào",
        shop_id: 2,
        pos_map_id: 10,
        pos_x: 100,
        pos_y: 200,
    },
    shop: { slots: [slot] },
}

test("NPC editor emits only changed fields", () => {
    const rows = draftShopSlots([slot])
    assert.deepEqual(buildNpcPatch(detail, { ...detail.npc }, rows), {})
    rows[0].money = 50
    assert.deepEqual(
        buildNpcPatch(detail, { ...detail.npc, name: "NPC 2" }, rows),
        {
            npc: { name: "NPC 2" },
            slots: { update: [{ id: 10, changes: { money: 50 } }] },
        }
    )
})

test("NPC editor turns removal into soft hide", () => {
    const rows = draftShopSlots([slot])
    rows[0]._hide = true
    assert.deepEqual(buildNpcPatch(detail, { ...detail.npc }, rows), {
        slots: { hideIds: [10] },
    })
})

test("NPC editor includes changed map coordinates", () => {
    const rows = draftShopSlots([slot])
    assert.deepEqual(
        buildNpcPatch(
            detail,
            {
                ...detail.npc,
                pos_map_id: 11,
                pos_x: -20,
                pos_y: 350,
            },
            rows
        ),
        { npc: { pos_map_id: 11, pos_x: -20, pos_y: 350 } }
    )
})

test("NPC editor never writes derived display metadata", () => {
    const rows = draftShopSlots([slot])
    rows.push({
        ...slot,
        id: -1,
        position: 2,
        _key: "new",
        _new: true,
        displayName: "Tên trong game",
        iconCode: 999,
    })
    const patch = buildNpcPatch(detail, { ...detail.npc }, rows)
    assert.equal(patch.slots.insert[0].displayName, undefined)
    assert.equal(patch.slots.insert[0].iconCode, undefined)
    assert.equal(patch.slots.insert[0].item_id, 1)
})
