"use client"

import {
    CharacterFieldForm,
    type CharacterField,
} from "../_components/character-field-form"

const FIELDS: CharacterField[] = [
    { key: "cookDex", label: "Thành thạo nấu ăn", dtoKey: "cookDex" },
    { key: "fishDex", label: "Thành thạo câu cá", dtoKey: "fishDex" },
    { key: "herbDex", label: "Thành thạo hái thuốc", dtoKey: "herbDex" },
    { key: "medicineDex", label: "Thành thạo chế thuốc", dtoKey: "medicineDex" },
    { key: "plantDex", label: "Thành thạo trồng cây", dtoKey: "plantDex" },
]

export function LifeSkillsTab({
    char,
    playerId,
}: {
    char: Record<string, unknown>
    playerId: number
}) {
    return (
        <CharacterFieldForm
            title="Kỹ năng sống"
            action="set_life_skills"
            char={char}
            playerId={playerId}
            fields={FIELDS}
            columns={5}
        />
    )
}
