import {
    Backpack,
    Boxes,
    Coins,
    Leaf,
    PawPrint,
    Package,
    Shield,
    Sparkles,
    Swords,
    Target,
    User,
    UsersRound,
    type LucideIcon,
} from "lucide-react"

type PlayerNavCategory =
    | "Core"
    | "Combat"
    | "Economy"
    | "Inventory"
    | "Customization"
    | "Systems"
    | "Social"

export type PlayerNavItem = {
    value: string
    label: string
    icon: LucideIcon
    category: PlayerNavCategory
}
export const PLAYER_NAV_CATEGORY_LABELS: Record<PlayerNavCategory, string> = {
    Core: "Thông tin cơ bản", Combat: "Chiến đấu", Economy: "Kinh tế",
    Inventory: "Kho đồ", Customization: "Tùy chỉnh", Systems: "Hệ thống", Social: "Quan hệ",
}

export const PLAYER_NAV_ITEMS: PlayerNavItem[] = [
    { value: "overview", label: "Tổng quan", icon: User, category: "Core" },
    {
        value: "progression",
        label: "Tiến trình",
        icon: Sparkles,
        category: "Core",
    },
    {
        value: "attributes",
        label: "Thuộc tính",
        icon: Target,
        category: "Core",
    },
    {
        value: "combat",
        label: "Chỉ số chiến đấu",
        icon: Swords,
        category: "Combat",
    },
    { value: "wallet", label: "Tiền tệ", icon: Coins, category: "Economy" },
    {
        value: "resources",
        label: "Sức chứa",
        icon: Backpack,
        category: "Economy",
    },
    {
        value: "items",
        label: "Vật phẩm",
        icon: Package,
        category: "Inventory",
    },
    {
        value: "pets",
        label: "Thú cưng",
        icon: PawPrint,
        category: "Inventory",
    },
    {
        value: "dress-panel",
        label: "Thời trang",
        icon: Sparkles,
        category: "Customization",
    },
    {
        value: "life-skills",
        label: "Kỹ năng sống",
        icon: Leaf,
        category: "Customization",
    },
    { value: "gm", label: "GM", icon: Shield, category: "Systems" },
    {
        value: "feature-state",
        label: "Hệ thống trò chơi",
        icon: Boxes,
        category: "Systems",
    },
    {
        value: "relationships",
        label: "Quan hệ",
        icon: UsersRound,
        category: "Social",
    },
]

export const PLAYER_NAV_CATEGORIES: PlayerNavCategory[] = [
    ...new Set(PLAYER_NAV_ITEMS.map(item => item.category)),
]
