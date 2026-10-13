export type QuestNumeric = string | null
export interface QuestRecord {
    id: number
    a_script_text: string | null; at: QuestNumeric; award_exp_re: QuestNumeric; color: QuestNumeric
    complete_text: string | null; finish_npc: QuestNumeric; gender: QuestNumeric; info: string | null
    is_rebirth: QuestNumeric; lm: QuestNumeric; max_level: QuestNumeric; min_level: QuestNumeric
    money_num: QuestNumeric; money_type: QuestNumeric; name: string | null; pre_quest_type: QuestNumeric
    qtest: string | null; req_class: string | null; rn: QuestNumeric; rt: QuestNumeric
    script_give_up: string | null; start_npc: QuestNumeric; start_text: string | null
    sub_type: string | null; type: QuestNumeric; start_grant_item: number | null; start_grant_qty: number | null
}
export interface QuestRequirement { id: number; item_id: QuestNumeric; kind: QuestNumeric; num: QuestNumeric; q: QuestNumeric; qid: QuestNumeric; type: QuestNumeric }
export interface QuestAward extends QuestRequirement { b: QuestNumeric }
export interface QuestPrerequisite { id: number; item_id: QuestNumeric; kind: QuestNumeric; num: QuestNumeric; qid: QuestNumeric; type: QuestNumeric }
export interface QuestLoop { id: number; info: string | null; max_level: QuestNumeric; min_level: QuestNumeric; name: string | null; nid: QuestNumeric; num: QuestNumeric; refresh: QuestNumeric; script_get: string | null; script_give_up: string | null; team: QuestNumeric }
export interface QuestActivation { databaseSaved: boolean; serverReloadRequired: true; clientRefreshRequired: true; mode: "database-only"; flashCatalogBlocked: true; clientCatalogPublicationRequired: true }
export interface QuestLoopDetail { loop: QuestLoop; revision: string; affectedQuestCount: number; activeUse: { activeLoops: number; activeQuests: number }; warnings: string[]; canWrite: boolean; activation: QuestActivation }
export interface QuestDetail {
    questId: number; revision: string; quest: QuestRecord; requirements: QuestRequirement[]; awards: QuestAward[]
    prerequisites: QuestPrerequisite[]; loopRef: QuestLoopDetail | null; warnings: string[]
    activeUse: { activeQuests: number; completedHistory: number; activeLoops: number }
    canWrite: boolean; activation: QuestActivation
}
export interface QuestListResponse { items: QuestRecord[]; total: number; page: number; pageSize: number; canWrite: boolean }
export type QuestLookupKind = "npc" | "monster" | "item" | "equipment" | "pet" | "quest" | "loop" | "class"
export interface QuestLookupResponse { items: { id: number; name: string | null; table: string }[]; total: number; page: number; pageSize: number }
export type QuestRootPatch = Partial<Omit<QuestRecord, "id" | "a_script_text" | "qtest" | "script_give_up">>
export interface QuestChildDelta<T> { insert?: Omit<T, "id" | "qid">[]; update?: { id: number; changes: Partial<Omit<T, "id" | "qid">> }[]; deleteIds?: number[] }
export interface QuestSaveRequest { expectedRevision: string; reason: string; requestId: string; patch: { quest?: QuestRootPatch; requirements?: QuestChildDelta<QuestRequirement>; awards?: QuestChildDelta<QuestAward>; prerequisites?: QuestChildDelta<QuestPrerequisite> } }
export interface QuestLoopSaveRequest { expectedRevision: string; expectedAffectedCount: number; confirmShared: true; reason: string; requestId: string; patch: Partial<Omit<QuestLoop, "id" | "script_get" | "script_give_up">> }
