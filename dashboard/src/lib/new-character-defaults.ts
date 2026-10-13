export const NEW_CHARACTER_DEFAULTS_KEY = "new_character_defaults"

export type StarterItem = {
    templateTableId: 19 | 29
    templateId: number
    quantity: number
    bound: boolean
    color: number
}

export type StarterPet = {
    templateId: number
    quantity: number
    bound: boolean
    growth: number
}

export type NewCharacterDefaults = {
    version: 1
    level: number
    gold: number
    goldBind: number
    items: StarterItem[]
    pets: StarterPet[]
}

export const INITIAL_CHARACTER_DEFAULTS: NewCharacterDefaults = {
    version: 1,
    level: 90,
    gold: 999999,
    goldBind: 999999,
    items: [],
    pets: [],
}

function object(value: unknown, fields: string[]): Record<string, unknown> {
    if (!value || typeof value !== "object" || Array.isArray(value))
        throw new Error("Cấu hình phải là đối tượng JSON.")
    const result = value as Record<string, unknown>
    if (Object.keys(result).some(key => !fields.includes(key)))
        throw new Error("Cấu hình chứa trường không được hỗ trợ.")
    return result
}

function integer(
    value: unknown,
    min: number,
    max: number,
    label: string
): number {
    if (
        typeof value !== "number" ||
        !Number.isSafeInteger(value) ||
        value < min ||
        value > max
    ) {
        throw new Error(`${label} phải là số nguyên từ ${min} đến ${max}.`)
    }
    return value
}

function bound(value: unknown): boolean {
    if (typeof value !== "boolean")
        throw new Error("Trạng thái khóa phải là giá trị đúng/sai.")
    return value
}

export function parseNewCharacterDefaults(
    value: unknown
): NewCharacterDefaults {
    const data = object(value, [
        "version",
        "level",
        "gold",
        "goldBind",
        "items",
        "pets",
    ])
    if (data.version !== 1)
        throw new Error("Phiên bản cấu hình không được hỗ trợ.")
    if (!Array.isArray(data.items) || !Array.isArray(data.pets))
        throw new Error("Danh sách vật phẩm và thú cưng không hợp lệ.")
    if (data.items.length > 90 || data.pets.length > 6)
        throw new Error("Gói quà vượt số mục cho phép của túi ban đầu.")
    const items = data.items.map(raw => {
        const item = object(raw, [
            "templateTableId",
            "templateId",
            "quantity",
            "bound",
            "color",
        ])
        if (item.templateTableId !== 19 && item.templateTableId !== 29)
            throw new Error("Chỉ hỗ trợ vật phẩm và trang bị.")
        return {
            templateTableId: item.templateTableId,
            templateId: integer(item.templateId, 1, 2147483647, "Mã vật phẩm"),
            quantity: integer(item.quantity, 1, 1000000, "Số lượng vật phẩm"),
            bound: bound(item.bound),
            color: integer(item.color, 0, 5, "Phẩm chất"),
        } as StarterItem
    })
    const pets = data.pets.map(raw => {
        const pet = object(raw, ["templateId", "quantity", "bound", "growth"])
        if (
            typeof pet.growth !== "number" ||
            !Number.isFinite(pet.growth) ||
            pet.growth <= 0 ||
            pet.growth > 10
        ) {
            throw new Error(
                "Trưởng thành thú cưng phải lớn hơn 0 và không quá 10."
            )
        }
        return {
            templateId: integer(pet.templateId, 1, 2147483647, "Mã thú cưng"),
            quantity: integer(pet.quantity, 1, 6, "Số lượng thú cưng"),
            bound: bound(pet.bound),
            growth: pet.growth,
        }
    })
    if (pets.reduce((sum, pet) => sum + pet.quantity, 0) > 6)
        throw new Error("Gói quà vượt 6 ô thú cưng ban đầu.")
    return {
        version: 1,
        level: integer(data.level, 1, 160, "Cấp khởi tạo"),
        gold: integer(data.gold, 0, Number.MAX_SAFE_INTEGER, "Vàng"),
        goldBind: integer(
            data.goldBind,
            0,
            Number.MAX_SAFE_INTEGER,
            "Kim phiếu"
        ),
        items,
        pets,
    }
}
