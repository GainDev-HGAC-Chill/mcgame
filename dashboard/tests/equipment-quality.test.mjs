import assert from "node:assert/strict"
import { test } from "node:test"

import {
    EQUIPMENT_QUALITY_OPTIONS,
    equipmentQualityLabel,
} from "../src/lib/equipment-quality.ts"
import { equipmentQualitySchema } from "../src/lib/equipment-quality-validation.ts"

test("equipment quality controls expose the complete 0 through 25 contract", () => {
    assert.deepEqual(
        EQUIPMENT_QUALITY_OPTIONS.map(option => option.quality),
        Array.from({ length: 26 }, (_, quality) => quality)
    )
    assert.equal(equipmentQualityLabel(0), "0 — Trắng, không tiền tố")
    assert.match(equipmentQualityLabel(25), /Cam/)
})

test("equipment quality validation accepts only integers from 0 through 25", () => {
    for (const quality of [0, 1, 25]) {
        assert.equal(equipmentQualitySchema.safeParse(quality).success, true)
    }
    for (const quality of [-1, 1.5, 26]) {
        assert.equal(equipmentQualitySchema.safeParse(quality).success, false)
    }
})
