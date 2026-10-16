"use client"

import { use, useEffect, useState } from "react"
import Link from "next/link"
import { Button } from "@/components/ui/button"
import {
    QuestApiError,
    questRequest,
    useQuestResource,
} from "@/hooks/use-quests"
import {
    aggregatePatch,
    draftRows,
    text,
    type Collection,
    type Detail,
    type DraftRow,
    type RecordRow,
    type Value,
} from "../_lib/editor"
import { ApiError } from "../_components/fields"
import { QuestTabs } from "../_components/quest-tabs"
import { ActivationWarning } from "../_components/activation"
import { SaveDialog } from "../_components/save-dialog"

export default function QuestDetailPage({
    params,
}: {
    params: Promise<{ id: string }>
}) {
    const { id } = use(params)
    const resource = useQuestResource<Detail>(
        `/api/quests/${encodeURIComponent(id)}`
    )
    const [baseline, setBaseline] = useState<Detail | null>(null)
    const [quest, setQuest] = useState<RecordRow | null>(null)
    const [rows, setRows] = useState<Record<Collection, DraftRow[]>>({
        requirements: [],
        awards: [],
        prerequisites: [],
    })
    const [confirm, setConfirm] = useState(false)
    const [reason, setReason] = useState("")
    const [saving, setSaving] = useState(false)
    const [error, setError] = useState<Error | null>(null)
    const [saved, setSaved] = useState(false)
    const [selectedLoop, setSelectedLoop] = useState("0")
    function initialize(detail: Detail) {
        setBaseline(detail)
        setQuest({ ...detail.quest })
        setSelectedLoop(String(detail.loopRef?.loop.id ?? "0"))
        setRows({
            requirements: draftRows(detail.requirements),
            awards: draftRows(detail.awards),
            prerequisites: draftRows(detail.prerequisites),
        })
    }
    useEffect(() => {
        if (resource.data) {
            initialize(resource.data)
            setSaved(false)
        }
    }, [resource.data])
    const patch = baseline && quest ? aggregatePatch(baseline, quest, rows) : {}
    const dirty = Object.keys(patch).length > 0
    useEffect(() => {
        const listener = (event: BeforeUnloadEvent) => {
            if (dirty) event.preventDefault()
        }
        window.addEventListener("beforeunload", listener)
        return () => window.removeEventListener("beforeunload", listener)
    }, [dirty])
    async function save() {
        if (!baseline || !dirty || !baseline.canWrite) return
        setSaving(true)
        setError(null)
        try {
            const result = await questRequest<Detail>(
                `/api/quests/${baseline.questId}`,
                {
                    method: "PATCH",
                    body: JSON.stringify({
                        expectedRevision: baseline.revision,
                        reason: reason.trim(),
                        requestId: crypto.randomUUID(),
                        patch,
                    }),
                }
            )
            initialize(result)
            setSaved(true)
            setConfirm(false)
            setReason("")
        } catch (failure) {
            setError(
                failure instanceof Error
                    ? failure
                    : new Error("Không lưu được nhiệm vụ")
            )
            setConfirm(false)
        } finally {
            setSaving(false)
        }
    }
    function edit(field: string, value: Value) {
        if (quest) setQuest({ ...quest, [field]: value })
    }
    const authFailure =
        error instanceof QuestApiError && [401, 403].includes(error.status)
    const disabled =
        !baseline?.canWrite ||
        saving ||
        authFailure ||
        resource.loading ||
        !!resource.error
    const structuralDisabled = disabled
    return (
        <div className="space-y-6 p-6">
            <Link
                className="text-sm underline"
                href="/dashboard/quests"
                onClick={event => {
                    if (
                        dirty &&
                        !window.confirm("Có thay đổi chưa lưu. Rời trang?")
                    )
                        event.preventDefault()
                }}
            >
                ← Danh sách nhiệm vụ
            </Link>
            <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                    <h1 className="text-2xl font-bold">
                        Nhiệm vụ #{id}
                        {quest && ` · ${text(quest.name)}`}
                    </h1>
                    <p className="text-sm text-muted-foreground">
                        {dirty ? "Có thay đổi chưa lưu" : "Chưa thay đổi"}
                        {baseline?.canWrite === false && " · Chỉ đọc"}
                    </p>
                </div>
                <div className="flex gap-2">
                    <Button
                        variant="outline"
                        disabled={saving}
                        onClick={() => {
                            if (
                                !dirty ||
                                window.confirm(
                                    "Bỏ thay đổi chưa lưu và tải lại?"
                                )
                            ) {
                                setError(null)
                                resource.reload()
                            }
                        }}
                    >
                        Tải lại
                    </Button>
                    <Button
                        disabled={disabled || !dirty}
                        onClick={() => setConfirm(true)}
                    >
                        Lưu nhiệm vụ
                    </Button>
                </div>
            </div>
            <ActivationWarning saved={saved} />
            {(error || resource.error) && (
                <ApiError error={(error || resource.error)!} />
            )}
            {resource.loading && <p>Đang tải nhiệm vụ…</p>}
            {quest && baseline && (
                <>
                    {baseline.canWrite === false && (
                        <p className="rounded-lg border p-3 text-sm">
                            Bạn có quyền xem. Các trường và nút lưu đã khóa;
                            không gửi thay đổi từ tài khoản chỉ đọc.
                        </p>
                    )}
                    {baseline.activeUse && (
                        <p className="text-sm">
                            Đang nhận: {baseline.activeUse.activeQuests} · Lịch
                            sử hoàn thành: {baseline.activeUse.completedHistory}{" "}
                            · Vòng đang hoạt động:{" "}
                            {baseline.activeUse.activeLoops}
                        </p>
                    )}
                    {(baseline.activeUse?.activeQuests ?? 0) > 0 && (
                        <div
                            role="alert"
                            className="rounded-lg border border-amber-500 p-4 text-sm"
                        >
                            Đang có người chơi nhận nhiệm vụ. Bạn vẫn có thể sửa
                            toàn bộ cấu hình; thay đổi được áp dụng khi runtime
                            đối chiếu lại nhiệm vụ. Dashboard không tự bỏ, đặt
                            lại hoặc bồi thường tiến độ hiện có.
                        </div>
                    )}
                    {baseline.warnings?.length > 0 && (
                        <section className="space-y-1 rounded-lg border border-amber-500 p-4">
                            {baseline.warnings.map((warning, index) => (
                                <p className="text-sm" key={index}>
                                    {warning}
                                </p>
                            ))}
                        </section>
                    )}
                    <QuestTabs
                        quest={quest}
                        baseline={baseline}
                        rows={rows}
                        edit={edit}
                        replaceQuest={setQuest}
                        replaceRows={(collection, value) =>
                            setRows({ ...rows, [collection]: value })
                        }
                        disabled={disabled}
                        structuralDisabled={structuralDisabled}
                        selectedLoop={selectedLoop}
                        setSelectedLoop={setSelectedLoop}
                        authFailure={authFailure}
                    />
                </>
            )}
            <SaveDialog
                open={confirm}
                onOpenChange={setConfirm}
                reason={reason}
                setReason={setReason}
                saving={saving}
                onSave={save}
            />
        </div>
    )
}
