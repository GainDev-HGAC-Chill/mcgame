import assert from "node:assert/strict"
import { test } from "node:test"

import {
    normalizeBoxItemAwardModePayload,
    orderBoxItemAwardRows,
    resolveBoxItemAwardMode,
} from "../src/lib/box-item-awards.ts"
import { boxItemAwardSaveSchema } from "../src/lib/box-item-award-config.ts"

const award = {
    awardId: 1676,
    type: 19,
    count: 1,
    rate: 100,
    quality: 10,
    preNameType: 3,
    payload: {},
}

test("selectable payload conversion writes and removes reserved metadata", () => {
    const selectable = normalizeBoxItemAwardModePayload(
        { guaranteed: true, countMin: 1, countMax: 2, custom: "kept" },
        "selectable",
        2,
        1
    )
    assert.deepEqual(selectable, {
        custom: "kept",
        selection: { order: 2, getNum: 1 },
    })

    const ordinary = normalizeBoxItemAwardModePayload(selectable, "award", 0, 1)
    assert.deepEqual(ordinary, { custom: "kept" })
})

test("selectable mode resolves contiguous order and persisted order", () => {
    const rows = [
        { id: 30, payload: { selection: { order: 2, getNum: 1 } } },
        { id: 10, payload: { selection: { order: 0, getNum: 1 } } },
        { id: 20, payload: { selection: { order: 1, getNum: 1 } } },
    ]
    assert.deepEqual(resolveBoxItemAwardMode(rows), {
        mode: "selectable",
        selectionCount: 1,
    })
    assert.deepEqual(
        orderBoxItemAwardRows(rows).map(row => row.id),
        [10, 20, 30]
    )
})

test("malformed, mixed, duplicate and inconsistent selection metadata fails", () => {
    for (const rows of [
        [{ payload: { selection: "invalid" } }],
        [{ payload: { selection: { order: 0.5, getNum: 1 } } }],
        [{ payload: { selection: { order: "0", getNum: 1 } } }],
        [{ payload: { selection: { order: 0, getNum: 1 } } }, { payload: {} }],
        [
            { payload: { selection: { order: 0, getNum: 1 } } },
            { payload: { selection: { order: 0, getNum: 1 } } },
        ],
        [
            { payload: { selection: { order: 0, getNum: 1 } } },
            { payload: { selection: { order: 1, getNum: 2 } } },
        ],
    ]) {
        assert.throws(() => resolveBoxItemAwardMode(rows))
    }
})

test("save envelope is strict and bounds selectable configuration", () => {
    const valid = {
        itemId: 3376,
        awards: [award],
        mode: "selectable",
        selectionCount: 1,
        revision: "a".repeat(32),
    }
    assert.equal(boxItemAwardSaveSchema.safeParse(valid).success, true)
    assert.equal(
        boxItemAwardSaveSchema.safeParse({
            ...valid,
            awards: [],
            mode: "award",
        }).success,
        true
    )
    for (const invalid of [
        { ...valid, awards: [] },
        { ...valid, selectionCount: 0 },
        { ...valid, revision: "" },
        { ...valid, extra: true },
        { ...valid, selectionCount: 2 },
        { ...valid, awards: [{ ...award, type: 32 }] },
        { ...valid, mode: "award", awards: [{ ...award, type: 999 }] },
        { ...valid, awards: [{ ...award, type: "19" }] },
        { ...valid, awards: [{ ...award, unknown: 1 }] },
    ]) {
        assert.equal(boxItemAwardSaveSchema.safeParse(invalid).success, false)
    }
})
