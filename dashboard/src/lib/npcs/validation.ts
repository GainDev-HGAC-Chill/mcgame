import { z } from "zod"

const optionalInt = z.preprocess(
    value => (value === "" || value === undefined ? undefined : value),
    z.coerce.number().int().min(0).max(2147483647).optional()
)
const optionalBool = z.preprocess(
    value =>
        value === "" || value === undefined
            ? undefined
            : value === "true"
              ? true
              : value === "false"
                ? false
                : value,
    z.boolean().optional()
)

export const npcListQuerySchema = z
    .object({
        search: z.string().max(200).default(""),
        mapId: optionalInt,
        type: optionalInt,
        hasDialogue: optionalBool,
        hasShop: optionalBool,
        page: z.coerce.number().int().min(1).max(100000).default(1),
        pageSize: z.coerce.number().int().min(1).max(100).default(25),
    })
    .strict()

export const npcLookupQuerySchema = z
    .object({
        kind: z.enum(["shop", "item", "equipment", "map"]),
        search: z.string().max(200).default(""),
        page: z.coerce.number().int().min(1).max(100000).default(1),
        pageSize: z.coerce.number().int().min(1).max(100).default(25),
    })
    .strict()

const integer = z.number().int().min(0).max(2147483647)
const slotShape = {
    amount: z.number().int().min(-1).max(2147483647),
    gold: integer,
    gt: z.union([z.literal(0), z.literal(1)]),
    item_id: integer.min(1),
    money: integer,
    p_num1: integer,
    p_num2: integer,
    p_type1: integer,
    p_type2: integer,
    point: z.literal(0),
    position: integer.min(1),
    price_all: integer.nullable(),
    quality: z.number().int().min(-1).max(2147483647),
    r: integer,
    sale: integer,
    st: z.union([z.literal(0), z.literal(5)]),
    type: z.union([z.literal(19), z.literal(29)]),
}
const slotInsertSchema = z.object(slotShape).strict()
const slotUpdateSchema = z
    .object({
        id: integer.min(1),
        changes: z
            .object({ ...slotShape, position: z.never().optional() })
            .partial()
            .strict()
            .refine(
                value => Object.keys(value).length > 0,
                "Không có thay đổi"
            ),
    })
    .strict()

export const npcSaveSchema = z
    .object({
        expectedRevision: z.string().regex(/^[a-f0-9]{32}$/),
        expectedSharedCount: integer,
        confirmShared: z.boolean(),
        reason: z.string().trim().min(3).max(1000),
        requestId: z.string().uuid(),
        patch: z
            .object({
                npc: z
                    .object({
                        name: z.string().trim().min(1).max(300).optional(),
                        on_service_text: z
                            .string()
                            .max(16000)
                            .nullable()
                            .optional(),
                        shop_id: integer.nullable().optional(),
                    })
                    .strict()
                    .refine(
                        value => Object.keys(value).length > 0,
                        "Không có thay đổi NPC"
                    )
                    .optional(),
                slots: z
                    .object({
                        insert: z.array(slotInsertSchema).max(100).optional(),
                        update: z.array(slotUpdateSchema).max(100).optional(),
                        hideIds: z.array(integer.min(1)).max(100).optional(),
                    })
                    .strict()
                    .refine(
                        value => Object.keys(value).length > 0,
                        "Không có thay đổi slot"
                    )
                    .optional(),
            })
            .strict()
            .refine(
                value => Object.keys(value).length > 0,
                "Không có thay đổi"
            ),
    })
    .strict()
