import "server-only"

import { parseNewCharacterDefaults } from "@/lib/new-character-defaults"
import { createAdminClient } from "@/lib/supabase/admin"

export async function validateStarterCatalog(value: unknown) {
    const config = parseNewCharacterDefaults(value)
    const client = createAdminClient()
    const counts = new Map<
        string,
        { quantity: number; stack: number; slot: string }
    >()
    for (const tableId of [19, 29] as const) {
        const entries = config.items.filter(
            item => item.templateTableId === tableId
        )
        if (!entries.length) continue
        const ids = [...new Set(entries.map(item => item.templateId))]
        const { data, error } = await client
            .schema("data")
            .from(
                tableId === 19
                    ? "data_tbl_equipt_template"
                    : "data_tbl_item_template"
            )
            .select("id,kind,type,bind_type,stack_max,single_flag")
            .in("id", ids)
        if (error)
            throw new Error(
                "Không kiểm tra được danh mục vật phẩm; cấu hình chưa được lưu."
            )
        const found = new Set((data || []).map(row => Number(row.id)))
        for (const id of ids)
            if (!found.has(id))
                throw new Error(
                    `Không có ${tableId === 19 ? "trang bị" : "vật phẩm"} mã ${id} trong danh mục.`
                )
        const catalog = new Map((data || []).map(row => [Number(row.id), row]))
        for (const entry of entries) {
            const row = catalog.get(entry.templateId)!
            if (Number(row.bind_type) > 0 && !entry.bound)
                throw new Error(
                    `Quà mã ${entry.templateId} bắt buộc khóa theo danh mục.`
                )
            if (
                tableId === 29 &&
                ((Number(row.kind) === 6 && entry.color > 4) ||
                    (Number(row.kind) !== 6 && entry.color !== 0))
            ) {
                throw new Error(
                    `Phẩm chất của vật phẩm ${entry.templateId} không hợp lệ: nguyên liệu dùng 0–4, vật phẩm khác dùng 0.`
                )
            }
            const rawStack = Number(row.stack_max)
            if (!Number.isSafeInteger(rawStack) || rawStack > 2147483647)
                throw new Error(
                    `Số lượng xếp chồng trong danh mục của quà ${entry.templateId} không hợp lệ.`
                )
            const stack =
                tableId === 19 || Number(row.single_flag) > 0
                    ? 1
                    : Math.max(1, rawStack)
            const type = Number(row.type)
            const slot =
                tableId === 19
                    ? Number(row.kind) === 9
                        ? "thú cưng"
                        : "thường"
                    : [508, 509, 550].includes(type)
                      ? "nhiệm vụ"
                      : [507, 512, 513].includes(type)
                        ? "thú cưng"
                        : "thường"
            const key = `${tableId}:${entry.templateId}:${entry.bound}:${entry.color}:${slot}`
            const previous = counts.get(key)
            counts.set(key, {
                quantity: (previous?.quantity || 0) + entry.quantity,
                stack,
                slot,
            })
        }
    }
    if (config.pets.length) {
        const ids = [...new Set(config.pets.map(pet => pet.templateId))]
        const { data, error } = await client
            .schema("data")
            .from("data_tbl_creature")
            .select("id,class_id,is_bind")
            .in("id", ids)
        if (error)
            throw new Error(
                "Không kiểm tra được danh mục thú cưng; cấu hình chưa được lưu."
            )
        const found = new Set((data || []).map(row => Number(row.id)))
        for (const id of ids)
            if (!found.has(id))
                throw new Error(`Không có thú cưng mã ${id} trong danh mục.`)
        const catalog = new Map((data || []).map(row => [Number(row.id), row]))
        for (const pet of config.pets) {
            const row = catalog.get(pet.templateId)!
            if (Number(row.class_id) !== 2)
                throw new Error(
                    `Mã ${pet.templateId} không phải thú cưng có thể sở hữu.`
                )
            if (Number(row.is_bind) > 0 && !pet.bound)
                throw new Error(
                    `Thú cưng ${pet.templateId} bắt buộc khóa theo danh mục.`
                )
        }
    }
    const slots = new Map<string, number>()
    for (const entry of counts.values())
        slots.set(
            entry.slot,
            (slots.get(entry.slot) || 0) +
                Math.ceil(entry.quantity / entry.stack)
        )
    for (const [slot, total] of slots)
        if (total > 30)
            throw new Error(
                `Gói quà cần ${total} ô túi ${slot}, vượt 30 ô ban đầu.`
            )
    return config
}
