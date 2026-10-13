"use client"

import {
    CharacterFieldForm,
    type CharacterField,
} from "../_components/character-field-form"

const FIELDS: CharacterField[] = [
    { key: "bagSlotNum", label: "Số trang túi", dtoKey: "bagSlotNum" },
    { key: "bankSlotNum", label: "Số trang kho", dtoKey: "bankSlotNum" },
    { key: "petSlots", label: "Ô thú cưng", dtoKey: "petSlots" },
    { key: "tempBagSlots", label: "Ô túi tạm", dtoKey: "tempBagSlots" },
    {
        key: "mxTempBagSlots",
        label: "Ô túi tạm tối đa",
        dtoKey: "mxTempBagSlots",
    },
]

export function ResourcesTab({
    char,
    playerId,
}: {
    char: Record<string, unknown>
    playerId: number
}) {
    const bagSlots = Number(char.bagSlots ?? char.bagSlotsBase ?? 0)
    const bankSlots = Number(char.bankSlots ?? char.bankSlotsBase ?? 0)

    return (
        <CharacterFieldForm
            title="Sức chứa"
            action="set_resources"
            char={char}
            playerId={playerId}
            fields={FIELDS}
            columns={5}
            description="Túi thường có 1–7 trang; kho có 1–5 trang. Trường để trống được giữ nguyên."
            note={`Mỗi trang túi hoặc kho có 30 ô. Trang túi 8 và 9 vẫn là trang nhiệm vụ và thú cưng mặc định. Tổng hiện tại: túi ${bagSlots}, kho ${bankSlots}.`}
        />
    )
}
