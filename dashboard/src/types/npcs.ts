export interface NpcRecord {
    id: number
    name: string | null
    type: number
    pos_map_id: number | null
    pos_x: number
    pos_y: number
    on_service_text: string | null
    shop_id: number | null
    qid: string | null
    offered_count?: number
    turn_in_count?: number
    loop_count?: number
    [key: string]: unknown
}

export interface NpcQuestLink {
    id: number
    name: string | null
    type: number
    min_level: number
    max_level: number
    start_npc: number
    finish_npc: number
    start_text: string | null
    complete_text: string | null
}

export interface NpcLoopLink {
    id: number
    name: string | null
    nid: number
    min_level: number
    max_level: number
}

export interface ShopContainer {
    id: number
    name: string | null
    type: number
}

export interface ShopSlotRecord {
    id: number
    amount: number
    gold: number
    gt: number
    item_id: number
    money: number
    p_num1: number
    p_num2: number
    p_type1: number
    p_type2: number
    point: number
    position: number
    price_all: number | null
    quality: number
    r: number
    sale: number
    sid: number
    st: number
    type: 19 | 29
    displayName: string | null
    iconCode: number
}

export interface NpcShopDetail {
    container: ShopContainer
    slots: ShopSlotRecord[]
    referencingNpcs: { id: number; name: string | null }[]
}

export interface NpcActivation {
    databaseSaved: boolean
    serverRestartRequired: true
    clientReconnectRequired: true
    sceneReentryRequired: true
    questCatalogBlocked: true
}

export interface NpcDetail {
    npcId: number
    npc: NpcRecord
    offeredQuests: NpcQuestLink[]
    turnInQuests: NpcQuestLink[]
    loops: NpcLoopLink[]
    legacyQid: { raw: string | null; ids: number[]; missingIds: number[] }
    shop: NpcShopDetail | null
    revision: string
    warnings: string[]
    canWrite: boolean
    activation: NpcActivation
}

export interface NpcListResponse {
    items: NpcRecord[]
    total: number
    page: number
    pageSize: number
    canWrite: boolean
}

export interface NpcLookupResponse {
    items: NpcLookupItem[]
    total: number
    page: number
    pageSize: number
}

export interface NpcLookupItem {
    id: number
    name: string | null
    table: string
    iconCode: number
}

export type ShopSlotInsert = Omit<
    ShopSlotRecord,
    "id" | "sid" | "displayName" | "iconCode"
>
export interface NpcSavePatch {
    npc?: Partial<Pick<NpcRecord, "name" | "on_service_text" | "shop_id">>
    slots?: {
        insert?: ShopSlotInsert[]
        update?: { id: number; changes: Partial<ShopSlotInsert> }[]
        hideIds?: number[]
    }
}
