"use client"

import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { questRequest, useQuestResource } from "@/hooks/use-quests"
import {
    changedFields,
    text,
    type LoopDetail,
    type RecordRow,
} from "../_lib/editor"
import { ApiError, EnumField, Field, Picker } from "./fields"
import { SaveDialog } from "./save-dialog"
import { ActivationWarning } from "./activation"

export function LoopEditor({
    linked,
    canWrite,
    authFailure,
}: {
    linked?: LoopDetail | null
    canWrite: boolean
    authFailure: boolean
}) {
    const [selected, setSelected] = useState(
        linked ? String(linked.loop.id) : "0"
    )
    const resource = useQuestResource<LoopDetail>(
        Number(selected) > 0 ? `/api/quests/loops/${selected}` : null
    )
    const [baseline, setBaseline] = useState<LoopDetail | null>(null)
    const [draft, setDraft] = useState<RecordRow | null>(null)
    const [confirm, setConfirm] = useState(false)
    const [reason, setReason] = useState("")
    const [saving, setSaving] = useState(false)
    const [error, setError] = useState<Error | null>(null)
    const [saved, setSaved] = useState(false)
    useEffect(() => {
        if (resource.data) {
            setBaseline(resource.data)
            setDraft({ ...resource.data.loop })
            setSaved(false)
        }
    }, [resource.data])
    const patch = baseline && draft ? changedFields(baseline.loop, draft) : {}
    const dirty = Object.keys(patch).length > 0
    const activeStatusKnown =
        baseline?.activeUse?.activeLoops != null ||
        baseline?.activeLoopCount != null
    const active =
        (baseline?.activeUse?.activeLoops ?? baseline?.activeLoopCount ?? 0) >
            0 || (baseline?.activeUse?.activeQuests ?? 0) > 0
    const authFailed = authFailure
    useEffect(() => {
        const listener = (event: BeforeUnloadEvent) => {
            if (dirty) event.preventDefault()
        }
        window.addEventListener("beforeunload", listener)
        return () => window.removeEventListener("beforeunload", listener)
    }, [dirty])
    async function save() {
        if (
            !baseline ||
            !draft ||
            !dirty ||
            !canWrite ||
            !baseline.canWrite ||
            active ||
            !activeStatusKnown ||
            authFailed
        )
            return
        setSaving(true)
        setError(null)
        try {
            const result = await questRequest<LoopDetail>(
                `/api/quests/loops/${baseline.loop.id}`,
                {
                    method: "PATCH",
                    body: JSON.stringify({
                        expectedRevision: baseline.revision,
                        expectedAffectedCount: baseline.affectedQuestCount,
                        confirmShared: true,
                        reason: reason.trim(),
                        requestId: crypto.randomUUID(),
                        patch,
                    }),
                }
            )
            setBaseline(result)
            setDraft({ ...result.loop })
            setSaved(true)
            setConfirm(false)
            setReason("")
        } catch (failure) {
            setError(
                failure instanceof Error
                    ? failure
                    : new Error("Không lưu được vòng lặp")
            )
            setConfirm(false)
        } finally {
            setSaving(false)
        }
    }
    return (
        <div className="space-y-4">
            <p className="text-sm">
                Vòng lặp là danh mục dùng chung, có revision riêng; không được
                lưu ngầm cùng nhiệm vụ. Vòng #1 ánh xạ pool 10, các vòng khác
                ánh xạ ID riêng.
            </p>
            <fieldset disabled={saving}>
                <Picker
                    label="Chọn vòng lặp để xem / chỉnh riêng"
                    catalog="loop"
                    value={selected}
                    sentinels={[[0, "Chưa chọn"]]}
                    onChange={value => {
                        if (
                            !dirty ||
                            window.confirm("Bỏ thay đổi vòng lặp chưa lưu?")
                        ) {
                            setSelected(value)
                            setBaseline(null)
                            setDraft(null)
                        }
                    }}
                />
            </fieldset>
            {resource.error && <ApiError error={resource.error} />}
            {error && <ApiError error={error} />}
            {resource.loading && <p>Đang tải vòng lặp…</p>}
            {saved && <ActivationWarning saved />}
            {draft && baseline && (
                <>
                    <p className="font-medium">
                        Vòng #{draft.id} · ảnh hưởng{" "}
                        {baseline.affectedQuestCount} nhiệm vụ ·{" "}
                        {dirty ? "Có thay đổi chưa lưu" : "Chưa thay đổi"}
                    </p>
                    {active && (
                        <p role="alert" className="text-sm text-destructive">
                            Vòng đang hoạt động: khóa sửa để bảo vệ tiến độ.
                            Việc hoàn thành / bỏ vòng phải do chủ sở hữu thực
                            hiện theo quy trình được phê duyệt.
                        </p>
                    )}
                    {baseline.warnings?.map((warning, index) => (
                        <p key={index} className="text-sm text-amber-600">
                            {warning}
                        </p>
                    ))}
                    {!activeStatusKnown && (
                        <p role="alert" className="text-sm text-amber-600">
                            API chưa cung cấp số vòng đang hoạt động. Tạm khóa
                            sửa vòng dùng chung để bảo vệ tiến độ; cần xác minh
                            trạng thái sử dụng trước khi mở chỉnh sửa.
                        </p>
                    )}
                    <fieldset
                        disabled={
                            !canWrite ||
                            !baseline.canWrite ||
                            saving ||
                            active ||
                            !activeStatusKnown ||
                            authFailed
                        }
                        className="grid gap-4 md:grid-cols-2"
                    >
                        {[
                            ["name", "Tên vòng"],
                            ["info", "Mô tả"],
                            ["num", "Số vòng / lượt"],
                            ["refresh", "Làm mới (giây)"],
                            ["min_level", "Cấp tối thiểu"],
                            ["max_level", "Cấp tối đa"],
                        ].map(([key, label]) => (
                            <Field
                                key={key}
                                label={label}
                                value={draft[key]}
                                numeric={!["name", "info"].includes(key)}
                                onChange={value =>
                                    setDraft({ ...draft, [key]: value })
                                }
                            />
                        ))}
                        <Picker
                            label="NPC vòng lặp"
                            catalog="npc"
                            value={draft.nid}
                            sentinels={[[-1, "NPC theo nghề"]]}
                            onChange={value =>
                                setDraft({ ...draft, nid: value })
                            }
                        />
                        <EnumField
                            label="Đội nhóm (ngữ nghĩa runtime hạn chế)"
                            value={draft.team}
                            options={[
                                [0, "Không"],
                                [1, "Đội nhóm"],
                                [2, "Chế độ lịch sử / chưa đầy đủ"],
                            ]}
                            onChange={value =>
                                setDraft({ ...draft, team: value })
                            }
                        />
                    </fieldset>
                    <p className="text-xs">
                        Khoảng cấp đảo ngược có thể là vòng bị vô hiệu hóa có
                        chủ ý; không tự sửa. Script không được chỉnh.
                    </p>
                    <details>
                        <summary>Script gốc (chỉ đọc)</summary>
                        <pre className="whitespace-pre-wrap text-xs">
                            {text(draft.script_get)}
                            {"\n"}
                            {text(draft.script_give_up)}
                        </pre>
                    </details>
                    <Button
                        disabled={
                            !dirty ||
                            !canWrite ||
                            !baseline.canWrite ||
                            saving ||
                            active ||
                            !activeStatusKnown ||
                            authFailed
                        }
                        onClick={() => setConfirm(true)}
                    >
                        Lưu riêng vòng lặp
                    </Button>
                </>
            )}
            <SaveDialog
                open={confirm}
                onOpenChange={setConfirm}
                reason={reason}
                setReason={setReason}
                saving={saving}
                onSave={save}
                sharedCount={baseline?.affectedQuestCount}
            />
        </div>
    )
}
