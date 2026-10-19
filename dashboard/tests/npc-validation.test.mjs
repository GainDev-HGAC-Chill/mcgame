import assert from "node:assert/strict"
import { test } from "node:test"

import {
    npcListQuerySchema,
    npcSaveSchema,
} from "../src/lib/npcs/validation.ts"

const slot = {
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
    st: 0,
    type: 29,
}
const base = {
    expectedRevision: "a".repeat(32),
    expectedSharedCount: 1,
    confirmShared: false,
    reason: "test reason",
    requestId: "11111111-1111-4111-8111-111111111111",
}

test("NPC save accepts bounded aggregate patches", () => {
    assert.equal(
        npcSaveSchema.safeParse({
            ...base,
            patch: {
                npc: { name: "NPC mới", on_service_text: null, shop_id: 2 },
            },
        }).success,
        true
    )
    assert.equal(
        npcSaveSchema.safeParse({
            ...base,
            patch: { slots: { insert: [slot] } },
        }).success,
        true
    )
    assert.equal(
        npcSaveSchema.safeParse({
            ...base,
            patch: {
                slots: {
                    update: [{ id: 1, changes: { money: 50 } }],
                    hideIds: [2],
                },
            },
        }).success,
        true
    )
})

test("NPC save rejects ownership and unsafe slot fields", () => {
    for (const patch of [
        {},
        { npc: {} },
        { slots: {} },
        { npc: { id: 2 } },
        { slots: { update: [{ id: 1, changes: { sid: 2 } }] } },
        { slots: { update: [{ id: 1, changes: { position: 2 } }] } },
        { slots: { insert: [{ ...slot, point: 1 }] } },
        { slots: { insert: [{ ...slot, type: 12 }] } },
    ])
        assert.equal(npcSaveSchema.safeParse({ ...base, patch }).success, false)
})

test("NPC list query is bounded and strict", () => {
    assert.equal(npcListQuerySchema.parse({ hasShop: "true" }).hasShop, true)
    for (const query of [
        { page: 0 },
        { pageSize: 101 },
        { mapId: -1 },
        { hasShop: "yes" },
        { hasDialogue: "0" },
        { table: "player.characters" },
    ])
        assert.equal(
            npcListQuerySchema.strict().safeParse(query).success,
            false
        )
})
