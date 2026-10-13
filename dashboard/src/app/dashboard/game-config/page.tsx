"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { toast } from "sonner"
import { NewCharacterDefaultsEditor } from "@/components/game-config/new-character-defaults"
import { GAME_CONFIG_LABELS } from "@/lib/game-config-labels"
import { NEW_CHARACTER_DEFAULTS_KEY } from "@/lib/new-character-defaults"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
    gameConfigService,
    type GameConfigEntry,
    type GameConfigGroup,
} from "@/services/game-config.service"

const GROUP_LABELS: Record<GameConfigGroup, string> = {
    server: "Điều khiển máy chủ",
    tuning: "Điều chỉnh trò chơi",
    gm: "GM và bảo mật",
}

const TUNING_CATEGORIES = ["rates", "events", "features"] as const
const CATEGORY_LABELS: Record<string, string> = { rates: "Tỷ lệ", events: "Sự kiện", features: "Tính năng", other: "Khác" }
type TuningCategory = (typeof TUNING_CATEGORIES)[number]

const HIDDEN_SERVER_KEYS: ReadonlySet<string> = new Set(["maintenance_mode", "login_enabled", NEW_CHARACTER_DEFAULTS_KEY])

function visibleEntries(group: GameConfigGroup, entries: GameConfigEntry[]): GameConfigEntry[] {
    if (group !== "server") return entries
    return entries.filter(entry => !HIDDEN_SERVER_KEYS.has(entry.key))
}

function detectKind(value: unknown): "bool" | "number" | "string" | "json" {
    if (typeof value === "boolean") return "bool"
    if (typeof value === "number") return "number"
    if (typeof value === "string") return "string"
    return "json"
}

function stringifyValue(value: unknown): string {
    if (typeof value === "string") return value
    return JSON.stringify(value)
}

function parseEditorValue(kind: "bool" | "number" | "string" | "json", raw: string): unknown {
    switch (kind) {
        case "bool":
            return raw === "true"
        case "number": {
            const parsed = Number(raw)
            if (!Number.isFinite(parsed)) throw new Error("Giá trị phải là số")
            return parsed
        }
        case "string":
            return raw
        case "json":
            return JSON.parse(raw)
    }
}

function EntryRow({
    entry,
    group,
    onSaved,
}: {
    entry: GameConfigEntry
    group: GameConfigGroup
    onSaved: (next: GameConfigEntry) => void
}) {
    const kind = detectKind(entry.value)
    const [editor, setEditor] = useState(stringifyValue(entry.value))
    const [busy, setBusy] = useState(false)

    async function save() {
        setBusy(true)
        try {
            const parsed = parseEditorValue(kind, editor)
            const updated = await gameConfigService.upsert(group, {
                key: entry.key,
                value: parsed,
                description: entry.description ?? undefined,
                category: entry.category ?? undefined,
            })
            onSaved(updated)
            toast.success(`Đã cập nhật ${entry.key}`)
        } catch (error) {
            toast.error(error instanceof Error ? error.message : "Lưu thất bại")
        } finally {
            setBusy(false)
        }
    }

    return (
        <div className="grid grid-cols-12 gap-3 rounded-md border p-3">
            <div className="col-span-3">
                <div className="text-sm font-medium">{GAME_CONFIG_LABELS[entry.key]?.name ?? entry.key}</div>
                <div className="font-mono text-xs text-muted-foreground">{entry.key}</div>
                {entry.description ? (
                    <div className="text-xs text-muted-foreground">{GAME_CONFIG_LABELS[entry.key]?.description ?? entry.description}</div>
                ) : null}
                {entry.category ? (
                    <div className="mt-1 text-xs uppercase tracking-wide text-muted-foreground">
                        {CATEGORY_LABELS[entry.category] ?? entry.category}
                    </div>
                ) : null}
            </div>
            <div className="col-span-7">
                {kind === "bool" ? (
                    <div className="flex items-center gap-2">
                        <Switch
                            aria-label={`Bật hoặc tắt ${entry.key}`}
                            checked={editor === "true"}
                            onCheckedChange={value => setEditor(value ? "true" : "false")}
                        />
                        <span className="text-sm text-muted-foreground">{editor === "true" ? "Bật" : "Tắt"}</span>
                    </div>
                ) : kind === "json" ? (
                    <Input
                        value={editor}
                        onChange={event => setEditor(event.target.value)}
                        className="font-mono text-xs"
                    />
                ) : (
                    <Input
                        value={editor}
                        onChange={event => setEditor(event.target.value)}
                        type={kind === "number" ? "number" : "text"}
                    />
                )}
            </div>
            <div className="col-span-2 text-right">
                <Button size="sm" onClick={save} disabled={busy}>
                    {busy ? "Đang lưu…" : "Lưu"}
                </Button>
            </div>
        </div>
    )
}

