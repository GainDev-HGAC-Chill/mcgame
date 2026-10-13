"use client"

import { questTypes } from "../_lib/editor"
import { EnumField, Field, Picker } from "./fields"
import type { QuestFormProps } from "./quest-form"

export function IdentityPanel({
    quest,
    disabled,
    structuralDisabled,
    edit,
}: QuestFormProps) {
    return (
        <div className="space-y-5">
            <fieldset disabled={disabled} className="grid gap-4 md:grid-cols-2">
                <Field
                    label="Tên nhiệm vụ"
                    value={quest.name}
                    onChange={value => edit("name", value)}
                />
                <Field
                    label="Mô tả"
                    multiline
                    value={quest.info}
                    onChange={value => edit("info", value)}
                />
                <div className="grid gap-4 md:grid-cols-2">
                    <Field
                        label="Hội thoại nhận nhiệm vụ"
                        multiline
                        value={quest.start_text}
                        onChange={value => edit("start_text", value)}
                    />
                    <Field
                        label="Hội thoại hoàn thành"
                        multiline
                        value={quest.complete_text}
                        onChange={value => edit("complete_text", value)}
                    />
                </div>
                <EnumField
                    label="Màu hiển thị nhiệm vụ"
                    value={quest.color}
                    options={[
                        [0, "Trắng"],
                        [1, "Xanh lá"],
                        [2, "Xanh dương"],
                        [3, "Tím"],
                        [4, "Cam"],
                    ]}
                    onChange={value => edit("color", value)}
                />
            </fieldset>
            <fieldset
                disabled={structuralDisabled}
                className="grid gap-4 md:grid-cols-2"
            >
                <EnumField
                    label="Loại nhiệm vụ"
                    value={quest.type}
                    options={questTypes}
                    onChange={value => edit("type", value)}
                />
                <Picker
                    label="NPC nhận"
                    catalog="npc"
                    value={quest.start_npc}
                    sentinels={[
                        [-1, "NPC theo nghề"],
                        [0, "Không NPC / hệ thống"],
                    ]}
                    onChange={value => edit("start_npc", value)}
                />
                <Picker
                    label="NPC hoàn thành"
                    catalog="npc"
                    value={quest.finish_npc}
                    sentinels={[
                        [-1, "NPC theo nghề"],
                        [0, "Hoàn thành hệ thống"],
                    ]}
                    onChange={value => edit("finish_npc", value)}
                />
            </fieldset>
            <p className="text-xs text-muted-foreground">
                Nội dung hiển thị dưới dạng văn bản an toàn; mã HTML / script
                không được thực thi.
            </p>
        </div>
    )
}
