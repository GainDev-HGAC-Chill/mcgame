export type Value = string | number | null
export type RecordRow = { id: number; [field: string]: Value }
export type DraftRow = {
    [field: string]: Value | undefined
    id?: number
    _key: string
}
export type Collection = "requirements" | "awards" | "prerequisites"
export type Detail = {
    questId: number
    revision: string
    quest: RecordRow
    requirements: RecordRow[]
    awards: RecordRow[]
    prerequisites: RecordRow[]
    canWrite: boolean
    warnings: string[]
    loopRef?: LoopDetail | null
    activeUse?: {
        activeQuests: number
        completedHistory: number
        activeLoops: number
    }
    activation?: {
        databaseSaved?: boolean
        serverReloadRequired?: boolean
        clientRefreshRequired?: boolean
    }
}
export type LoopDetail = {
    loop: RecordRow
    revision: string
    affectedQuestCount: number
    canWrite: boolean
    warnings: string[]
    activeUse?: { activeLoops?: number; activeQuests?: number }
    activeLoopCount?: number
}
export type Lookup = { id: number; name: string; [key: string]: unknown }
export const questTypes = [
    [1, "Hướng dẫn"],
    [2, "Chính tuyến"],
    [3, "Nghề nghiệp"],
    [4, "Thế giới"],
    [5, "Thương mại"],
    [6, "Hoạt động"],
    [7, "Vòng lặp"],
    [8, "Thử thách"],
    [9, "Bảng nhiệm vụ"],
    [10, "Trưởng thành"],
    [11, "Bang hội"],
    [12, "May mắn"],
] as const
export const text = (value: Value | undefined) =>
    value == null ? "" : String(value)
export function changedFields(
    before: Record<string, Value>,
    after: Record<string, Value>
) {
    return Object.fromEntries(
        Object.entries(after).filter(
            ([key, value]) =>
                key !== "id" &&
                key !== "qid" &&
                !key.startsWith("_") &&
                value !== before[key]
        )
    )
}
export function draftRows(rows: RecordRow[]): DraftRow[] {
    return rows.map(row => ({ ...row, _key: `existing-${row.id}` }))
}
export function rowPatch(before: RecordRow[], after: DraftRow[]) {
    const update = after
        .filter(row => row.id != null)
        .flatMap(row => {
            const original = before.find(old => old.id === row.id)
            if (!original) return []
            const fields = changedFields(original, row as Record<string, Value>)
            return Object.keys(fields).length
                ? [{ id: row.id, changes: fields }]
                : []
        })
    const insert = after
        .filter(row => row.id == null)
        .map(row =>
            Object.fromEntries(
                Object.entries(row).filter(
                    ([key]) =>
                        key !== "id" && key !== "qid" && !key.startsWith("_")
                )
            )
        )
    const deleteIds = before
        .filter(row => !after.some(current => current.id === row.id))
        .map(row => row.id)
    return { update, insert, deleteIds }
}
export function aggregatePatch(
    detail: Detail,
    quest: RecordRow,
    rows: Record<Collection, DraftRow[]>
) {
    const patch: Record<string, unknown> = {}
    const fields = changedFields(detail.quest, quest)
    if (Object.keys(fields).length) patch.quest = fields
    for (const collection of [
        "requirements",
        "awards",
        "prerequisites",
    ] as const) {
        const changes = rowPatch(detail[collection], rows[collection])
        if (
            changes.update.length ||
            changes.insert.length ||
            changes.deleteIds.length
        )
            patch[collection] = changes
    }
    return patch
}
export function classValues(value: Value | undefined) {
    return text(value).split("|").filter(Boolean)
}
export function serializeClasses(values: string[]) {
    return values.length ? `|${values.join("|")}|` : "all"
}
