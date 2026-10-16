"use client"

import Link from "next/link"
import { useEffect, useState } from "react"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { QuestApiError, useQuestResource } from "@/hooks/use-quests"
import { text, type Lookup, type Value } from "../_lib/editor"

export function ApiError({ error }: { error: Error }) {
    const status = error instanceof QuestApiError ? error.status : 0
    return (
        <div
            role="alert"
            className="rounded-lg border border-destructive p-4 text-sm"
        >
            {status === 401 ? (
                <>
                    Bạn cần đăng nhập bằng tài khoản đã xác thực.{" "}
                    <Link className="underline" href="/login">
                        Đăng nhập
                    </Link>
                </>
            ) : status === 403 ? (
                "Tài khoản không có quyền xem hoặc chỉnh sửa nhiệm vụ. Liên hệ quản trị viên để được cấp quyền."
            ) : status === 409 ? (
                `Xung đột dữ liệu: ${error.message}. Giữ bản chỉnh sửa để đối chiếu; tải lại trước khi lưu.`
            ) : (
                error.message
            )}
            {error instanceof QuestApiError && error.issues != null && (
                <pre className="mt-2 overflow-auto whitespace-pre-wrap text-xs">
                    {JSON.stringify(error.issues, null, 2)}
                </pre>
            )}
        </div>
    )
}
export function Field({
    label,
    value,
    onChange,
    multiline = false,
    numeric = false,
}: {
    label: string
    value: Value | undefined
    onChange: (value: string) => void
    multiline?: boolean
    numeric?: boolean
}) {
    return (
        <Label className="grid gap-2">
            {label}
            {multiline ? (
                <Textarea
                    value={text(value)}
                    placeholder={
                        value === null
                            ? "NULL (giữ nguyên nếu không sửa)"
                            : undefined
                    }
                    onChange={event => onChange(event.target.value)}
                />
            ) : (
                <Input
                    value={text(value)}
                    inputMode={numeric ? "numeric" : undefined}
                    placeholder={
                        value === null ? "NULL (giữ nguyên)" : undefined
                    }
                    onChange={event => onChange(event.target.value)}
                />
            )}
        </Label>
    )
}
export function EnumField({
    label,
    value,
    options,
    onChange,
}: {
    label: string
    value: Value | undefined
    options: readonly (readonly [string | number, string])[]
    onChange: (value: string) => void
}) {
    const known = options.some(([id]) => String(id) === text(value))
    return (
        <Label className="grid gap-2">
            {label}
            <select
                className="h-10 rounded-md border bg-background px-3"
                value={value === null ? "__null" : text(value)}
                onChange={event => onChange(event.target.value)}
            >
                {!known && (
                    <option value={value === null ? "__null" : text(value)}>
                        Giá trị cũ: {value === null ? "NULL" : text(value)}
                    </option>
                )}
                {options.map(([id, name]) => (
                    <option key={id} value={id}>
                        {name} ({id})
                    </option>
                ))}
            </select>
        </Label>
    )
}
export function Picker({
    label,
    catalog,
    value,
    onChange,
    sentinels = [],
}: {
    label: string
    catalog: string
    value: Value | undefined
    onChange: (id: string) => void
    sentinels?: readonly (readonly [number, string])[]
}) {
    const [search, setSearch] = useState("")
    const [query, setQuery] = useState("")
    useEffect(() => {
        const timer = setTimeout(() => setQuery(search), 250)
        return () => clearTimeout(timer)
    }, [search])
    const resource = useQuestResource<{ items: Lookup[] }>(
        `/api/quests/lookups?kind=${encodeURIComponent(catalog)}&search=${encodeURIComponent(query)}&pageSize=30`
    )
    const selected = useQuestResource<{ items: Lookup[] }>(
        value != null && Number(value) > 0
            ? `/api/quests/lookups?kind=${encodeURIComponent(catalog)}&search=${encodeURIComponent(text(value))}&pageSize=30`
            : null
    )
    const items = [...(resource.data?.items ?? [])]
    const current = selected.data?.items.find(
        item => String(item.id) === text(value)
    )
    if (current && !items.some(item => item.id === current.id))
        items.unshift(current)
    return (
        <div className="grid gap-2">
            <Label>{label}</Label>
            <Input
                aria-label={`Tìm ${label}`}
                placeholder="Tìm tên hoặc ID…"
                value={search}
                onChange={event => setSearch(event.target.value)}
            />
            <select
                aria-label={label}
                className="h-10 max-w-full rounded-md border bg-background px-3"
                value={text(value)}
                onChange={event => onChange(event.target.value)}
            >
                {!items.some(item => String(item.id) === text(value)) &&
                    !sentinels.some(([id]) => String(id) === text(value)) && (
                        <option value={text(value)}>
                            Giữ giá trị cũ:{" "}
                            {value === null ? "NULL" : text(value)} (chưa có
                            tên)
                        </option>
                    )}
                {sentinels.map(([id, name]) => (
                    <option value={id} key={id}>
                        {name} ({id})
                    </option>
                ))}
                {items.map(item => (
                    <option value={item.id} key={item.id}>
                        {item.name || "Không tên"} · #{item.id}
                    </option>
                ))}
            </select>
            {resource.loading && <p className="text-xs">Đang tìm…</p>}
            {resource.error && <ApiError error={resource.error} />}
        </div>
    )
}
