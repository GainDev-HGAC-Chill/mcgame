"use client"

import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { text } from "../_lib/editor"
import { Field, Picker } from "./fields"
import { RowsEditor } from "./rows"
import { LoopEditor } from "./loops"
import { IdentityPanel } from "./identity-panel"
import { EligibilityPanel } from "./eligibility-panel"
import { AwardsPanel } from "./awards-panel"
import type { QuestFormProps } from "./quest-form"

export function QuestTabs(props: QuestFormProps & { authFailure?: boolean }) {
    const {
        quest,
        baseline,
        structuralDisabled,
        edit,
        replaceQuest,
        rows,
        replaceRows,
        selectedLoop,
        setSelectedLoop,
        authFailure = false,
    } = props
    return (
        <Tabs defaultValue="identity" className="space-y-4">
            <TabsList className="h-auto flex-wrap justify-start">
                {[
                    ["identity", "Tên & hội thoại"],
                    ["eligibility", "Điều kiện"],
                    ["requirements", "Mục tiêu"],
                    ["awards", "Phần thưởng"],
                    ["grant", "Vật phẩm khi nhận"],
                    ["loops", "Vòng lặp"],
                    ["advanced", "Nâng cao / chỉ đọc"],
                ].map(([value, label]) => (
                    <TabsTrigger key={value} value={value}>
                        {label}
                    </TabsTrigger>
                ))}
            </TabsList>
            <TabsContent value="identity">
                <IdentityPanel {...props} />
            </TabsContent>
            <TabsContent value="eligibility">
                <EligibilityPanel {...props} />
            </TabsContent>
            <TabsContent value="requirements">
                <fieldset disabled={structuralDisabled}>
                    <RowsEditor
                        collection="requirements"
                        rows={rows.requirements}
                        onChange={value => replaceRows("requirements", value)}
                    />
                </fieldset>
            </TabsContent>
            <TabsContent value="awards">
                <AwardsPanel {...props} />
            </TabsContent>
            <TabsContent value="grant" className="space-y-4">
                <p className="text-sm">
                    Vật phẩm cấp khi nhận nhiệm vụ, không phải thưởng hoàn
                    thành. Chỉnh danh mục không cấp lại cho người đang nhận.
                </p>
                <fieldset
                    disabled={structuralDisabled}
                    className="grid gap-4 md:grid-cols-2"
                >
                    <Picker
                        label="Vật phẩm cấp khi nhận"
                        catalog="item"
                        value={quest.start_grant_item}
                        onChange={value =>
                            edit("start_grant_item", Number(value))
                        }
                    />
                    <Field
                        label="Số lượng cấp khi nhận"
                        numeric
                        value={quest.start_grant_qty}
                        onChange={value => {
                            if (/^[0-9]+$/.test(value))
                                edit("start_grant_qty", Number(value))
                        }}
                    />
                    <Button
                        variant="outline"
                        onClick={() => {
                            if (
                                window.confirm(
                                    "Xóa cấu hình cấp vật phẩm khi nhận (đặt NULL)?"
                                )
                            )
                                replaceQuest({
                                    ...quest,
                                    start_grant_item: null,
                                    start_grant_qty: null,
                                })
                        }}
                    >
                        Xóa cấu hình cấp khi nhận
                    </Button>
                </fieldset>
                <p className="text-xs">
                    NULL / 0 lịch sử giữ nguyên khi không sửa; runtime có thể
                    mặc định số lượng 1.
                </p>
            </TabsContent>
            <TabsContent value="loops" className="space-y-5">
                <fieldset disabled={structuralDisabled} className="space-y-4">
                    <Picker
                        label="Gắn nhiệm vụ vào vòng lặp"
                        catalog="loop"
                        value={selectedLoop}
                        sentinels={[[0, "Giữ cấu hình cũ (không đổi subtype)"]]}
                        onChange={value => {
                            setSelectedLoop(value)
                            if (Number(value) > 0)
                                replaceQuest({
                                    ...quest,
                                    type: "7",
                                    sub_type: `7-${value === "1" ? "10" : value}`,
                                })
                            else
                                replaceQuest({
                                    ...quest,
                                    type: baseline.quest.type,
                                    sub_type: baseline.quest.sub_type,
                                })
                        }}
                    />
                    <p className="text-sm">
                        Subtype đang lưu: {text(quest.sub_type)}. Chọn vòng chỉ
                        sửa type/sub_type của nhiệm vụ; không đổi danh mục vòng.
                    </p>
                </fieldset>
                <LoopEditor
                    linked={baseline.loopRef}
                    canWrite={baseline.canWrite}
                    authFailure={authFailure}
                />
            </TabsContent>
            <TabsContent value="advanced" className="space-y-4">
                <p className="text-sm">
                    Trường lịch sử / script chỉ đọc, không được tự sửa hay chuẩn
                    hóa. rt/rn là yêu cầu tiền tệ, không phải thưởng; chưa có
                    semantics được xác minh.
                </p>
                <div className="grid gap-4 md:grid-cols-2">
                    {[
                        "a_script_text",
                        "qtest",
                        "script_give_up",
                        "rt",
                        "rn",
                        "lm",
                        "sub_type",
                    ].map(field => (
                        <section className="rounded-lg border p-3" key={field}>
                            <h3 className="font-medium">{field}</h3>
                            <pre className="overflow-auto whitespace-pre-wrap text-sm">
                                {quest[field] === null
                                    ? "NULL"
                                    : text(quest[field])}
                            </pre>
                        </section>
                    ))}
                </div>
                <details>
                    <summary>Toàn bộ dữ liệu danh mục (chỉ đọc)</summary>
                    <pre className="overflow-auto whitespace-pre-wrap text-xs">
                        {JSON.stringify(baseline, null, 2)}
                    </pre>
                </details>
            </TabsContent>
        </Tabs>
    )
}
