"use client"

import {
    CharacterFieldForm,
    type CharacterField,
} from "../_components/character-field-form"

const BASE_FIELDS: CharacterField[] = [
    { key: "strength", label: "Sức mạnh", dtoKey: "attStrength" },
    { key: "agility", label: "Nhanh nhẹn", dtoKey: "attAgility" },
    { key: "stamina", label: "Thể lực", dtoKey: "attStamina" },
    {
        key: "intelligence",
        label: "Trí lực",
        dtoKey: "attIntelligence",
    },
    { key: "spirit", label: "Tinh thần", dtoKey: "attEnergy" },
    { key: "attrPoints", label: "Điểm thuộc tính", dtoKey: "attLastPoint" },
    { key: "hp", label: "HP hiện tại", dtoKey: "currentHp" },
    { key: "mp", label: "MP hiện tại", dtoKey: "currentMp" },
    { key: "sp", label: "SP hiện tại", dtoKey: "currentSp" },
]

const APTITUDE_FIELDS: CharacterField[] = [
    { key: "aptStrength", label: "Tư chất sức mạnh", dtoKey: "aptStrength" },
    {
        key: "aptStrengthEvolution",
        label: "Tư chất sức mạnh tiến hóa",
        dtoKey: "aptStrengthEvolution",
    },
    { key: "aptAgility", label: "Tư chất nhanh nhẹn", dtoKey: "aptAgility" },
    {
        key: "aptAgilityEvolution",
        label: "Tư chất nhanh nhẹn tiến hóa",
        dtoKey: "aptAgilityEvolution",
    },
    { key: "aptStamina", label: "Tư chất thể lực", dtoKey: "aptStamina" },
    {
        key: "aptStaminaEvolution",
        label: "Tư chất thể lực tiến hóa",
        dtoKey: "aptStaminaEvolution",
    },
    {
        key: "aptIntelligence",
        label: "Tư chất trí lực",
        dtoKey: "aptIntelligence",
    },
    {
        key: "aptIntelligenceEvolution",
        label: "Tư chất trí lực tiến hóa",
        dtoKey: "aptIntelligenceEvolution",
    },
    { key: "aptEnergy", label: "Tư chất tinh thần", dtoKey: "aptEnergy" },
    {
        key: "aptEnergyEvolution",
        label: "Tư chất tinh thần tiến hóa",
        dtoKey: "aptEnergyEvolution",
    },
]

export function AttributesTab({
    char,
    playerId,
}: {
    char: Record<string, unknown>
    playerId: number
}) {
    return (
        <div className="space-y-6">
            <CharacterFieldForm
                title="Thuộc tính"
                action="set_attributes"
                char={char}
                playerId={playerId}
                fields={BASE_FIELDS}
                columns={5}
                note="Chỉ số được tự động tính lại sau khi lưu."
            />
            <CharacterFieldForm
                title="Tư chất"
                action="set_attributes"
                char={char}
                playerId={playerId}
                fields={APTITUDE_FIELDS}
                columns={5}
            />
        </div>
    )
}
