"use client"

import Link from "next/link"
import { use, useEffect, useMemo, useState } from "react"

import { Button } from "@/components/ui/button"
import { npcRequest, useNpcResource } from "@/hooks/use-npcs"
import {
    buildNpcPatch,
    draftShopSlots,
    type DraftShopSlot,
} from "@/lib/npcs/editor"
import type { NpcDetail, NpcRecord } from "@/types/npcs"
import { NpcSaveDialog } from "../_components/save-dialog"
import { NpcTabs } from "../_components/npc-tabs"

export default function NpcDetailPage({
    params,
}: {
    params: Promise<{ id: string }>
}) {
    const { id } = use(params)
    const resource = useNpcResource<NpcDetail>(
        `/api/npcs/${encodeURIComponent(id)}`
    )
    const [baseline, setBaseline] = useState<NpcDetail | null>(null)
    const [npc, setNpc] = useState<NpcRecord | null>(null)
    const [slots, setSlots] = useState<DraftShopSlot[]>([])
    const [confirm, setConfirm] = useState(false)
    const [reason, setReason] = useState("")
    const [saving, setSaving] = useState(false)
    const [error, setError] = useState<Error | null>(null)
    const [saved, setSaved] = useState(false)
    function initialize(detail: NpcDetail) {
        setBaseline(detail)
        setNpc({ ...detail.npc })
        setSlots(draftShopSlots(detail.shop?.slots ?? []))
    }
    useEffect(() => {
        if (resource.data) {
            initialize(resource.data)
            setSaved(false)
        }
    }, [resource.data])
    const patch = useMemo(
        () => (baseline && npc ? buildNpcPatch(baseline, npc, slots) : {}),
        [baseline, npc, slots]
    )
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
            const sharedCount = baseline.shop?.referencingNpcs.length ?? 0
            const result = await npcRequest<NpcDetail>(
                `/api/npcs/${baseline.npcId}`,
                {
                    method: "PATCH",
                    body: JSON.stringify({
                        expectedRevision: baseline.revision,
                        expectedSharedCount: sharedCount,
                        confirmShared: sharedCount > 1,
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
                    : new Error("Không lưu được NPC")
            )
            setConfirm(false)
        } finally {
            setSaving(false)
        }
    }
    const disabled =
        !baseline?.canWrite || saving || resource.loading || !!resource.error
    return (
        <div className="space-y-6 p-6">
            <Link
                className="text-sm underline"
                href="/dashboard/npcs"
                onClick={event => {
                    if (
                        dirty &&
                        !window.confirm("Có thay đổi chưa lưu. Rời trang?")
                    )
                        event.preventDefault()
                }}
            >
                ← Danh sách NPC
            </Link>
            <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                    <h1 className="text-2xl font-bold">
                        NPC #{id}
                        {npc?.name && ` · ${npc.name}`}
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
                            )
                                resource.reload()
                        }}
                    >
                        Tải lại
                    </Button>
                    <Button
                        disabled={disabled || !dirty}
                        onClick={() => setConfirm(true)}
                    >
                        Lưu NPC
                    </Button>
                </div>
            </div>
            <p
                className={`rounded-lg border p-4 text-sm ${saved ? "border-emerald-500" : "border-amber-500"}`}
            >
                {saved
                    ? "Đã lưu DB và audit. Chưa restart/kích hoạt runtime."
                    : "Thay đổi DB chưa tự có hiệu lực trong game; không tự restart hoặc phát hành danh mục."}
            </p>
            {(error || resource.error) && (
                <p
                    role="alert"
                    className="rounded-md border border-destructive p-3 text-destructive"
                >
                    {(error || resource.error)!.message}
                </p>
            )}
            {resource.loading && <p>Đang tải NPC…</p>}
            {baseline?.warnings?.length ? (
                <section className="space-y-1 rounded-lg border border-amber-500 p-4">
                    {baseline.warnings.map((warning, index) => (
                        <p className="text-sm" key={index}>
                            {warning}
                        </p>
                    ))}
                </section>
            ) : null}
            {baseline && npc && (
                <NpcTabs
                    detail={baseline}
                    npc={npc}
                    setNpc={setNpc}
                    slots={slots}
                    setSlots={setSlots}
                    disabled={disabled}
                />
            )}
            <NpcSaveDialog
                open={confirm}
                onOpenChange={setConfirm}
                reason={reason}
                setReason={setReason}
                saving={saving}
                sharedCount={baseline?.shop?.referencingNpcs.length ?? 0}
                onSave={save}
            />
        </div>
    )
}
