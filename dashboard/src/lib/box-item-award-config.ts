import { z } from "zod"

import { BOX_ITEM_GAME_CURRENCY_IDS } from "./box-item-awards.ts"

const supportedAwardTypes = new Set([
    12, 19, 29, 30, 31, 32, 33, 34, 35, 37, 38,
])

export const boxItemAwardInputSchema = z
    .object({
        id: z.number().int().positive().optional(),
        awardId: z.number().int().min(0),
        type: z
            .number()
            .int()
            .refine(value => supportedAwardTypes.has(value)),
        count: z.number().int().min(1).max(100000000),
        rate: z.number().int().min(0).max(100000000),
        quality: z.number().int().min(0).max(25),
        preNameType: z.number().int().min(0).max(5),
        payload: z.record(z.string(), z.unknown()),
    })
    .strict()

export const boxItemAwardSaveSchema = z
    .object({
        itemId: z.number().int().positive(),
        awards: z.array(boxItemAwardInputSchema).max(200),
        mode: z.enum(["award", "selectable"]),
        selectionCount: z.number().int().min(1).max(200),
        revision: z.string().regex(/^[a-f0-9]{32}$/i),
    })
    .strict()
    .superRefine((value, context) => {
        if (
            value.mode === "selectable" &&
            (value.awards.length === 0 ||
                value.selectionCount > value.awards.length)
        ) {
            context.addIssue({
                code: "custom",
                path: ["selectionCount"],
                message: "Số lượt chọn vượt quá số phần thưởng.",
            })
        }
        if (value.mode === "selectable") {
            value.awards.forEach((award, index) => {
                if (
                    award.type !== 12 &&
                    award.type !== 19 &&
                    award.type !== 29
                ) {
                    context.addIssue({
                        code: "custom",
                        path: ["awards", index, "type"],
                        message: "Loại phần thưởng tự chọn không được hỗ trợ.",
                    })
                }
            })
        }
        value.awards.forEach((award, index) => {
            if (
                award.type === 35 &&
                !BOX_ITEM_GAME_CURRENCY_IDS.has(award.awardId)
            ) {
                context.addIssue({
                    code: "custom",
                    path: ["awards", index, "awardId"],
                    message: "Loại tiền tệ game không được hỗ trợ.",
                })
            }
        })
    })
