import type {
    Collection,
    Detail,
    DraftRow,
    RecordRow,
    Value,
} from "../_lib/editor"

export type QuestFormProps = {
    quest: RecordRow
    baseline: Detail
    rows: Record<Collection, DraftRow[]>
    edit: (field: string, value: Value) => void
    replaceQuest: (quest: RecordRow) => void
    replaceRows: (collection: Collection, rows: DraftRow[]) => void
    disabled: boolean
    structuralDisabled: boolean
    selectedLoop: string
    setSelectedLoop: (value: string) => void
}
