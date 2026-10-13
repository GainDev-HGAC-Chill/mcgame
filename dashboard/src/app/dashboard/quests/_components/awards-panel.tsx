"use client"

import { text } from "../_lib/editor"
import { EnumField, Field } from "./fields"
import { RowsEditor } from "./rows"
import type { QuestFormProps } from "./quest-form"

export function AwardsPanel({
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
                    label="EXP — award_exp_re"
                    numeric
                    value={quest.award_exp_re}
                    onChange={value => edit("award_exp_re", value)}
                />
                <EnumField
                    label="Tiền thưởng"
                    value={quest.money_type}
                    options={[
                        [-1, "Không"],
                        [0, "Không"],
                        [1, "Bạc khóa"],
                        [2, "Bạc"],
                        [3, "Vàng khóa"],
                        [4, "Vàng"],
                    ]}
                    onChange={value => edit("money_type", value)}
                />
                <Field
                    label="Số tiền"
                    numeric
                    value={quest.money_num}
                    onChange={value => edit("money_num", value)}
                />
                <EnumField
                    label="Cách cấp thưởng"
                    value={quest.at}
                    options={[[1, "Tất cả (được hỗ trợ)"]]}
                    onChange={value => edit("at", value)}
                />
            </div>
            {text(quest.at) !== "1" && (
                <p className="text-sm text-amber-600">
                    Chế độ thưởng lịch sử {text(quest.at)} được giữ nguyên;
                    runtime không hỗ trợ chọn một / lọc theo nghề. Không coi đây
                    là lựa chọn thưởng hoạt động.
                </p>
            )}
            <p className="text-sm">
                Chỉ hỗ trợ vật phẩm, trang bị và thú. Tất cả phần thưởng luôn
                khóa; không hỗ trợ thưởng kỹ năng hoặc lựa chọn thưởng. Kiểm tra
                sức chứa kho và túi thú.
            </p>
            <RowsEditor
                collection="awards"
                rows={rows.awards}
                onChange={value => replaceRows("awards", value)}
            />
        </fieldset>
    )
}
