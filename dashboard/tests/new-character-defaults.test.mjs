import assert from "node:assert/strict"
import { test } from "node:test"
import {
    INITIAL_CHARACTER_DEFAULTS,
    parseNewCharacterDefaults,
} from "../src/lib/new-character-defaults.ts"

const base = () => structuredClone(INITIAL_CHARACTER_DEFAULTS)

test("keeps the approved existing defaults without adding gifts", () => {
    assert.deepEqual(parseNewCharacterDefaults(base()), base())
})

test("accepts explicit item and pet gift fields", () => {
    const config = base()
    config.items.push({
        templateTableId: 19,
        templateId: 758,
        quantity: 1,
        bound: true,
        color: 3,
    })
    config.pets.push({ templateId: 91, quantity: 1, bound: false, growth: 1.9 })
    assert.deepEqual(parseNewCharacterDefaults(config), config)
})

test("rejects unknown fields, fractional levels, wrong types and unsafe wallets", () => {
    for (const patch of [
        { unexpected: true },
        { level: 1.5 },
        { level: 161 },
        { gold: "100" },
        { goldBind: -1 },
        { gold: 2 ** 53 },
        { items: null },
        { version: 2 },
    ]) {
        assert.throws(() => parseNewCharacterDefaults({ ...base(), ...patch }))
    }
})

test("rejects invalid gifts and pet overflow", () => {
    const item = {
        templateTableId: 29,
        templateId: 1,
        quantity: 1,
        bound: true,
        color: 0,
    }
    for (const patch of [
        { quantity: 0 },
        { quantity: 1.5 },
        { templateId: -1 },
        { bound: "true" },
        { color: 6 },
        { templateTableId: 12 },
        { injected: true },
    ]) {
        assert.throws(() =>
            parseNewCharacterDefaults({
                ...base(),
                items: [{ ...item, ...patch }],
            })
        )
    }
    const pet = { templateId: 91, quantity: 1, bound: true, growth: 1.9 }
    for (const patch of [
        { growth: 0 },
        { growth: NaN },
        { growth: Infinity },
        { quantity: 7 },
    ]) {
        assert.throws(() =>
            parseNewCharacterDefaults({
                ...base(),
                pets: [{ ...pet, ...patch }],
            })
        )
    }
    assert.throws(() =>
        parseNewCharacterDefaults({
            ...base(),
            pets: [
                { ...pet, quantity: 4 },
                { ...pet, quantity: 3 },
            ],
        })
    )
})

test("checks ninety-entry format limit separately from catalog capacity", () => {
    const item = { templateTableId: 29, templateId: 1, quantity: 1000000, bound: true, color: 0 }
    const config = { ...base(), items: Array.from({ length: 90 }, () => ({ ...item })) }
    assert.equal(parseNewCharacterDefaults(config).items.length, 90)
    assert.throws(() => parseNewCharacterDefaults({ ...base(), items: [...config.items, item] }))
    assert.throws(() => parseNewCharacterDefaults({ ...base(), items: [{ ...item, quantity: 1000001 }] }))
})
