"use client"

import { ClassSelector } from "./classes"
import { EnumField, Field } from "./fields"
import { RowsEditor } from "./rows"
import type { QuestFormProps } from "./quest-form"

export function EligibilityPanel({
    quest,
    structuralDisabled,
    edit,
    rows,
    replaceRows,
}: QuestFormProps) {
    return (
        <fieldset disabled={structuralDisabled} className="space-y-5">
            <div className="grid gap-4 md:grid-cols-2">
                <Field
                    label="Cấp tối thiểu"
                    numeric
                    value={quest.min_level}
                    onChange={value => edit("min_level", value)}
                />
                <Field
                    label="Cấp tối đa"
                    numeric
                    value={quest.max_level}
                    onChange={value => edit("max_level", value)}
                />
                <EnumField
                    label="Giới tính"
                    value={quest.gender}
                    options={[
                        [0, "Nam"],
                        [1, "Nữ"],
                        [2, "Không giới hạn"],
                    ]}
                    onChange={value => edit("gender", value)}
                />
                <EnumField
                    label="Điều kiện chuyển sinh"
                    value={quest.is_rebirth}
                    options={[
                        [0, "Không"],
                        [1, "Có"],
                    ]}
                    onChange={value => edit("is_rebirth", value)}
                />
            </div>
            <ClassSelector
                value={quest.req_class}
                onChange={value => edit("req_class", value)}
            />
            <EnumField
                label="Nhiệm vụ tiên quyết"
                value={quest.pre_quest_type}
                options={[
                    [0, "Bỏ qua gate nhiệm vụ"],
                    [1, "AND — hoàn thành tất cả"],
                    [2, "OR — hoàn thành một"],
                ]}
                onChange={value => edit("pre_quest_type", value)}
            />
            <p className="text-sm">
                Vật phẩm tiên quyết luôn phải mang theo, không bị OR của nhiệm
                vụ thay thế và không bị tiêu hao khi kiểm tra.
            </p>
            <RowsEditor
                collection="prerequisites"
                rows={rows.prerequisites}
                onChange={value => replaceRows("prerequisites", value)}
            />
        </fieldset>
    )
}
