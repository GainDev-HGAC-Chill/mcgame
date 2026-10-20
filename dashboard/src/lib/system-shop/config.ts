/** Existing TBL_SHOP containers used by SystemShopPanel.as (not NPC shops). */
export const SYSTEM_SHOPS = [
    { sid: 67, label: "Pet" },
    { sid: 65, label: "Vật phẩm quý" },
    { sid: 66, label: "Nguyên liệu" },
    { sid: 36, label: "Bảo thạch" },
    { sid: 35, label: "Cuộn sách" },
    { sid: 81, label: "Sách" },
    { sid: 70, label: "Điểm" },
    { sid: 88, label: "Thời trang" },
    { sid: 89, label: "Phi hành" },
    { sid: 41, label: "Thực phẩm" },
] as const

export type SystemShopId = (typeof SYSTEM_SHOPS)[number]["sid"]
export const SYSTEM_SHOP_IDS = SYSTEM_SHOPS.map(row => row.sid)

export const SLOT_STATES = [
    { id: 0, label: "Thường" },
    { id: 1, label: "Hot" },
    { id: 2, label: "Discount" },
    { id: 4, label: "New" },
    { id: 5, label: "Ẩn" },
] as const

export type ShopSlotForm = {
    type: 19 | 29
    item_id: number
    gold: number
    money: number
    point: number
    gt: 0 | 1
    quality: number
    amount: number
    st: 0 | 1 | 2 | 4 | 5
    position: number
}

export interface ShopSlotRow extends ShopSlotForm {
    id: number
    sid: number
    p_num1: number
    p_num2: number
    p_type1: number
    p_type2: number
    price_all: number | null
    r: number
    sale: number
    displayName: string | null
    iconCode: number | null
}

export interface SystemShopDetail {
    shop: { id: number; name: string; type: number }
    slots: ShopSlotRow[]
    revision: string
    canWrite: boolean
    changedSlotId?: number
}

export function emptyShopSlot(slots: ShopSlotRow[] = []): ShopSlotForm {
    return {
        type: 29,
        item_id: 1,
        gold: 600,
        money: 0,
        point: 0,
        gt: 0,
        quality: 0,
        amount: -1,
        st: 4,
        position: Math.max(0, ...slots.map(row => row.position)) + 1,
    }
}
