import { z } from "zod"

export const questIdSchema = z.number().int().min(1).max(2147483647)
const numeric = z.string().regex(/^(0|[1-9][0-9]{0,14})$/, "Số nguyên không âm, tối đa 15 chữ số")
const positive = numeric.refine(value => BigInt(value) > 0n, "Phải lớn hơn 0")
const quality = z.string().regex(/^(-1|0|[1-9][0-9]{0,3})$/)
const text = z.string().max(16000).nullable()
const npc = z.union([z.literal("-1"), numeric])
const nonemptyPatch = <T extends z.ZodRawShape>(shape: T) => z.object(shape).partial().strict().refine(value => Object.keys(value).length > 0, "Không có trường thay đổi")
export const questRootPatchSchema = nonemptyPatch({
    name: z.string().min(1).max(300), info: text, start_text: text, complete_text: text,
    type: z.enum(["1","2","3","4","5","6","7","8","9","10","11","12"]), sub_type: z.string().max(100).nullable(),
    at: z.literal("1"), award_exp_re: numeric, color: z.enum(["0","1","2","3","4"]),
    start_npc: npc, finish_npc: npc, gender: z.enum(["0","1","2"]), is_rebirth: z.enum(["0","1"]).nullable(),
    lm: z.union([z.literal("-1"), numeric]), max_level: numeric, min_level: numeric,
    money_type: z.enum(["-1","0","1","2","3","4"]), money_num: numeric,
    pre_quest_type: z.enum(["0","1","2"]), req_class: z.string().regex(/^(all|\|[1-9][0-9]*(\|[1-9][0-9]*)*\|)$/).nullable(),
    rt: z.literal("0"), rn: z.literal("0"), start_grant_item: questIdSchema.nullable(), start_grant_qty: questIdSchema.nullable(),
})
const requirementShape = { item_id: positive, kind: z.enum(["1","2","3"]), num: positive, q: quality, type: z.enum(["12","19","29"]) }
const awardShape = { ...requirementShape, kind: z.enum(["1","2"]), b: z.literal("1") }
const preShape = { item_id: positive, kind: z.enum(["1","2"]), num: positive, type: z.enum(["29","45"]) }
function pair(row: {kind?: string; type?: string}, group: string): boolean {
    if (!row.kind || !row.type) return true
    return group === "pre" ? (row.kind === "1" ? row.type === "29" : row.type === "45")
        : row.kind === "1" ? ["19","29"].includes(row.type) : row.type === "12"
}
function delta(shape: z.ZodRawShape, group: string) {
    const check = (row: Record<string, unknown>) => pair(row, group) && !(group === "award" && row.type === "12" && row.num !== undefined && row.num !== "1")
    const row = z.object(shape).strict().refine(check, "Cặp loại/bảng hoặc số lượng thú không hỗ trợ")
    const changes = nonemptyPatch(shape).refine(check, "Cặp loại/bảng hoặc số lượng thú không hỗ trợ")
    return z.object({ insert: z.array(row).max(100).optional(), update: z.array(z.object({id: questIdSchema, changes}).strict()).max(100).optional(), deleteIds: z.array(questIdSchema).max(100).optional() }).strict().superRefine((value, ctx) => {
        const ids = [...(value.update ?? []).map(row => row.id), ...(value.deleteIds ?? [])]
        if (new Set(ids).size !== ids.length) ctx.addIssue({code:"custom",message:"ID hàng trùng lặp"})
    })
}
const envelope = { expectedRevision: z.string().regex(/^[a-f0-9]{32}$/), reason: z.string().trim().min(3).max(1000), requestId: z.string().uuid() }
export const questSaveSchema = z.object({ ...envelope, patch: z.object({ quest: questRootPatchSchema.optional(), requirements: delta(requirementShape,"require").optional(), awards: delta(awardShape,"award").optional(), prerequisites: delta(preShape,"pre").optional() }).strict().refine(value => Object.keys(value).length > 0) }).strict()
export const questLoopSaveSchema = z.object({ ...envelope, expectedAffectedCount: z.number().int().nonnegative(), confirmShared: z.literal(true), patch: nonemptyPatch({name: z.string().min(1).max(300), info: text, nid: npc, num: positive, min_level: numeric, max_level: numeric, refresh: positive, team: z.enum(["0","1","2"])}) }).strict()
export const questLookupKindSchema = z.enum(["npc","monster","item","equipment","pet","quest","loop","class"])
export const questListQuerySchema = z.object({search:z.string().max(200).default(""),id:z.coerce.number().int().positive().max(2147483647).optional(),type:z.coerce.number().int().min(1).max(12).optional(),minLevel:z.coerce.number().int().nonnegative().max(10000).optional(),maxLevel:z.coerce.number().int().nonnegative().max(10000).optional(),page:z.coerce.number().int().min(1).max(100000).default(1),pageSize:z.coerce.number().int().min(1).max(100).default(25)}).strict()
