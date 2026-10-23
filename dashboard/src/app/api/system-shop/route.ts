import { z } from "zod"

import { SYSTEM_SHOP_IDS } from "@/lib/system-shop/config"
import { equipmentQualitySchema } from "@/lib/equipment-quality-validation"
import {
    npcRoute,
    npcRpc,
    readNpcBody,
    NpcApiError,
} from "@/lib/server/npc-admin"
import type { SystemShopDetail } from "@/lib/system-shop/config"

export const dynamic = "force-dynamic"

const shopID = z
    .number()
    .int()
    .refine(id => SYSTEM_SHOP_IDS.some(known => known === id))
const nonnegative = z.number().int().min(0).max(2147483647)
const state = z.union([
    z.literal(0),
    z.literal(1),
    z.literal(2),
    z.literal(4),
    z.literal(5),
])
const slot = z
    .object({
        type: z.union([z.literal(19), z.literal(29)]),
        item_id: nonnegative.min(1),
        gold: nonnegative,
        money: nonnegative,
        point: nonnegative,
        gt: z.union([z.literal(0), z.literal(1)]),
        quality: equipmentQualitySchema,
        amount: z
            .number()
            .int()
            .min(-1)
            .max(2147483647)
            .refine(v => v !== 0),
        st: state,
        position: nonnegative.min(1),
    })
    .strict()
    .refine(
        v =>
            Number(v.gold > 0) + Number(v.money > 0) + Number(v.point > 0) ===
            1,
        "Chỉ được chọn một loại giá dương (vàng, bạc hoặc điểm)."
    )
const saveBody = z
    .object({
        sid: shopID,
        expectedRevision: z.string().regex(/^[a-f0-9]{32}$/),
        action: z.enum(["insert", "update"]),
        slotId: z.number().int().positive().nullable(),
        slot,
        reason: z.string().trim().min(3).max(1000),
        requestId: z.string().uuid(),
    })
    .strict()
    .superRefine((v, ctx) => {
        if ((v.action === "insert") !== (v.slotId === null)) {
            ctx.addIssue({
                code: "custom",
                message: "Slot ID và thao tác không khớp",
                path: ["slotId"],
            })
        }
    })

export async function GET(request: Request) {
    return npcRoute(request, "read", async auth => {
        const raw = new URL(request.url).searchParams.get("sid") ?? "65"
        if (!/^[1-9]\d{0,3}$/.test(raw))
            throw new NpcApiError(422, "system_shop_id", "Shop ID không hợp lệ")
        const sid = shopID.parse(Number(raw))
        const detail = await npcRpc<SystemShopDetail>(
            "system_shop_admin_detail",
            { p_sid: sid }
        )
        return { ...detail, canWrite: auth.canWrite }
    })
}

export async function POST(request: Request) {
    return npcRoute(request, "write", async auth => {
        const data = saveBody.parse(await readNpcBody(request))
        const result = await npcRpc<SystemShopDetail>(
            "system_shop_admin_save",
            {
                p_sid: data.sid,
                p_expected_revision: data.expectedRevision,
                p_action: data.action,
                p_slot_id: data.slotId,
                p_slot: data.slot,
                p_reason: data.reason,
                p_request_id: data.requestId,
                p_actor_id: auth.userId,
            }
        )
        return { ...result, canWrite: auth.canWrite }
    })
}
