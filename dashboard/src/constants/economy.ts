/**
 * Economy constants
 */

export const TRANSACTION_TYPES = {
    purchase: { label: "Mua", color: "bg-blue-500" },
    trade: { label: "Giao dịch", color: "bg-purple-500" },
    gift: { label: "Quà tặng", color: "bg-green-500" },
    sell: { label: "Bán", color: "bg-yellow-500" },
    compensate: { label: "Đền bù", color: "bg-pink-500" },
}

export const ALERT_TYPES = {
    critical: { label: "Nghiêm trọng", color: "text-red-500", bg: "bg-red-500/10" },
    warning: {
        label: "Cảnh báo",
        color: "text-yellow-500",
        bg: "bg-yellow-500/10",
    },
    info: { label: "Thông tin", color: "text-blue-500", bg: "bg-blue-500/10" },
}

export const CURRENCY_TYPES = {
    gold: { label: "Vàng", color: "bg-yellow-500", icon: "coins" },
    gems: { label: "Đá quý", color: "bg-purple-500", icon: "gem" },
    coins: { label: "Xu", color: "bg-blue-500", icon: "coins" },
}
