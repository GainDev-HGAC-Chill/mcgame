"use client"

import { useRouter } from "next/navigation"
import { FormEvent, useEffect, useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import {
    gatewayConfigService,
    type GatewayLine,
    type GatewayLineInput,
} from "@/services/gateway-config.service"

type Mode = "create" | "edit"

const DEFAULT_RTMP_BASE = "rtmp://127.0.0.1:1935/line"

function buildLineURL(base: string, id: number | undefined): string {
    const trimmed = base.replace(/\/+$/, "")
    const suffix = id === undefined ? "" : String(id)
    return `${trimmed}/${suffix}`
}

function parseBaseFromURL(url: string | undefined): string {
    if (!url) return DEFAULT_RTMP_BASE
    const lastSlash = url.lastIndexOf("/")
    if (lastSlash <= 0) return DEFAULT_RTMP_BASE
    return url.slice(0, lastSlash)
}

function pickNextLineID(lines: GatewayLine[]): number {
    if (lines.length === 0) return 0
    const maxID = lines.reduce((acc, l) => (l.id > acc ? l.id : acc), -1)
    return Math.min(maxID + 1, 32767)
}

export function LineForm({
    mode,
    initial,
}: {
    mode: Mode
    initial?: GatewayLine
}) {
    const router = useRouter()
    const [submitting, setSubmitting] = useState(false)
    const [rtmpBase, setRtmpBase] = useState<string>(
        mode === "edit" ? parseBaseFromURL(initial?.url) : DEFAULT_RTMP_BASE
    )
    const [idLoading, setIdLoading] = useState(mode === "create")

    const [form, setForm] = useState<GatewayLineInput>({
        id: initial?.id,
        name: initial?.name ?? "",
        url: initial?.url ?? buildLineURL(DEFAULT_RTMP_BASE, 0),
        maxClients: initial?.max_clients ?? 1000,
        auction: initial?.auction ?? false,
        guild: initial?.guild ?? false,
        status: initial?.status ?? "online",
        sortOrder: initial?.sort_order ?? 0,
    })

    useEffect(() => {
        if (mode !== "create") return
        let cancelled = false
        ;(async () => {
            try {
                const lines = await gatewayConfigService.list()
                if (cancelled) return
                const nextId = pickNextLineID(lines)
                setForm(f => ({ ...f, id: nextId, url: buildLineURL(rtmpBase, nextId) }))
            } catch (error) {
                if (cancelled) return
                toast.error(error instanceof Error ? error.message : "Không tải được danh sách kênh")
                setForm(f => ({ ...f, id: 0, url: buildLineURL(rtmpBase, 0) }))
            } finally {
                if (!cancelled) setIdLoading(false)
            }
        })()
        return () => {
            cancelled = true
        }
    }, [mode, rtmpBase])

    useEffect(() => {
        if (mode !== "create") return
        setForm(f => ({ ...f, url: buildLineURL(rtmpBase, f.id) }))
    }, [mode, rtmpBase])

    async function handleSubmit(event: FormEvent) {
        event.preventDefault()
        setSubmitting(true)
        try {
            if (mode === "create") {
                if (form.id === undefined) throw new Error("Vui lòng nhập ID")
                const created = await gatewayConfigService.create(form)
                toast.success(`Đã tạo kênh ${created.id}`)
            } else if (initial) {
                await gatewayConfigService.update(initial.id, form)
                toast.success(`Đã cập nhật kênh ${initial.id}`)
            }
            router.push("/dashboard")
            router.refresh()
        } catch (error) {
            toast.error(error instanceof Error ? error.message : "Lưu thất bại")
        } finally {
            setSubmitting(false)
        }
    }

    return (
        <form onSubmit={handleSubmit} className="space-y-6">
            <Card>
                <CardHeader>
                    <CardTitle>{mode === "create" ? "Tạo kênh" : `Chỉnh sửa kênh ${initial?.id}`}</CardTitle>
                </CardHeader>
                <CardContent className="grid gap-4">
                    {mode === "create" ? (
                        <div className="grid gap-2">
                            <Label htmlFor="id">ID (tự tăng)</Label>
                            <Input
                                id="id"
                                type="number"
                                value={idLoading ? "" : form.id ?? ""}
                                readOnly
                                disabled
                                placeholder={idLoading ? "Đang tải…" : ""}
                            />
                            <p className="text-xs text-muted-foreground">
                                ID smallint tiếp theo (ID lớn nhất hiện có + 1).
                            </p>
                        </div>
                    ) : null}

                    <div className="grid gap-2">
                        <Label htmlFor="name">Tên</Label>
                        <Input
                            id="name"
                            value={form.name}
                            onChange={event => setForm(f => ({ ...f, name: event.target.value }))}
                            required
                            maxLength={80}
                        />
                    </div>

                    {mode === "create" ? (
                        <div className="grid gap-2">
                            <Label htmlFor="rtmpBase">Địa chỉ RTMP gốc</Label>
                            <div className="flex items-center gap-2">
                                <Input
                                    id="rtmpBase"
                                    value={rtmpBase}
                                    onChange={event => setRtmpBase(event.target.value)}
                                    required
                                    maxLength={180}
                                    placeholder="rtmp://host:port/line"
                                />
                                <span className="text-sm text-muted-foreground whitespace-nowrap">
                                    /{idLoading ? "…" : form.id ?? ""}
                                </span>
                            </div>
                            <p className="text-xs text-muted-foreground">
                                URL đầy đủ: <code>{form.url}</code>
                            </p>
                        </div>
                    ) : (
                        <div className="grid gap-2">
                            <Label htmlFor="url">URL RTMP</Label>
                            <Input
                                id="url"
                                value={form.url}
                                onChange={event => setForm(f => ({ ...f, url: event.target.value }))}
                                required
                                maxLength={200}
                                placeholder="rtmp://host:port/line/<id>"
                            />
                        </div>
                    )}

                    <div className="grid grid-cols-2 gap-4">
                        <div className="grid gap-2">
                            <Label htmlFor="maxClients">Số người chơi tối đa</Label>
                            <Input
                                id="maxClients"
                                type="number"
                                min={0}
                                max={100000}
                                value={form.maxClients}
                                onChange={event =>
                                    setForm(f => ({ ...f, maxClients: Number(event.target.value || 0) }))
                                }
                            />
                        </div>
                        <div className="grid gap-2">
                            <Label htmlFor="sortOrder">Thứ tự sắp xếp</Label>
                            <Input
                                id="sortOrder"
                                type="number"
                                min={0}
                                max={32767}
                                value={form.sortOrder}
                                onChange={event =>
                                    setForm(f => ({ ...f, sortOrder: Number(event.target.value || 0) }))
                                }
                            />
                        </div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div className="flex items-center justify-between rounded-md border p-3">
                            <Label htmlFor="auction" className="cursor-pointer">
                                Đấu giá
                            </Label>
                            <Switch
                                id="auction"
                                checked={form.auction}
                                onCheckedChange={value => setForm(f => ({ ...f, auction: value }))}
                            />
                        </div>
                        <div className="flex items-center justify-between rounded-md border p-3">
                            <Label htmlFor="guild" className="cursor-pointer">
                                Bang hội
                            </Label>
                            <Switch
                                id="guild"
                                checked={form.guild}
                                onCheckedChange={value => setForm(f => ({ ...f, guild: value }))}
                            />
                        </div>
                    </div>

                    <div className="grid gap-2">
                        <Label htmlFor="status">Trạng thái</Label>
                        <Select
                            value={form.status}
                            onValueChange={value =>
                                setForm(f => ({ ...f, status: value as GatewayLineInput["status"] }))
                            }
                        >
                            <SelectTrigger id="status">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="online">Trực tuyến</SelectItem>
                                <SelectItem value="maintenance">Bảo trì</SelectItem>
                                <SelectItem value="offline">Ngoại tuyến</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>
                </CardContent>
            </Card>

            <div className="flex gap-2">
                <Button type="submit" disabled={submitting || (mode === "create" && idLoading)}>
                    {submitting ? "Đang lưu…" : mode === "create" ? "Tạo" : "Lưu thay đổi"}
                </Button>
                <Button type="button" variant="outline" onClick={() => router.push("/dashboard")}>
                    Hủy
                </Button>
            </div>
        </form>
    )
}