function NewEntryForm({
    group,
    onCreated,
}: {
    group: GameConfigGroup
    onCreated: (next: GameConfigEntry) => void
}) {
    const [key, setKey] = useState("")
    const [value, setValue] = useState("")
    const [description, setDescription] = useState("")
    const [category, setCategory] = useState<TuningCategory>("rates")
    const [busy, setBusy] = useState(false)

    async function submit() {
        if (key.trim() === NEW_CHARACTER_DEFAULTS_KEY) {
            toast.error("Hãy dùng biểu mẫu Quà khởi tạo nhân vật để chỉnh sửa cấu hình này")
            return
        }
        if (!key.trim()) {
            toast.error("Vui lòng nhập khóa cấu hình")
            return
        }
        if (value.trim() === "") {
            toast.error("Vui lòng nhập giá trị JSON: true / 1.5 / \"văn bản\"")
            return
        }
        setBusy(true)
        try {
            const parsedValue = JSON.parse(value)
            const created = await gameConfigService.upsert(group, {
                key: key.trim(),
                value: parsedValue,
                description: description.trim() || undefined,
                category: group === "tuning" ? category : undefined,
            })
            onCreated(created)
            setKey("")
            setValue("")
            setDescription("")
            toast.success("Đã thêm khóa cấu hình")
        } catch (error) {
            toast.error(error instanceof Error ? error.message : "Tạo thất bại")
        } finally {
            setBusy(false)
        }
    }

    return (
        <div className="grid gap-3 rounded-md border bg-muted/30 p-4">
            <div className="grid grid-cols-12 gap-3">
                <div className="col-span-3">
                    <Label className="text-xs">Khóa cấu hình</Label>
                    <Input
                        value={key}
                        onChange={event => setKey(event.target.value)}
                        placeholder="example_setting"
                    />
                </div>
                <div className="col-span-5">
                    <Label className="text-xs">Giá trị (JSON)</Label>
                    <Input
                        value={value}
                        onChange={event => setValue(event.target.value)}
                        placeholder='true / 1.5 / "text" / {"k":1}'
                        className="font-mono text-xs"
                    />
                </div>
                <div className="col-span-3">
                    <Label className="text-xs">Mô tả</Label>
                    <Input value={description} onChange={event => setDescription(event.target.value)} />
                </div>
                <div className="col-span-1 flex items-end">
                    <Button size="sm" onClick={submit} disabled={busy}>
                        Thêm
                    </Button>
                </div>
            </div>
            {group === "tuning" ? (
                <div className="grid grid-cols-12 gap-3">
                    <div className="col-span-3">
                        <Label className="text-xs">Nhóm</Label>
                        <Select value={category} onValueChange={value => setCategory(value as TuningCategory)}>
                            <SelectTrigger>
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                {TUNING_CATEGORIES.map(c => (
                                    <SelectItem key={c} value={c}>
                                        {CATEGORY_LABELS[c]}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>
                </div>
            ) : null}
        </div>
    )
}

function GroupTab({ group }: { group: GameConfigGroup }) {
    const [entries, setEntries] = useState<GameConfigEntry[]>([])
    const [loading, setLoading] = useState(true)

    const refresh = useCallback(async () => {
        setLoading(true)
        try {
            const fetched = await gameConfigService.list(group)
            setEntries(visibleEntries(group, fetched))
        } catch (error) {
            toast.error(error instanceof Error ? error.message : "Không tải được dữ liệu")
        } finally {
            setLoading(false)
        }
    }, [group])

    useEffect(() => {
        void refresh()
    }, [refresh])

    const grouped = useMemo(() => {
        if (group !== "tuning") return { all: entries }
        const buckets: Record<string, GameConfigEntry[]> = {}
        for (const entry of entries) {
            const cat = entry.category ?? "other"
            buckets[cat] = buckets[cat] ?? []
            buckets[cat].push(entry)
        }
        return buckets
    }, [entries, group])

    function handleSaved(next: GameConfigEntry) {
        if (group === "server" && HIDDEN_SERVER_KEYS.has(next.key)) return
        setEntries(prev => {
            const exists = prev.some(e => e.key === next.key)
            return exists ? prev.map(e => (e.key === next.key ? next : e)) : [...prev, next]
        })
    }

    return (
        <div className="space-y-4">
            <NewEntryForm group={group} onCreated={handleSaved} />

            {loading ? (
                <p className="text-sm text-muted-foreground">Đang tải…</p>
            ) : entries.length === 0 ? (
                <p className="text-sm text-muted-foreground">Chưa có khóa cấu hình.</p>
            ) : (
                <div className="space-y-4">
                    {Object.entries(grouped).map(([bucket, items]) => (
                        <div key={bucket}>
                            {group === "tuning" ? (
                                <h3 className="mb-2 text-xs uppercase tracking-wide text-muted-foreground">
                                    {CATEGORY_LABELS[bucket] ?? bucket}
                                </h3>
                            ) : null}
                            <div className="space-y-2">
                                {items.map(entry => (
                                    <EntryRow
                                        key={entry.key}
                                        entry={entry}
                                        group={group}
                                        onSaved={handleSaved}
                                    />
                                ))}
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    )
}

export default function GameConfigPage() {
    return (
        <div className="space-y-6 p-6">
            <div>
                <h1 className="text-2xl font-bold">Cấu hình trò chơi</h1>
                <p className="text-sm text-muted-foreground">
                    Điều khiển máy chủ, điều chỉnh trò chơi và cấu hình GM/bảo mật. Mỗi lần lưu sẽ thông báo
                    đến máy chủ Go đang chạy qua Postgres NOTIFY.
                </p>
            </div>

            <NewCharacterDefaultsEditor />
            <Card>
                <CardHeader>
                    <CardTitle>Cấu hình vận hành</CardTitle>
                </CardHeader>
                <CardContent>
                    <Tabs defaultValue="server" className="space-y-4">
                        <TabsList>
                            <TabsTrigger value="server">{GROUP_LABELS.server}</TabsTrigger>
                            <TabsTrigger value="tuning">{GROUP_LABELS.tuning}</TabsTrigger>
                            <TabsTrigger value="gm">{GROUP_LABELS.gm}</TabsTrigger>
                        </TabsList>
                        <TabsContent value="server">
                            <GroupTab group="server" />
                        </TabsContent>
                        <TabsContent value="tuning">
                            <GroupTab group="tuning" />
                        </TabsContent>
                        <TabsContent value="gm">
                            <GroupTab group="gm" />
                        </TabsContent>
                    </Tabs>
                </CardContent>
            </Card>
        </div>
    )
}
